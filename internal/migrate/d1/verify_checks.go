package d1

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	execabs "golang.org/x/sys/execabs"
)

// VerifyCheckResult is a single post-import verification check.
type VerifyCheckResult struct {
	Name    string `json:"name"`
	Table   string `json:"table,omitempty"`
	Column  string `json:"column,omitempty"`
	Matched bool   `json:"matched"`
	Message string `json:"message,omitempty"`
	Source  string `json:"source,omitempty"`
	Dest    string `json:"dest,omitempty"`
}

type tableFingerprint struct {
	RowCount int64
	IDSum    int64
}

type booleanDistribution struct {
	TrueCount  int64
	FalseCount int64
	NullCount  int64
}

func verifyRowCounts(tableNames []string, sourceCounts, destCounts map[string]int64) ([]TableVerifyResult, bool) {
	results := make([]TableVerifyResult, 0, len(tableNames))
	matched := true
	for _, name := range tableNames {
		ok := sourceCounts[name] == destCounts[name]
		if !ok {
			matched = false
		}
		results = append(results, TableVerifyResult{
			Table:      name,
			SourceRows: sourceCounts[name],
			DestRows:   destCounts[name],
			Match:      ok,
		})
	}
	return results, matched
}

func verifyIdentitySequences(ctx context.Context, db *sql.DB, tables []TableSchema) ([]VerifyCheckResult, bool) {
	var checks []VerifyCheckResult
	matched := true

	for _, table := range tables {
		if IsORMMetadataTable(table.Name) {
			continue
		}
		for _, col := range table.Columns {
			if !col.AutoIncrement {
				continue
			}
			check, ok, err := verifyTableSequence(ctx, db, table.Name, col.Name)
			if err != nil {
				checks = append(checks, VerifyCheckResult{
					Name:    "sequences",
					Table:   table.Name,
					Column:  col.Name,
					Matched: false,
					Message: err.Error(),
				})
				matched = false
				continue
			}
			checks = append(checks, check)
			if !ok {
				matched = false
			}
		}
	}
	return checks, matched
}

func verifyTableSequence(ctx context.Context, db *sql.DB, table, column string) (VerifyCheckResult, bool, error) {
	check := VerifyCheckResult{
		Name:   "sequences",
		Table:  table,
		Column: column,
	}

	var maxID sql.NullInt64
	maxQuery := fmt.Sprintf(`SELECT MAX(%s) FROM %s`, quoteIdent(column), quoteIdent(table))
	if err := db.QueryRowContext(ctx, maxQuery).Scan(&maxID); err != nil {
		return check, false, fmt.Errorf("max %s.%s: %w", table, column, err)
	}
	if !maxID.Valid {
		check.Matched = true
		check.Message = "empty table"
		return check, true, nil
	}

	var seqName sql.NullString
	if err := db.QueryRowContext(ctx,
		`SELECT pg_get_serial_sequence($1, $2)`,
		"public."+table,
		column,
	).Scan(&seqName); err != nil {
		return check, false, fmt.Errorf("sequence lookup %s.%s: %w", table, column, err)
	}
	if !seqName.Valid || seqName.String == "" {
		check.Matched = true
		check.Message = "no sequence attached (non-identity column)"
		return check, true, nil
	}

	var lastValue int64
	var isCalled bool
	seqQuery := fmt.Sprintf(`SELECT last_value, is_called FROM %s`, seqName.String)
	if err := db.QueryRowContext(ctx, seqQuery).Scan(&lastValue, &isCalled); err != nil {
		return check, false, fmt.Errorf("read sequence %s: %w", seqName.String, err)
	}

	nextValue := lastValue
	if isCalled {
		nextValue = lastValue + 1
	}
	ok := maxID.Int64 < nextValue
	check.Matched = ok
	check.Source = fmt.Sprintf("max=%d", maxID.Int64)
	check.Dest = fmt.Sprintf("next=%d (last_value=%d is_called=%t)", nextValue, lastValue, isCalled)
	if !ok {
		check.Message = "sequence next value would collide with existing rows"
	} else {
		check.Message = "sequence ready for new inserts"
	}
	return check, ok, nil
}

func verifyBooleanColumns(ctx context.Context, db *sql.DB, sqlitePath string, tables []TableSchema) ([]VerifyCheckResult, bool, error) {
	var checks []VerifyCheckResult
	matched := true

	for _, table := range tables {
		if IsORMMetadataTable(table.Name) {
			continue
		}
		for _, col := range table.Columns {
			if !isBooleanColumn(col) {
				continue
			}
			src, err := sqliteBooleanDistribution(ctx, sqlitePath, table.Name, col.Name)
			if err != nil {
				return checks, false, err
			}
			dest, err := postgresBooleanDistribution(ctx, db, table.Name, col.Name)
			if err != nil {
				return checks, false, err
			}
			ok := src.TrueCount == dest.TrueCount && src.FalseCount == dest.FalseCount && src.NullCount == dest.NullCount
			check := VerifyCheckResult{
				Name:    "boolean_columns",
				Table:   table.Name,
				Column:  col.Name,
				Matched: ok,
				Source:  fmt.Sprintf("true=%d false=%d null=%d", src.TrueCount, src.FalseCount, src.NullCount),
				Dest:    fmt.Sprintf("true=%d false=%d null=%d", dest.TrueCount, dest.FalseCount, dest.NullCount),
			}
			if !ok {
				check.Message = "boolean value distribution mismatch after import"
				matched = false
			} else {
				check.Message = "boolean coercion matches source 0/1 distribution"
			}
			checks = append(checks, check)
		}
	}
	return checks, matched, nil
}

func sqliteBooleanDistribution(ctx context.Context, sqlitePath, table, column string) (booleanDistribution, error) {
	query := fmt.Sprintf(
		`SELECT SUM(CASE WHEN %q = 1 THEN 1 ELSE 0 END), SUM(CASE WHEN %q = 0 THEN 1 ELSE 0 END), SUM(CASE WHEN %q IS NULL THEN 1 ELSE 0 END) FROM %q;`,
		column, column, column, table,
	)
	return querySQLiteDistribution(ctx, sqlitePath, query)
}

func postgresBooleanDistribution(ctx context.Context, db *sql.DB, table, column string) (booleanDistribution, error) {
	query := fmt.Sprintf(
		`SELECT COUNT(*) FILTER (WHERE %s = TRUE), COUNT(*) FILTER (WHERE %s = FALSE), COUNT(*) FILTER (WHERE %s IS NULL) FROM %s`,
		quoteIdent(column), quoteIdent(column), quoteIdent(column), quoteIdent(table),
	)
	var dist booleanDistribution
	if err := db.QueryRowContext(ctx, query).Scan(&dist.TrueCount, &dist.FalseCount, &dist.NullCount); err != nil {
		return dist, err
	}
	return dist, nil
}

func querySQLiteDistribution(ctx context.Context, sqlitePath, query string) (booleanDistribution, error) {
	sqlite3, err := FindSQLite3()
	if err != nil {
		return booleanDistribution{}, err
	}
	out, err := runSQLiteQuery(ctx, sqlite3, sqlitePath, query)
	if err != nil {
		return booleanDistribution{}, err
	}
	parts := parseSQLiteCLIFields(out)
	if len(parts) < 3 {
		return booleanDistribution{}, fmt.Errorf("unexpected boolean count output: %q", string(out))
	}
	var dist booleanDistribution
	for i, ptr := range []*int64{&dist.TrueCount, &dist.FalseCount, &dist.NullCount} {
		if parts[i] == "" || parts[i] == "NULL" {
			continue
		}
		if _, err := fmt.Sscanf(parts[i], "%d", ptr); err != nil {
			return booleanDistribution{}, err
		}
	}
	return dist, nil
}

// parseSQLiteCLIFields splits sqlite3 CLI output. Multi-column results use '|'.
func parseSQLiteCLIFields(out []byte) []string {
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil
	}
	if strings.Contains(s, "|") {
		parts := strings.Split(s, "|")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	}
	return strings.Fields(s)
}

func verifyTableFingerprints(ctx context.Context, db *sql.DB, sqlitePath string, tables []TableSchema) ([]VerifyCheckResult, bool, error) {
	var checks []VerifyCheckResult
	matched := true

	for _, table := range tables {
		if IsORMMetadataTable(table.Name) {
			continue
		}
		pkCol := identityColumn(table)
		src, err := tableFingerprintFromSQLite(ctx, sqlitePath, table, pkCol, tables)
		if err != nil {
			return checks, false, err
		}
		dest, err := tableFingerprintFromPostgres(ctx, db, table, pkCol, tables)
		if err != nil {
			return checks, false, err
		}
		ok := src.RowCount == dest.RowCount && src.IDSum == dest.IDSum
		check := VerifyCheckResult{
			Name:    "table_fingerprint",
			Table:   table.Name,
			Matched: ok,
			Source:  fmt.Sprintf("rows=%d id_sum=%d", src.RowCount, src.IDSum),
			Dest:    fmt.Sprintf("rows=%d id_sum=%d", dest.RowCount, dest.IDSum),
		}
		if !ok {
			check.Message = "aggregate fingerprint mismatch"
			matched = false
		} else if shouldFingerprintPKSum(table, pkCol, tables) {
			check.Message = "row count and integer PK sum match"
		} else {
			check.Message = "row count match"
		}
		checks = append(checks, check)
	}
	return checks, matched, nil
}

func identityColumn(table TableSchema) string {
	for _, col := range table.Columns {
		if col.AutoIncrement {
			return col.Name
		}
	}
	for _, col := range table.Columns {
		if col.PrimaryKey {
			return col.Name
		}
	}
	return ""
}

func shouldFingerprintPKSum(table TableSchema, pkCol string, all []TableSchema) bool {
	if pkCol == "" {
		return false
	}
	col := columnByName(table, pkCol)
	if col.Name == "" {
		return false
	}
	if isUUIDColumn(col, table, all) {
		return false
	}
	upper := strings.ToUpper(col.Type)
	return col.AutoIncrement || strings.Contains(upper, "INT")
}

func tableFingerprintFromSQLite(ctx context.Context, sqlitePath string, table TableSchema, pkCol string, all []TableSchema) (tableFingerprint, error) {
	var query string
	if pkCol != "" && shouldFingerprintPKSum(table, pkCol, all) {
		query = fmt.Sprintf(`SELECT COUNT(*), COALESCE(SUM(CAST(%q AS INTEGER)), 0) FROM %q;`, pkCol, table.Name)
	} else {
		query = fmt.Sprintf(`SELECT COUNT(*), 0 FROM %q;`, table.Name)
	}
	sqlite3, err := FindSQLite3()
	if err != nil {
		return tableFingerprint{}, err
	}
	out, err := runSQLiteQuery(ctx, sqlite3, sqlitePath, query)
	if err != nil {
		return tableFingerprint{}, fmt.Errorf("sqlite fingerprint %s: %w", table.Name, err)
	}
	var fp tableFingerprint
	fields := parseSQLiteCLIFields(out)
	if len(fields) < 2 {
		return tableFingerprint{}, fmt.Errorf("sqlite fingerprint %s: unexpected output %q", table.Name, string(out))
	}
	if _, err := fmt.Sscanf(fields[0], "%d", &fp.RowCount); err != nil {
		return tableFingerprint{}, fmt.Errorf("sqlite fingerprint %s row count: %w", table.Name, err)
	}
	if _, err := fmt.Sscanf(fields[1], "%d", &fp.IDSum); err != nil {
		return tableFingerprint{}, fmt.Errorf("sqlite fingerprint %s id sum: %w", table.Name, err)
	}
	return fp, nil
}

func tableFingerprintFromPostgres(ctx context.Context, db *sql.DB, table TableSchema, pkCol string, all []TableSchema) (tableFingerprint, error) {
	var fp tableFingerprint
	var query string
	if pkCol != "" && shouldFingerprintPKSum(table, pkCol, all) {
		query = fmt.Sprintf(`SELECT COUNT(*), COALESCE(SUM(%s::bigint), 0) FROM %s`, quoteIdent(pkCol), quoteIdent(table.Name))
	} else {
		query = fmt.Sprintf(`SELECT COUNT(*), 0 FROM %s`, quoteIdent(table.Name))
	}
	if err := db.QueryRowContext(ctx, query).Scan(&fp.RowCount, &fp.IDSum); err != nil {
		return fp, fmt.Errorf("postgres fingerprint %s: %w", table.Name, err)
	}
	return fp, nil
}

func verifySampleRows(ctx context.Context, db *sql.DB, sqlitePath string, tables []TableSchema, maxTables, samplesPerTable int) ([]VerifyCheckResult, bool, error) {
	var checks []VerifyCheckResult
	matched := true
	checked := 0

	for _, table := range tables {
		if IsORMMetadataTable(table.Name) {
			continue
		}
		if checked >= maxTables {
			break
		}
		pkCol := identityColumn(table)
		if pkCol == "" {
			continue
		}
		ids, err := samplePrimaryKeys(ctx, sqlitePath, table.Name, pkCol, samplesPerTable)
		if err != nil {
			return checks, false, err
		}
		if len(ids) == 0 {
			continue
		}
		checked++

		for _, id := range ids {
			src, err := sqliteRowSignature(ctx, sqlitePath, table, pkCol, id)
			if err != nil {
				return checks, false, err
			}
			dest, err := postgresRowSignature(ctx, db, table, pkCol, id, tables)
			if err != nil {
				return checks, false, err
			}
			ok := rowSignaturesMatch(src, dest, table, tables)
			check := VerifyCheckResult{
				Name:    "sample_rows",
				Table:   table.Name,
				Column:  pkCol,
				Matched: ok,
				Source:  src,
				Dest:    dest,
			}
			if !ok {
				check.Message = fmt.Sprintf("row signature mismatch for %s=%s", pkCol, id)
				matched = false
			} else {
				check.Message = fmt.Sprintf("row signature match for %s=%s", pkCol, id)
			}
			checks = append(checks, check)
		}
	}
	return checks, matched, nil
}

func samplePrimaryKeys(ctx context.Context, sqlitePath, table, pkCol string, limit int) ([]string, error) {
	sqlite3, err := FindSQLite3()
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`SELECT %q FROM %q ORDER BY %q LIMIT %d;`, pkCol, table, pkCol, limit)
	out, err := runSQLiteQuery(ctx, sqlite3, sqlitePath, query)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var ids []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			ids = append(ids, line)
		}
	}
	return ids, nil
}

func sqliteSignatureColumnExpr(col ColumnSchema) string {
	if isBooleanColumn(col) {
		return fmt.Sprintf(`CASE WHEN %q IN (1, '1') THEN '1' WHEN %q IN (0, '0') THEN '0' ELSE '' END`, col.Name, col.Name)
	}
	if isJSONText(col) {
		return fmt.Sprintf(`COALESCE(json(%q), CAST(%q AS TEXT), '')`, col.Name, col.Name)
	}
	if isBlobColumn(col) {
		return fmt.Sprintf(`COALESCE(hex(%q), '')`, col.Name)
	}
	return fmt.Sprintf(`COALESCE(CAST(%q AS TEXT), '')`, col.Name)
}

func postgresSignatureColumnExpr(col ColumnSchema, table TableSchema, all []TableSchema) string {
	pgType := sqliteTypeToPostgres(col, table, all)
	switch pgType {
	case "BOOLEAN":
		name := quoteIdent(col.Name)
		return fmt.Sprintf(`CASE WHEN %s IS TRUE THEN '1' WHEN %s IS FALSE THEN '0' ELSE '' END`, name, name)
	case "TIMESTAMPTZ":
		name := quoteIdent(col.Name)
		return fmt.Sprintf(`COALESCE(to_char(%s AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')`, name)
	case "JSONB":
		name := quoteIdent(col.Name)
		return fmt.Sprintf(`COALESCE(%s::jsonb::text, '')`, name)
	case "BYTEA":
		name := quoteIdent(col.Name)
		return fmt.Sprintf(`COALESCE(encode(%s, 'hex'), '')`, name)
	default:
		return fmt.Sprintf(`COALESCE(%s::text, '')`, quoteIdent(col.Name))
	}
}

func rowSignaturesMatch(src, dest string, table TableSchema, all []TableSchema) bool {
	srcParts := strings.Split(src, "|")
	destParts := strings.Split(dest, "|")
	if len(srcParts) != len(destParts) || len(srcParts) != len(table.Columns) {
		return src == dest
	}
	for i, col := range table.Columns {
		pgType := sqliteTypeToPostgres(col, table, all)
		switch pgType {
		case "JSONB":
			if !jsonValuesEqual(srcParts[i], destParts[i]) {
				return false
			}
		case "BYTEA":
			if !byteaValuesEqual(srcParts[i], destParts[i]) {
				return false
			}
		default:
			if srcParts[i] != destParts[i] {
				return false
			}
		}
	}
	return true
}

func jsonValuesEqual(a, b string) bool {
	ca, errA := canonicalJSON(a)
	cb, errB := canonicalJSON(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return ca == cb
}

func canonicalJSON(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return "", err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func byteaValuesEqual(sqliteText, pgText string) bool {
	if sqliteText == pgText {
		return true
	}
	if strings.HasPrefix(pgText, `\x`) {
		decoded, err := hex.DecodeString(strings.TrimPrefix(pgText, `\x`))
		if err == nil && sqliteText == string(decoded) {
			return true
		}
	}
	return false
}

func isBlobColumn(col ColumnSchema) bool {
	return strings.Contains(strings.ToUpper(col.Type), "BLOB")
}

func sqliteRowSignature(ctx context.Context, sqlitePath string, table TableSchema, pkCol, pkVal string) (string, error) {
	cols := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		cols = append(cols, sqliteSignatureColumnExpr(col))
	}
	query := fmt.Sprintf(
		`SELECT %s FROM %q WHERE %q = %s LIMIT 1;`,
		strings.Join(cols, " || '|' || "),
		table.Name,
		pkCol,
		sqliteLiteral(pkVal),
	)
	sqlite3, err := FindSQLite3()
	if err != nil {
		return "", err
	}
	out, err := runSQLiteQuery(ctx, sqlite3, sqlitePath, query)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func postgresRowSignature(ctx context.Context, db *sql.DB, table TableSchema, pkCol, pkVal string, all []TableSchema) (string, error) {
	cols := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		cols = append(cols, postgresSignatureColumnExpr(col, table, all))
	}
	query := fmt.Sprintf(
		`SELECT %s FROM %s WHERE %s = $1 LIMIT 1`,
		strings.Join(cols, " || '|' || "),
		quoteIdent(table.Name),
		quoteIdent(pkCol),
	)
	var sig sql.NullString
	if err := db.QueryRowContext(ctx, query, pkVal).Scan(&sig); err != nil {
		return "", err
	}
	if !sig.Valid {
		return "", fmt.Errorf("row not found in %s where %s = %s", table.Name, pkCol, pkVal)
	}
	return sig.String, nil
}

func sqliteLiteral(val string) string {
	var n int64
	if _, err := fmt.Sscanf(val, "%d", &n); err == nil {
		return val
	}
	return "'" + strings.ReplaceAll(val, "'", "''") + "'"
}

func runSQLiteQuery(ctx context.Context, sqlite3, sqlitePath, query string) ([]byte, error) {
	cmd := execabs.CommandContext(ctx, sqlite3, sqlitePath, query)
	return cmd.Output()
}
