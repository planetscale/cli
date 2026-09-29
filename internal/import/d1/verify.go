package d1

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/planetscale/cli/internal/postgres"
	execabs "golang.org/x/sys/execabs"
)

// Verify compares SQLite source data with PlanetScale Postgres after import.
func Verify(ctx context.Context, opts VerifyOptions) (result *VerifyResult, err error) {
	verifyStart := time.Now()
	verifyChecksPassed := false
	defer func() {
		if err == nil || verifyChecksPassed {
			return
		}
		payload := importNotificationPayload{
			DurationMs: time.Since(verifyStart).Milliseconds(),
		}
		notifyImportFailure(opts.NotifyAPI, opts.Org, opts.Database, opts.Branch, opts.MigrationID, payload, err, result)
	}()

	if opts.DestURI == "" {
		return nil, newMigrationError(
			ErrCodeInvalidInput,
			"destination database connection required for verify",
			"Pass database and branch as positional arguments so verify can compare against PlanetScale Postgres",
		)
	}

	opts, sqlitePath, err := resolveVerifySQLitePath(opts)
	if err != nil {
		return nil, err
	}

	if opts.InputPath != "" && opts.SQLitePath == "" {
		if err := EnsureSQLiteFromDump(ctx, opts.InputPath, sqlitePath); err != nil {
			return nil, newMigrationError(
				ErrCodeVerifyFailed,
				fmt.Sprintf("build sqlite staging: %v", err),
				"Ensure the dump is valid and sqlite3 is installed; pass --sqlite for a custom staging path",
			)
		}
	}

	opts.DBName = ResolveVerifyDBName(opts, false)

	opts.notifyBase = notifyPayloadFromVerify(opts)

	tables, err := ParseDump(opts.InputPath)
	if err != nil {
		return nil, err
	}

	coerceCtx, err := BuildTypeCoercionContext(opts.InputPath, tables)
	if err != nil {
		return nil, err
	}

	tableNames := make([]string, 0, len(tables))
	dataTables := make([]TableSchema, 0, len(tables))
	for _, t := range tables {
		if IsORMMetadataTable(t.Name) {
			continue
		}
		tableNames = append(tableNames, t.Name)
		dataTables = append(dataTables, t)
	}

	NotifyImportEventSync(opts.NotifyAPI, opts.Org, opts.Database, opts.Branch, opts.MigrationID, NotifyEventVerifying, importNotificationPayload{})

	opts.reportProgress(ImportProgress{Stage: VerifyStageRowCounts, Total: len(tableNames)})
	sourceCounts, err := countSQLiteRowsWithProgress(ctx, opts, sqlitePath, tableNames)
	if err != nil {
		return nil, newMigrationError(
			ErrCodeVerifyFailed,
			fmt.Sprintf("count source rows: %v", err),
			"Ensure sqlite3 is installed and the staging database is readable; pass --sqlite if using a custom path",
		)
	}

	destCounts, err := countPostgresRowsWithProgress(ctx, opts, tableNames)
	if err != nil {
		return nil, err
	}
	destCounts, extraTables, err := mergeImportScopedDestRowCounts(ctx, opts, tableNames, destCounts)
	if err != nil {
		return nil, err
	}

	result = &VerifyResult{
		MigrationID: opts.MigrationID,
		Matched:     true,
		Checks:      []VerifyCheckResult{},
	}

	verifyTables := append(append([]string{}, tableNames...), extraTables...)
	var rowCountsOK bool
	result.Tables, rowCountsOK = verifyRowCounts(verifyTables, sourceCounts, destCounts)
	if !rowCountsOK {
		result.Matched = false
	}

	db, err := OpenPostgres(opts.DestURI)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	opts.reportProgress(ImportProgress{Stage: VerifyStageSequences})
	seqChecks, ok := verifyIdentitySequences(ctx, db, dataTables)
	result.Checks = append(result.Checks, seqChecks...)
	if !ok {
		result.Matched = false
	}

	opts.reportProgress(ImportProgress{Stage: VerifyStageBoolean})
	boolChecks, ok, err := verifyBooleanColumns(ctx, db, sqlitePath, dataTables, coerceCtx)
	if err != nil {
		return nil, err
	}
	result.Checks = append(result.Checks, boolChecks...)
	if !ok {
		result.Matched = false
	}

	opts.reportProgress(ImportProgress{Stage: VerifyStageFingerprints})
	fpChecks, ok, err := verifyTableFingerprints(ctx, db, sqlitePath, dataTables, coerceCtx)
	if err != nil {
		return nil, err
	}
	result.Checks = append(result.Checks, fpChecks...)
	if !ok {
		result.Matched = false
	}

	opts.reportProgress(ImportProgress{Stage: VerifyStageSampleRows})
	sampleChecks, ok, err := verifySampleRows(ctx, db, sqlitePath, dataTables, coerceCtx, 8, 3)
	if err != nil {
		return nil, err
	}
	result.Checks = append(result.Checks, sampleChecks...)
	if !ok {
		result.Matched = false
	}

	if !result.Matched {
		return result, newMigrationError(
			ErrCodeVerifyFailed,
			"import verification failed (row counts, sequences, coercion, or content checks)",
			"Re-run import or inspect failing checks in verify JSON output",
		)
	}

	verifyChecksPassed = true

	if opts.MigrationID != "" {
		if err := SetMigrationPhase(opts.Org, opts.Database, opts.Branch, opts.MigrationID, PhaseVerified); err != nil {
			return result, errStatePersist("verify", err)
		}
	}

	if !opts.NotifyAPI.Disabled && opts.NotifyAPI.Client != nil {
		matched := result.Matched
		NotifyImportEventSync(opts.NotifyAPI, opts.Org, opts.Database, opts.Branch, opts.MigrationID, NotifyEventVerified, importNotificationPayload{
			Matched:    &matched,
			DurationMs: time.Since(verifyStart).Milliseconds(),
		})
	}

	return result, nil
}

// CountSQLiteRows counts rows using sqlite3 CLI.
func CountSQLiteRows(ctx context.Context, sqlitePath string, tables []string) (map[string]int64, error) {
	return countSQLiteRowsWithProgress(ctx, VerifyOptions{}, sqlitePath, tables)
}

func countSQLiteRowsWithProgress(ctx context.Context, opts VerifyOptions, sqlitePath string, tables []string) (map[string]int64, error) {
	sqlite3, err := FindSQLite3()
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64, len(tables))
	for i, table := range tables {
		opts.reportProgress(ImportProgress{
			Stage:   VerifyStageRowCounts,
			Current: i + 1,
			Total:   len(tables),
			Detail:  table + " (sqlite)",
		})
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s;", quoteSQLiteIdentifier(table))
		cmd := execabs.CommandContext(ctx, sqlite3, sqliteCLIArgs(sqlitePath, query)...)
		out, err := cmd.Output()
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) && len(ee.Stderr) > 0 {
				return nil, fmt.Errorf("sqlite count %s: %w: %s", table, err, truncateLoadError(strings.TrimSpace(string(ee.Stderr)), 200))
			}
			return nil, fmt.Errorf("sqlite count %s: %w", table, err)
		}
		count, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("sqlite count %s: unexpected output %q", table, truncateLoadError(string(out), 120))
		}
		counts[table] = count
	}
	return counts, nil
}

// CountPostgresRows counts rows in public schema tables.
func CountPostgresRows(ctx context.Context, destURI string, tables []string) (map[string]int64, error) {
	return countPostgresRowsWithProgress(ctx, VerifyOptions{DestURI: destURI}, tables)
}

func countPostgresRowsWithProgress(ctx context.Context, opts VerifyOptions, tables []string) (map[string]int64, error) {
	db, err := OpenPostgres(opts.DestURI)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	counts := make(map[string]int64, len(tables))
	for i, table := range tables {
		opts.reportProgress(ImportProgress{
			Stage:   VerifyStageRowCounts,
			Current: i + 1,
			Total:   len(tables),
			Detail:  table + " (postgres)",
		})
		var count int64
		query := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, postgres.QuoteIdentifier(table))
		if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return nil, fmt.Errorf("count %s: %w", table, err)
		}
		counts[table] = count
	}
	return counts, nil
}

func mergeImportScopedDestRowCounts(ctx context.Context, opts VerifyOptions, sourceTables []string, destCounts map[string]int64) (map[string]int64, []string, error) {
	if opts.MigrationID == "" {
		return destCounts, nil, nil
	}
	state, err := LoadState(opts.Org, opts.Database, opts.Branch, opts.MigrationID)
	if err != nil {
		return destCounts, nil, nil
	}
	if len(state.LoadedTables) == 0 {
		return destCounts, nil, nil
	}

	sourceSet := make(map[string]struct{}, len(sourceTables))
	for _, name := range sourceTables {
		sourceSet[name] = struct{}{}
	}

	var extra []string
	for _, name := range state.LoadedTables {
		if _, ok := sourceSet[name]; ok {
			continue
		}
		extra = append(extra, name)
	}
	if len(extra) == 0 {
		return destCounts, nil, nil
	}

	db, err := OpenPostgres(opts.DestURI)
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()

	if destCounts == nil {
		destCounts = make(map[string]int64)
	}
	for _, name := range extra {
		var count int64
		query := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, postgres.QuoteIdentifier(name))
		if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return nil, nil, fmt.Errorf("count import-scoped table %s: %w", name, err)
		}
		destCounts[name] = count
	}
	return destCounts, extra, nil
}

// ResolveVerifyDBName returns the Postgres database name for verify. When dbNameExplicit
// is false and migration state records a db_name, that value is preferred over the CLI default.
func ResolveVerifyDBName(opts VerifyOptions, dbNameExplicit bool) string {
	return ResolveMigrationDBName(opts.Org, opts.Database, opts.Branch, opts.MigrationID, opts.DBName, dbNameExplicit)
}

func resolveVerifySQLitePath(opts VerifyOptions) (VerifyOptions, string, error) {
	if opts.SQLitePath != "" {
		if opts.InputPath == "" && opts.MigrationID != "" {
			state, err := LoadState(opts.Org, opts.Database, opts.Branch, opts.MigrationID)
			if err != nil {
				return opts, "", err
			}
			opts.InputPath = state.InputPath
		}
		return opts, opts.SQLitePath, nil
	}

	if opts.MigrationID != "" {
		state, err := LoadState(opts.Org, opts.Database, opts.Branch, opts.MigrationID)
		if err != nil {
			return opts, "", err
		}
		if err := validateInputPathAgainstState(opts.InputPath, state.InputPath); err != nil {
			return opts, "", err
		}
		if opts.InputPath == "" {
			opts.InputPath = state.InputPath
		}
		if state.SQLitePath != "" {
			return opts, state.SQLitePath, nil
		}
		if opts.InputPath == "" {
			return opts, "", newMigrationError(
				ErrCodeMissingInput,
				"input dump path required for verify",
				"Pass --input or run verify with a migration-id from a prior import",
			)
		}
		return opts, DefaultSQLitePath(opts.InputPath), nil
	}

	if opts.InputPath == "" {
		return opts, "", newMigrationError(
			ErrCodeMissingInput,
			"input dump path required for verify",
			"Pass --input or run verify with a migration-id from a prior import",
		)
	}

	return opts, DefaultSQLitePath(opts.InputPath), nil
}
