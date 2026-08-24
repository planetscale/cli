package sqlquery

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"slices"
	"strings"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/passwordutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/proxyutil"
	"github.com/planetscale/cli/internal/roleutil"
	"vitess.io/vitess/go/mysql"
)

// Options selects the branch and SQL to execute.
type Options struct {
	Organization string
	Database     string
	Branch       string
	Query        string
	// Keyspace is the Vitess keyspace for MySQL. Optional; defaults to @primary (same as pscale shell).
	Keyspace string
	// PostgresDB is the PostgreSQL database name. Defaults to postgres.
	PostgresDB string
	// PostgresAdditionalRoles adds built-in roles to the ephemeral credential.
	PostgresAdditionalRoles []string
	// Role is reader, writer, readwriter, or admin (same as pscale shell --role).
	Role string
	// Replica routes reads to replicas when true (same as pscale shell --replica).
	Replica bool
	// Force allows destructive SQL (DELETE, DROP, TRUNCATE) after explicit user approval.
	Force bool
}

// Result is returned for `pscale sql --format json`.
type Result struct {
	Status       string           `json:"status"`
	Database     string           `json:"database"`
	Branch       string           `json:"branch"`
	Kind         string           `json:"kind"`
	Role         string           `json:"role,omitempty"`
	Replica      bool             `json:"replica,omitempty"`
	RowCount     int              `json:"row_count"`
	RowsAffected int64            `json:"rows_affected,omitempty"`
	Columns      []string         `json:"columns,omitempty"`
	Rows         []map[string]any `json:"rows,omitempty"`
	NextSteps    []string         `json:"next_steps,omitempty"`
}

type queryOutcome struct {
	columns      []string
	rows         []map[string]any
	rowsAffected int64
}

// Execute runs SQL against MySQL or PostgreSQL using ephemeral credentials.
func Execute(ctx context.Context, ch *cmdutil.Helper, opts Options) (*Result, error) {
	if opts.Query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if !opts.Force && IsDestructiveQuery(opts.Query) {
		return nil, &DestructiveQueryError{}
	}
	if opts.Organization == "" {
		return nil, fmt.Errorf("organization is required (use --org or set org in pscale.yml)")
	}
	if opts.Database == "" || opts.Branch == "" {
		return nil, fmt.Errorf("database and branch are required")
	}

	role, err := cmdutil.ResolveAccessRole(opts.Role, opts.Replica, cmdutil.ReaderRole)
	if err != nil {
		return nil, err
	}

	client, err := ch.Client()
	if err != nil {
		return nil, err
	}

	dbInfo, err := client.Databases.Get(ctx, &ps.GetDatabaseRequest{
		Organization: opts.Organization,
		Database:     opts.Database,
	})
	if err != nil {
		return nil, fmt.Errorf("database lookup: %w", err)
	}

	dbBranch, err := client.DatabaseBranches.Get(ctx, &ps.GetDatabaseBranchRequest{
		Organization: opts.Organization,
		Database:     opts.Database,
		Branch:       opts.Branch,
	})
	if err != nil {
		return nil, fmt.Errorf("branch lookup: %w", err)
	}
	if !dbBranch.Ready {
		return nil, fmt.Errorf("database branch is not ready yet")
	}

	result := &Result{
		Status:   "ok",
		Database: opts.Database,
		Branch:   opts.Branch,
		Kind:     string(dbInfo.Kind),
		Role:     role.ToString(),
		Replica:  opts.Replica,
	}

	var outcome *queryOutcome

	switch string(dbInfo.Kind) {
	case "mysql":
		outcome, err = queryMySQL(ctx, ch, opts, role)
	case "postgresql", "horizon":
		pgDB := opts.PostgresDB
		if pgDB == "" {
			pgDB = "postgres"
		}
		outcome, err = queryPostgres(ctx, ch, opts, pgDB, role)
	default:
		return nil, fmt.Errorf("unsupported database kind %q", dbInfo.Kind)
	}
	if err != nil {
		return nil, err
	}

	result.Columns = outcome.columns
	result.Rows = outcome.rows
	result.RowCount = len(outcome.rows)
	result.RowsAffected = outcome.rowsAffected
	result.NextSteps = []string{
		cmdutil.AgentSQLCmd(opts.Organization, opts.Database, opts.Branch, false),
	}
	return result, nil
}

func queryMySQL(ctx context.Context, ch *cmdutil.Helper, opts Options, role cmdutil.PasswordRole) (*queryOutcome, error) {
	db, cleanup, err := openMySQL(ctx, ch, opts, role)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	return runQuery(ctx, db, opts.Query)
}

// openMySQL mints an ephemeral branch password, starts an in-process proxy,
// and opens a connection through it. The returned cleanup closes the
// connection and proxy and deletes the password.
func openMySQL(ctx context.Context, ch *cmdutil.Helper, opts Options, role cmdutil.PasswordRole) (*sql.DB, func(), error) {
	client, err := ch.Client()
	if err != nil {
		return nil, nil, err
	}

	pw, err := passwordutil.New(ctx, client, passwordutil.Options{
		Organization: opts.Organization,
		Database:     opts.Database,
		Branch:       opts.Branch,
		Role:         role,
		Name:         passwordutil.GenerateName("pscale-cli-sql"),
		TTL:          5 * time.Minute,
		Replica:      opts.Replica,
	})
	if err != nil {
		return nil, nil, err
	}
	cleanupPassword := func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = pw.Cleanup(cleanupCtx)
	}

	proxy := proxyutil.New(proxyutil.Config{
		Logger:       cmdutil.NewZapLogger(ch.Debug()),
		UpstreamAddr: pw.Password.Hostname,
		Username:     pw.Password.Username,
		Password:     pw.Password.PlainText,
	})

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		proxy.Close()
		cleanupPassword()
		return nil, nil, err
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- proxy.Serve(l, mysql.CachingSha2Password)
	}()

	// Config.FormatDSN escapes shard targets such as keyspace/-80@replica.
	dsn := mysqlDSN(l.Addr().String(), opts)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		proxy.Close()
		l.Close()
		<-errCh
		cleanupPassword()
		return nil, nil, err
	}
	db.SetConnMaxLifetime(30 * time.Second)

	cleanup := func() {
		db.Close()
		proxy.Close()
		l.Close()
		<-errCh
		cleanupPassword()
	}
	return db, cleanup, nil
}

func mysqlDSN(addr string, opts Options) string {
	dsnCfg := gomysql.NewConfig()
	dsnCfg.User = "root"
	dsnCfg.Net = "tcp"
	dsnCfg.Addr = addr
	dsnCfg.DBName = mysqlDSNDatabase(opts)
	return dsnCfg.FormatDSN()
}

// mysqlDSNDatabase picks the MySQL database name in the DSN, matching pscale shell:
// default @primary for the main keyspace, no default when --replica is set, or an explicit --keyspace.
func mysqlDSNDatabase(opts Options) string {
	if opts.Keyspace != "" {
		return opts.Keyspace
	}
	if opts.Replica {
		return ""
	}
	return "@primary"
}

func queryPostgres(ctx context.Context, ch *cmdutil.Helper, opts Options, pgDB string, role cmdutil.PasswordRole) (*queryOutcome, error) {
	db, cleanup, err := openPostgres(ctx, ch, opts, pgDB, role)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	return runQuery(ctx, db, opts.Query)
}

// openPostgres mints an ephemeral role and opens a direct connection to the
// branch. The returned cleanup closes the connection and deletes the role.
func openPostgres(ctx context.Context, ch *cmdutil.Helper, opts Options, pgDB string, role cmdutil.PasswordRole) (*sql.DB, func(), error) {
	client, err := ch.Client()
	if err != nil {
		return nil, nil, err
	}

	inheritedRoles, successor := cmdutil.PostgresInheritedRoles(role)
	inheritedRoles = append(inheritedRoles, opts.PostgresAdditionalRoles...)

	pgRole, err := roleutil.New(ctx, client, roleutil.Options{
		Organization:   opts.Organization,
		Database:       opts.Database,
		Branch:         opts.Branch,
		Name:           passwordutil.GenerateName("pscale-cli-sql"),
		TTL:            5 * time.Minute,
		InheritedRoles: inheritedRoles,
	})
	if err != nil {
		return nil, nil, err
	}
	cleanupRole := func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = pgRole.Cleanup(cleanupCtx, successor)
	}

	username := pgRole.Role.Username
	if opts.Replica {
		username = username + "|replica"
	}

	remoteHost, remotePort, err := net.SplitHostPort(pgRole.Role.AccessHostURL)
	if err != nil {
		remoteHost = pgRole.Role.AccessHostURL
		remotePort = "5432"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=verify-full",
		remoteHost, remotePort, username, pgRole.Role.Password, pgDB)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		cleanupRole()
		return nil, nil, err
	}
	db.SetConnMaxLifetime(30 * time.Second)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		cleanupRole()
		return nil, nil, err
	}

	cleanup := func() {
		db.Close()
		cleanupRole()
	}
	return db, cleanup, nil
}

var readQueryPrefixes = []string{"SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN", "TABLE"}

func isReadQuery(query string) bool {
	q := strings.TrimSpace(stripSQLGuardIgnoredText(query))
	upper := strings.ToUpper(q)
	if hasKeywordPrefix(upper, "WITH") {
		afterCTEs, ok := queryAfterCTEs(q)
		if !ok {
			// Preserve the old behavior for unusual CTE syntax we cannot parse.
			return true
		}
		return isReadQuery(afterCTEs)
	}
	return slices.ContainsFunc(readQueryPrefixes, func(prefix string) bool {
		return hasKeywordPrefix(upper, prefix)
	})
}

func queryReturnsRows(query string) bool {
	upper := strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(strings.ToUpper(stripSQLGuardIgnoredText(query)))
	return containsWord(upper, "RETURNING")
}

func queryAfterCTEs(query string) (string, bool) {
	_, after, ok := walkCTEs(query)
	return after, ok
}

func walkCTEs(query string) (bodies []string, after string, ok bool) {
	rest := strings.TrimSpace(query)
	if !hasKeywordPrefix(strings.ToUpper(rest), "WITH") {
		return nil, rest, true
	}
	rest = strings.TrimSpace(rest[len("WITH"):])
	if hasKeywordPrefix(strings.ToUpper(rest), "RECURSIVE") {
		rest = strings.TrimSpace(rest[len("RECURSIVE"):])
	}

	for {
		asIdx := topLevelKeywordIndex(rest, "AS")
		if asIdx < 0 {
			return nil, "", false
		}
		rest = strings.TrimSpace(rest[asIdx+len("AS"):])
		rest = skipCTEMaterializedModifier(rest)
		if !strings.HasPrefix(rest, "(") {
			return nil, "", false
		}
		end := matchingParenIndex(rest)
		if end < 0 {
			return nil, "", false
		}
		bodies = append(bodies, rest[1:end])
		rest = strings.TrimSpace(rest[end+1:])
		if strings.HasPrefix(rest, ",") {
			rest = strings.TrimSpace(rest[1:])
			continue
		}
		return bodies, rest, true
	}
}

func skipCTEMaterializedModifier(rest string) string {
	trimmed := strings.TrimSpace(rest)
	upper := strings.ToUpper(trimmed)
	if hasKeywordPrefix(upper, "NOT") {
		trimmed = strings.TrimSpace(trimmed[len("NOT"):])
		upper = strings.ToUpper(trimmed)
		if hasKeywordPrefix(upper, "MATERIALIZED") {
			return strings.TrimSpace(trimmed[len("MATERIALIZED"):])
		}
	}
	if hasKeywordPrefix(upper, "MATERIALIZED") {
		return strings.TrimSpace(trimmed[len("MATERIALIZED"):])
	}
	return trimmed
}

func topLevelKeywordIndex(s, keyword string) int {
	upper := strings.ToUpper(s)
	depth := 0
	for i := 0; i < len(upper); i++ {
		switch upper[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 && (i == 0 || !isIdentifierChar(upper[i-1])) && hasKeywordPrefix(upper[i:], keyword) {
				return i
			}
		}
	}
	return -1
}

func matchingParenIndex(s string) int {
	depth := 0
	quote := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == quote {
				if i+1 < len(s) && s[i+1] == quote {
					i++
					continue
				}
				quote = 0
			}
			continue
		}
		switch c {
		case '\'', '"', '`':
			quote = c
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func hasKeywordPrefix(s, keyword string) bool {
	if !strings.HasPrefix(s, keyword) {
		return false
	}
	if len(s) == len(keyword) {
		return true
	}
	return !isIdentifierChar(s[len(keyword)])
}

func isIdentifierChar(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func runQuery(ctx context.Context, db *sql.DB, query string) (*queryOutcome, error) {
	if isReadQuery(query) || queryReturnsRows(query) {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		scannedRows, cols, err := scanRows(rows)
		if err != nil {
			return nil, err
		}
		return &queryOutcome{rows: scannedRows, columns: cols}, nil
	}

	res, err := db.ExecContext(ctx, query)
	if err != nil {
		return nil, err
	}
	affected, _ := res.RowsAffected()
	return &queryOutcome{rowsAffected: affected}, nil
}

func scanRows(rows *sql.Rows) ([]map[string]any, []string, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	values := make([]any, len(columns))
	scanArgs := make([]any, len(columns))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	var out []map[string]any
	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, nil, err
		}
		rowMap := make(map[string]any, len(columns))
		for i, col := range columns {
			val := values[i]
			switch v := val.(type) {
			case []byte:
				rowMap[col] = string(v)
			default:
				rowMap[col] = v
			}
		}
		out = append(out, rowMap)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return out, columns, nil
}
