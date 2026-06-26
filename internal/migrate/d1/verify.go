package d1

import (
	"context"
	"fmt"

	execabs "golang.org/x/sys/execabs"
)

// Verify compares SQLite source data with PlanetScale Postgres after import.
func Verify(ctx context.Context, opts VerifyOptions) (*VerifyResult, error) {
	if opts.DestURI == "" {
		return nil, newMigrationError(
			ErrCodeInvalidInput,
			"destination database connection required for verify",
			"Pass --database (and --org/--branch) so verify can compare against PlanetScale Postgres",
		)
	}

	sqlitePath := opts.SQLitePath
	if sqlitePath == "" {
		state, err := LoadState(opts.Org, opts.Database, opts.Branch, opts.MigrationID)
		if err != nil {
			if opts.InputPath != "" {
				sqlitePath = DefaultSQLitePath(opts.InputPath)
			} else {
				return nil, err
			}
		} else {
			sqlitePath = state.SQLitePath
			if opts.InputPath == "" {
				opts.InputPath = state.InputPath
			}
		}
	}
	if opts.InputPath == "" {
		return nil, newMigrationError(
			ErrCodeMissingInput,
			"input dump path required for verify",
			"Pass --input or run verify with a migration-id from a prior import",
		)
	}

	tables, err := ParseDump(opts.InputPath)
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

	sourceCounts, err := CountSQLiteRows(ctx, sqlitePath, tableNames)
	if err != nil {
		insertCounts, insertErr := CountInsertRows(opts.InputPath)
		if insertErr != nil {
			return nil, err
		}
		sourceCounts = mapStringIntToInt64(insertCounts)
	}

	destCounts, err := CountPostgresRows(ctx, opts.DestURI, tableNames)
	if err != nil {
		return nil, err
	}

	result := &VerifyResult{
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

	seqChecks, ok := verifyIdentitySequences(ctx, db, dataTables)
	result.Checks = append(result.Checks, seqChecks...)
	if !ok {
		result.Matched = false
	}

	boolChecks, ok, err := verifyBooleanColumns(ctx, db, sqlitePath, dataTables)
	if err != nil {
		return nil, err
	}
	result.Checks = append(result.Checks, boolChecks...)
	if !ok {
		result.Matched = false
	}

	fpChecks, ok, err := verifyTableFingerprints(ctx, db, sqlitePath, dataTables)
	if err != nil {
		return nil, err
	}
	result.Checks = append(result.Checks, fpChecks...)
	if !ok {
		result.Matched = false
	}

	sampleChecks, ok, err := verifySampleRows(ctx, db, sqlitePath, dataTables, 8, 3)
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

	if opts.MigrationID != "" {
		if err := SetMigrationPhase(opts.Org, opts.Database, opts.Branch, opts.MigrationID, PhaseVerified); err != nil {
			return result, err
		}
	}

	return result, nil
}

func mapStringIntToInt64(in map[string]int) map[string]int64 {
	out := make(map[string]int64, len(in))
	for k, v := range in {
		out[k] = int64(v)
	}
	return out
}

// CountSQLiteRows counts rows using sqlite3 CLI.
func CountSQLiteRows(ctx context.Context, sqlitePath string, tables []string) (map[string]int64, error) {
	sqlite3, err := FindSQLite3()
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64, len(tables))
	for _, table := range tables {
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
	db, err := OpenPostgres(destURI)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	counts := make(map[string]int64, len(tables))
	for _, table := range tables {
		var count int64
		query := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, quoteIdent(table))
		if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return nil, fmt.Errorf("count %s: %w", table, err)
		}
		counts[table] = count
	}
	return counts, nil
}
