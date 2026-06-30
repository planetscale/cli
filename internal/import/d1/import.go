package d1

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/planetscale/cli/internal/postgres"
	"github.com/planetscale/cli/internal/roleutil"
	execabs "golang.org/x/sys/execabs"
)

// ImportOptions configures D1 import into PlanetScale Postgres.
type ImportOptions struct {
	Org         string
	Database    string
	Branch      string
	InputPath   string
	Method      string
	MigrationID string
	DBName      string
	DryRun      bool
	DestURI     string // optional override for testing
	NotifyAPI   NotifyAPIConfig
	OnProgress  ImportProgressFunc
	// PgloaderVerbose emits full pgloader reports to stderr (defaults to false).
	PgloaderVerbose bool
	notifyBase      importNotificationPayload
}

// ImportClient abstracts PlanetScale API access for import.
type ImportClient interface {
	GetDatabase(ctx context.Context, org, database string) (*ps.Database, error)
}

// DefaultImportClient wraps planetscale client.
type DefaultImportClient struct {
	Client *ps.Client
}

func (c *DefaultImportClient) GetDatabase(ctx context.Context, org, database string) (*ps.Database, error) {
	return c.Client.Databases.Get(ctx, &ps.GetDatabaseRequest{
		Organization: org,
		Database:     database,
	})
}

// Import loads a D1 SQLite dump into PlanetScale Postgres.
// Pass prepared when the caller already ran PrepareImport (e.g. human confirm flow).
func Import(ctx context.Context, psClient *ps.Client, client ImportClient, opts ImportOptions, prepared *ImportPrepareResult) (result *ImportResult, err error) {
	if prepared == nil {
		prepared, err = PrepareImport(opts)
		if err != nil {
			return nil, err
		}
	}

	opts.MigrationID = prepared.MigrationID
	opts.Method = prepared.Method

	result = importResultFromPrepare(prepared, opts.DryRun)

	if !prepared.CanProceed {
		return result, ErrLintBlocked(prepared.BlockedReason)
	}

	if opts.DryRun {
		return result, nil
	}

	if _, err := FindPgloader(); err != nil {
		return result, err
	}

	importStarted := false
	importDataLoaded := false
	var importStart time.Time
	defer func() {
		if err != nil && importStarted && !importDataLoaded {
			_ = saveImportMigrationState(opts, PhaseFailed, "")
			if !opts.DryRun {
				payload := notifyPayloadFromImport(opts, result)
				payload.DurationMs = time.Since(importStart).Milliseconds()
				notifyImportFailure(opts.NotifyAPI, opts.Org, opts.Database, opts.Branch, opts.MigrationID, payload, err, nil)
			}
		}
	}()

	importStart = time.Now()
	timings := &ImportTimings{}
	importStarted = true
	opts.notifyBase = notifyPayloadFromImport(opts, result)
	NotifyImportEventSync(opts.NotifyAPI, opts.Org, opts.Database, opts.Branch, opts.MigrationID, NotifyEventStarting, opts.notifyBase)

	opts.reportProgress(ImportProgress{Stage: ImportStageConnecting})

	db, err := client.GetDatabase(ctx, opts.Org, opts.Database)
	if err != nil {
		return result, fmt.Errorf("get database: %w", err)
	}
	if db.Kind != "postgresql" {
		return result, newMigrationError(
			ErrCodeInvalidInput,
			fmt.Sprintf("database %s is not PostgreSQL", opts.Database),
			"Create a PostgreSQL database branch for D1 migration",
		)
	}

	sqlitePath := DefaultSQLitePath(opts.InputPath)
	if state, stateErr := LoadState(opts.Org, opts.Database, opts.Branch, opts.MigrationID); stateErr == nil {
		if state.InputPath != "" && state.InputPath != opts.InputPath {
			return result, newMigrationError(
				ErrCodeInvalidInput,
				fmt.Sprintf("input path %q does not match migration state %q", opts.InputPath, state.InputPath),
				"Use the same --input as the original import or omit --migration-id to start fresh",
			)
		}
		if state.SQLitePath != "" {
			sqlitePath = state.SQLitePath
		}
	}

	if importResumeEnabled(opts) {
		if err := saveImportMigrationState(opts, PhaseImporting, ""); err != nil {
			return result, err
		}
	} else if err := resetImportProgress(opts, PhaseImporting, ""); err != nil {
		return result, err
	}

	sqliteStart := time.Now()
	opts.reportProgress(ImportProgress{Stage: ImportStageSQLiteStaging})
	if err := EnsureSQLiteFromDump(ctx, opts.InputPath, sqlitePath); err != nil {
		remediation := "Ensure the dump is valid and the host has enough memory and disk for SQLite staging"
		if errors.Is(err, context.Canceled) {
			remediation = "Import was interrupted; re-run start to resume or start fresh"
		}
		return result, newMigrationError(ErrCodeImportFailed, err.Error(), remediation)
	}
	timings.SQLiteStagingMs = time.Since(sqliteStart).Milliseconds()

	destURI, cleanup, err := ResolveDestURI(ctx, psClient, opts)
	if err != nil {
		return result, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	currentUser, err := usernameFromDestURI(destURI)
	if err != nil {
		return result, err
	}
	if err := cleanupStaleImportRoles(ctx, psClient, opts, currentUser); err != nil {
		return result, err
	}

	switch opts.Method {
	case MethodPgloader:
		if err := importWithPgloader(ctx, opts, destURI, sqlitePath, timings); err != nil {
			return result, err
		}
	case MethodPsql:
		if err := importSmall(ctx, opts, destURI, sqlitePath); err != nil {
			return result, err
		}
	default:
		return result, newMigrationError(ErrCodeInvalidInput, "unknown import method: "+opts.Method, "Use pgloader (large dumps) or psql (small dumps; data loaded via pgloader)")
	}
	importDataLoaded = true

	tables, err := ParseDump(opts.InputPath)
	if err == nil {
		for _, table := range tables {
			if !IsORMMetadataTable(table.Name) {
				result.TablesLoaded++
			}
		}
	}

	timings.TotalMs = time.Since(importStart).Milliseconds()
	result.Timings = timings

	loadedTables, _ := PgloaderLoadTables(opts.InputPath)
	state := &MigrationState{
		MigrationID:  opts.MigrationID,
		Org:          opts.Org,
		Database:     opts.Database,
		Branch:       opts.Branch,
		InputPath:    opts.InputPath,
		SQLitePath:   sqlitePath,
		DBName:       opts.DBName,
		Method:       opts.Method,
		Phase:        PhaseImported,
		LoadedTables: loadedTables,
	}
	if !opts.DryRun {
		if err := SaveState(state); err != nil {
			return result, errStatePersist("import", err)
		}
		NotifyImportEventSync(opts.NotifyAPI, opts.Org, opts.Database, opts.Branch, opts.MigrationID, NotifyEventImported, notifyPayloadFromImport(opts, result))
	}

	return result, nil
}

func importWithPgloader(ctx context.Context, opts ImportOptions, destURI, sqlitePath string, timings *ImportTimings) error {
	resume := importResumeEnabled(opts)
	if !resume {
		opts.reportProgress(ImportProgress{Stage: ImportStageSchema})
		schemaStart := time.Now()
		if err := applyPostgresSchema(ctx, opts, destURI); err != nil {
			return err
		}
		timings.SchemaMs = time.Since(schemaStart).Milliseconds()
	}
	return loadTablesAndFinalize(ctx, opts, destURI, sqlitePath, timings, resume)
}

// importSmall loads dumps under 1GB: schema via psql, data via pgloader.
func importSmall(ctx context.Context, opts ImportOptions, destURI, sqlitePath string) error {
	resume := importResumeEnabled(opts)
	if !resume {
		opts.reportProgress(ImportProgress{Stage: ImportStageSchema})
		if err := applyPostgresSchema(ctx, opts, destURI); err != nil {
			return err
		}
	}
	return loadTablesAndFinalize(ctx, opts, destURI, sqlitePath, nil, resume)
}

func importResumeEnabled(opts ImportOptions) bool {
	state, err := LoadState(opts.Org, opts.Database, opts.Branch, opts.MigrationID)
	if err != nil {
		return false
	}
	if state.Phase != PhaseFailed && state.Phase != PhaseImporting {
		return false
	}
	if len(state.LoadedTables) > 0 {
		return true
	}
	return state.SchemaApplied
}

func loadTablesAndFinalize(ctx context.Context, opts ImportOptions, destURI, sqlitePath string, timings *ImportTimings, resume bool) error {
	loadTables, err := PgloaderLoadTables(opts.InputPath)
	if err != nil {
		return err
	}

	var skipTables []string
	if resume {
		if state, err := LoadState(opts.Org, opts.Database, opts.Branch, opts.MigrationID); err == nil {
			skipTables = state.LoadedTables
		}
	}

	pgTimings, err := RunPgloader(ctx, PgloaderOptions{
		SQLitePath:      sqlitePath,
		DestURI:         destURI,
		InputPath:       opts.InputPath,
		DataOnly:        true,
		Tables:          loadTables,
		SkipTables:      skipTables,
		OnProgress:      opts.reportProgress,
		PgloaderVerbose: opts.PgloaderVerbose,
		OnTableLoaded: func(table string) error {
			return appendLoadedTable(opts, table)
		},
	})
	if err != nil {
		return err
	}
	if timings != nil {
		timings.PgloaderMs = pgTimings.PgloaderMs
		timings.TableLoads = pgTimings.TableLoads
	}

	opts.reportProgress(ImportProgress{Stage: ImportStageIndexes})
	indexStart := time.Now()
	if err := applyPostgresIndexes(ctx, opts, destURI); err != nil {
		return err
	}
	if timings != nil {
		timings.IndexBuildMs = time.Since(indexStart).Milliseconds()
	}

	seqStart := time.Now()
	opts.reportProgress(ImportProgress{Stage: ImportStageSequences})
	if err := ResetImportedSequences(ctx, destURI, opts.InputPath); err != nil {
		return err
	}
	if timings != nil {
		timings.SequenceResetMs = time.Since(seqStart).Milliseconds()
	}
	return nil
}

// ResolveDestURI creates a short-lived Postgres role and returns a connection string.
func ResolveDestURI(ctx context.Context, psClient *ps.Client, opts ImportOptions) (string, func() error, error) {
	if opts.DestURI != "" {
		return opts.DestURI, func() error { return nil }, nil
	}
	if psClient == nil {
		return "", nil, fmt.Errorf("planetscale client required for import")
	}

	roleName := fmt.Sprintf("d1-import-%d", time.Now().Unix())
	role, err := roleutil.New(ctx, psClient, roleutil.Options{
		Organization:   opts.Org,
		Database:       opts.Database,
		Branch:         opts.Branch,
		Name:           roleName,
		TTL:            2 * time.Hour,
		InheritedRoles: []string{"postgres"},
	})
	if err != nil {
		return "", nil, fmt.Errorf("create destination role: %w", err)
	}

	dbName := opts.DBName
	if dbName == "" {
		dbName = "postgres"
	}

	uri := postgres.BuildConnectionString(&postgres.Config{
		Host:     role.Role.AccessHostURL,
		Port:     5432,
		User:     role.Role.Username,
		Password: role.Role.Password,
		Database: dbName,
		SSLMode:  "require",
		Options:  map[string]string{},
	})

	return uri, func() error { return role.Cleanup(ctx, "postgres") }, nil
}

// ResetImportedSequences aligns identity sequences with MAX(column) after pgloader import.
// Per-table pgloader runs may leave sequences at their initial value; setval is idempotent.
func ResetImportedSequences(ctx context.Context, destURI, inputPath string) error {
	tables, err := ParseDump(inputPath)
	if err != nil {
		return err
	}

	db, err := OpenPostgres(destURI)
	if err != nil {
		return err
	}
	defer db.Close()

	for _, table := range tables {
		if IsORMMetadataTable(table.Name) {
			continue
		}
		for _, col := range table.Columns {
			if !col.AutoIncrement {
				continue
			}
			query := fmt.Sprintf(
				`SELECT setval(pg_get_serial_sequence($1, $2), GREATEST(COALESCE((SELECT MAX(%s) FROM %s), 1), 1), true)`,
				quoteIdent(col.Name),
				quoteIdent(table.Name),
			)
			if _, err := db.ExecContext(ctx, query, "public."+table.Name, col.Name); err != nil {
				return fmt.Errorf("reset sequence %s.%s: %w", table.Name, col.Name, err)
			}
		}
	}
	return nil
}

func appendLoadedTable(opts ImportOptions, table string) error {
	return updateMigrationState(opts.Org, opts.Database, opts.Branch, opts.MigrationID, func(state *MigrationState) {
		for _, existing := range state.LoadedTables {
			if existing == table {
				return
			}
		}
		state.LoadedTables = append(state.LoadedTables, table)
	})
}

func setSchemaApplied(opts ImportOptions) error {
	return updateMigrationState(opts.Org, opts.Database, opts.Branch, opts.MigrationID, func(state *MigrationState) {
		state.SchemaApplied = true
	})
}

func resetImportProgress(opts ImportOptions, phase, sqlitePath string) error {
	return updateMigrationState(opts.Org, opts.Database, opts.Branch, opts.MigrationID, func(state *MigrationState) {
		state.Phase = phase
		state.SchemaApplied = false
		state.LoadedTables = nil
		if opts.InputPath != "" {
			state.InputPath = opts.InputPath
		}
		if opts.Method != "" {
			state.Method = opts.Method
		}
		if opts.DBName != "" {
			state.DBName = opts.DBName
		}
		if sqlitePath != "" {
			state.SQLitePath = sqlitePath
		}
	})
}

func applyPostgresSchema(ctx context.Context, opts ImportOptions, destURI string) error {
	tables, err := ParseDump(opts.InputPath)
	if err != nil {
		return err
	}

	importNames := importTableNames(tables)
	existing, err := existingPublicTables(ctx, destURI, importNames)
	if err != nil {
		return err
	}
	if conflicts := conflictingImportTables(importNames, existing); len(conflicts) > 0 {
		return errExistingImportTables(conflicts)
	}

	workDir, err := os.MkdirTemp("", "pscale-d1-schema-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	var b strings.Builder
	b.WriteString("-- Generated by pscale import d1\n")
	b.WriteString("-- Source: ")
	b.WriteString(opts.InputPath)
	b.WriteString("\n\n")
	b.WriteString(buildImportTablesSQL(tables))

	combinedPath := filepath.Join(workDir, fmt.Sprintf("postgres-tables-%s.sql", opts.MigrationID))
	if err := os.WriteFile(combinedPath, []byte(b.String()), 0o600); err != nil {
		return err
	}

	if err := runPsqlFile(ctx, destURI, combinedPath); err != nil {
		return err
	}
	return setSchemaApplied(opts)
}

func applyPostgresIndexes(ctx context.Context, opts ImportOptions, destURI string) error {
	parts, _, err := ConvertSchemaParts(opts.InputPath)
	if err != nil {
		return err
	}
	if strings.TrimSpace(parts.Indexes) == "" {
		return nil
	}

	workDir, err := os.MkdirTemp("", "pscale-d1-indexes-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	var b strings.Builder
	b.WriteString("-- Generated by pscale import d1 (post-load indexes)\n")
	fmt.Fprintf(&b, "SET maintenance_work_mem TO '%s';\n", pgloaderIndexMaintenanceWorkMem)
	b.WriteString(parts.Indexes)

	indexPath := filepath.Join(workDir, fmt.Sprintf("postgres-indexes-%s.sql", opts.MigrationID))
	if err := os.WriteFile(indexPath, []byte(b.String()), 0o600); err != nil {
		return err
	}

	return runPsqlFile(ctx, destURI, indexPath)
}

func runPsqlFile(ctx context.Context, destURI, path string) error {
	psqlPath, err := postgres.FindPsqlPath()
	if err != nil {
		return err
	}

	cmd := execabs.CommandContext(ctx, psqlPath, destURI, "-v", "ON_ERROR_STOP=1", "-f", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("psql %s: %w: %s", filepath.Base(path), err, string(out))
	}
	return nil
}

// Status returns migration state for status polling.
func Status(org, database, branch, migrationID string) (*MigrationState, error) {
	return LoadState(org, database, branch, migrationID)
}
