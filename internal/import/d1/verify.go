package d1

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

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

	dbName := opts.DBName
	if dbName == "" && opts.MigrationID != "" {
		if state, err := LoadState(opts.Org, opts.Database, opts.Branch, opts.MigrationID); err == nil && state.DBName != "" {
			dbName = state.DBName
		}
	}
	if dbName == "" {
		dbName = "postgres"
	}
	opts.DBName = dbName

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

	result = &VerifyResult{
		MigrationID: opts.MigrationID,
		Matched:     true,
		Checks:      []VerifyCheckResult{},
	}

	var rowCountsOK bool
	result.Tables, rowCountsOK = verifyRowCounts(tableNames, sourceCounts, destCounts)
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
		query := fmt.Sprintf("SELECT COUNT(*) FROM %q;", table)
		cmd := execabs.CommandContext(ctx, sqlite3, sqlitePath, query)
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("sqlite count %s: %w", table, err)
		}
		var count int64
		if _, err := fmt.Sscanf(string(out), "%d", &count); err != nil {
			return nil, err
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
		query := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, quoteIdent(table))
		if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return nil, fmt.Errorf("count %s: %w", table, err)
		}
		counts[table] = count
	}
	return counts, nil
}

func resolveVerifySQLitePath(opts VerifyOptions) (VerifyOptions, string, error) {
	if opts.SQLitePath != "" {
		return opts, opts.SQLitePath, nil
	}

	if state, err := LoadState(opts.Org, opts.Database, opts.Branch, opts.MigrationID); err == nil {
		if err := validateInputPathAgainstState(opts.InputPath, state.InputPath); err != nil {
			return opts, "", err
		}
		if opts.InputPath == "" {
			opts.InputPath = state.InputPath
		}
		if state.SQLitePath != "" {
			return opts, state.SQLitePath, nil
		}
	} else if opts.InputPath == "" {
		return opts, "", err
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

func validateInputPathAgainstState(provided, saved string) error {
	if provided == "" || saved == "" {
		return nil
	}
	a, errA := filepath.Abs(provided)
	b, errB := filepath.Abs(saved)
	if errA != nil {
		a = provided
	}
	if errB != nil {
		b = saved
	}
	if a != b {
		return newMigrationError(
			ErrCodeInvalidInput,
			fmt.Sprintf("input path %q does not match migration state %q", provided, saved),
			"Use the same --input as the original import or omit --input to use saved state",
		)
	}
	return nil
}
