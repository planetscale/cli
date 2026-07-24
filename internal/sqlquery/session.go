package sqlquery

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
)

// Session is an open connection to a database branch using ephemeral
// credentials. Unlike Execute, which opens and tears down a connection per
// query, a Session lets callers run several queries (e.g. a suite of
// diagnostic checks) over a single connection. Close must be called to release
// the connection and clean up the ephemeral credentials.
type Session struct {
	// Kind is the database kind as reported by the API: "mysql",
	// "postgresql", or "horizon".
	Kind string

	db      *sql.DB
	cleanup func()
}

// NewSession validates the options, looks up the database and branch, and
// opens a connection with ephemeral credentials (a branch password for MySQL,
// a temporary role for PostgreSQL).
func NewSession(ctx context.Context, ch *cmdutil.Helper, opts Options) (*Session, error) {
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

	var db *sql.DB
	var cleanup func()

	switch string(dbInfo.Kind) {
	case "mysql":
		db, cleanup, err = openMySQL(ctx, ch, opts, role)
	case "postgresql", "horizon":
		pgDB := opts.PostgresDB
		if pgDB == "" {
			pgDB = "postgres"
		}
		db, cleanup, err = openPostgres(ctx, ch, opts, pgDB, role)
	default:
		return nil, fmt.Errorf("unsupported database kind %q", dbInfo.Kind)
	}
	if err != nil {
		return nil, err
	}

	return &Session{Kind: string(dbInfo.Kind), db: db, cleanup: cleanup}, nil
}

// Engine normalizes Kind to "mysql" or "postgresql".
func (s *Session) Engine() string {
	if s.Kind == "mysql" {
		return "mysql"
	}
	return "postgresql"
}

// Query runs a single query over the session's connection and returns the
// column names (in result order) and rows.
func (s *Session) Query(ctx context.Context, query string) ([]string, []map[string]any, error) {
	outcome, err := runQuery(ctx, s.db, query)
	if err != nil {
		return nil, nil, err
	}
	return outcome.columns, outcome.rows, nil
}

// Close releases the connection and cleans up the ephemeral credentials.
func (s *Session) Close() {
	if s.cleanup != nil {
		s.cleanup()
		s.cleanup = nil
	}
}
