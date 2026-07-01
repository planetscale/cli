package d1

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/planetscale/cli/internal/postgres"
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

const verifySignatureFieldMaxLen = 64

// summarizeRowSignatureForOutput shortens row signatures for CLI/JSON output.
// Full signatures are still used internally for comparison.
func summarizeRowSignatureForOutput(sig string, table TableSchema) string {
	parts := strings.Split(sig, "|")
	if len(parts) != len(table.Columns) {
		return truncateSignatureValue(sig, verifySignatureFieldMaxLen*4, false)
	}
	for i, part := range parts {
		parts[i] = summarizeSignatureField(part, table.Columns[i])
	}
	return strings.Join(parts, "|")
}

func summarizeSignatureField(value string, col ColumnSchema) string {
	if value == "" {
		return value
	}
	if isBlobColumn(col) {
		byteLen := len(value) / 2
		if utf8.ValidString(value) && len(value)%2 == 0 {
			if _, err := hex.DecodeString(value); err == nil && len(value) > verifySignatureFieldMaxLen {
				return truncateSignatureValue(value, verifySignatureFieldMaxLen, true) +
					fmt.Sprintf(" (%d bytes)", byteLen)
			}
		}
	}
	return truncateSignatureValue(value, verifySignatureFieldMaxLen, true)
}

func truncateSignatureValue(value string, maxLen int, addEllipsis bool) string {
	if len(value) <= maxLen {
		return value
	}
	if addEllipsis {
		return value[:maxLen] + "..."
	}
	return value[:maxLen]
}

type tableFingerprint struct {
	RowCount int64
	IDSum    string
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
	maxQuery := fmt.Sprintf(`SELECT MAX(%s) FROM %s`, postgres.QuoteIdentifier(column), postgres.QuoteIdentifier(table))
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

func verifyBooleanColumns(ctx context.Context, db *sql.DB, sqlitePath string, tables []TableSchema, coerceCtx *TypeCoercionContext) ([]VerifyCheckResult, bool, error) {
	var checks []VerifyCheckResult
	matched := true

	for _, table := range tables {
		if IsORMMetadataTable(table.Name) {
			continue
		}
		for _, col := range table.Columns {
			if !isBooleanLikeColumn(col, table, coerceCtx) {
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
		postgres.QuoteIdentifier(column), postgres.QuoteIdentifier(column), postgres.QuoteIdentifier(column), postgres.QuoteIdentifier(table),
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

func verifyTableFingerprints(ctx context.Context, db *sql.DB, sqlitePath string, tables []TableSchema, coerceCtx *TypeCoercionContext) ([]VerifyCheckResult, bool, error) {
	var checks []VerifyCheckResult
	matched := true

	for _, table := range tables {
		if IsORMMetadataTable(table.Name) {
			continue
		}
		pkCol := identityColumn(table)
		src, err := tableFingerprintFromSQLite(ctx, sqlitePath, table, pkCol, tables, coerceCtx)
		if err != nil {
			return checks, false, err
		}
		dest, err := tableFingerprintFromPostgres(ctx, db, table, pkCol, tables, coerceCtx)
		if err != nil {
			return checks, false, err
		}
		ok := src.RowCount == dest.RowCount && src.IDSum == dest.IDSum
		check := VerifyCheckResult{
			Name:    "table_fingerprint",
			Table:   table.Name,
			Matched: ok,
			Source:  fmt.Sprintf("rows=%d id_sum=%s", src.RowCount, src.IDSum),
			Dest:    fmt.Sprintf("rows=%d id_sum=%s", dest.RowCount, dest.IDSum),
		}
		if !ok {
			check.Message = "aggregate fingerprint mismatch"
			matched = false
		} else if shouldFingerprintPKSum(table, pkCol, tables, coerceCtx) {
			check.Message = "row count and integer PK sum match"
		} else {
			check.Message = "row count match"
		}
		checks = append(checks, check)
	}
	return checks, matched, nil
}

func identityColumn(table TableSchema) string {
	if cols := primaryKeyColumns(table); len(cols) == 1 {
		return cols[0]
	}
	return ""
}

func primaryKeyColumns(table TableSchema) []string {
	var pks []string
	for _, col := range table.Columns {
		if col.PrimaryKey {
			pks = append(pks, col.Name)
		}
	}
	if len(pks) > 0 {
		return pks
	}
	for _, col := range table.Columns {
		if col.AutoIncrement {
			return []string{col.Name}
		}
	}
	return nil
}

func shouldFingerprintPKSum(table TableSchema, pkCol string, all []TableSchema, coerceCtx *TypeCoercionContext) bool {
	if pkCol == "" {
		return false
	}
	col := columnByName(table, pkCol)
	if col.Name == "" {
		return false
	}
	if isUUIDColumn(col, table, all, coerceCtx) {
		return false
	}
	upper := strings.ToUpper(col.Type)
	return col.AutoIncrement || strings.Contains(upper, "INT")
}

func tableFingerprintFromSQLite(ctx context.Context, sqlitePath string, table TableSchema, pkCol string, all []TableSchema, coerceCtx *TypeCoercionContext) (tableFingerprint, error) {
	var query string
	if pkCol != "" && shouldFingerprintPKSum(table, pkCol, all, coerceCtx) {
		query = fmt.Sprintf(`SELECT COUNT(*), COALESCE(CAST(SUM(CAST(%q AS INTEGER)) AS TEXT), '0') FROM %q;`, pkCol, table.Name)
	} else {
		query = fmt.Sprintf(`SELECT COUNT(*), '0' FROM %q;`, table.Name)
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
	fp.IDSum = fields[1]
	return fp, nil
}

func tableFingerprintFromPostgres(ctx context.Context, db *sql.DB, table TableSchema, pkCol string, all []TableSchema, coerceCtx *TypeCoercionContext) (tableFingerprint, error) {
	var fp tableFingerprint
	var query string
	if pkCol != "" && shouldFingerprintPKSum(table, pkCol, all, coerceCtx) {
		query = fmt.Sprintf(`SELECT COUNT(*), COALESCE(SUM(%s::numeric)::text, '0') FROM %s`, postgres.QuoteIdentifier(pkCol), postgres.QuoteIdentifier(table.Name))
	} else {
		query = fmt.Sprintf(`SELECT COUNT(*), '0' FROM %s`, postgres.QuoteIdentifier(table.Name))
	}
	if err := db.QueryRowContext(ctx, query).Scan(&fp.RowCount, &fp.IDSum); err != nil {
		return fp, fmt.Errorf("postgres fingerprint %s: %w", table.Name, err)
	}
	return fp, nil
}

func verifySampleRows(ctx context.Context, db *sql.DB, sqlitePath string, tables []TableSchema, coerceCtx *TypeCoercionContext, maxTables, samplesPerTable int) ([]VerifyCheckResult, bool, error) {
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
		pkCols := primaryKeyColumns(table)
		if len(pkCols) != 1 {
			checks = append(checks, VerifyCheckResult{
				Name:    "sample_rows",
				Table:   table.Name,
				Message: "skipped (requires single-column primary key for row sampling)",
			})
			continue
		}
		pkCol := pkCols[0]
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
			src, err := sqliteRowSignature(ctx, sqlitePath, table, pkCol, id, coerceCtx)
			if err != nil {
				return checks, false, err
			}
			dest, err := postgresRowSignature(ctx, db, table, pkCol, id, tables, coerceCtx)
			if err != nil {
				return checks, false, err
			}
			ok := rowSignaturesMatch(src, dest, table, tables, coerceCtx)
			check := VerifyCheckResult{
				Name:    "sample_rows",
				Table:   table.Name,
				Column:  pkCol,
				Matched: ok,
			}
			if !ok {
				check.Source = summarizeRowSignatureForOutput(src, table)
				check.Dest = summarizeRowSignatureForOutput(dest, table)
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

func sqliteSignatureColumnExpr(col ColumnSchema, table TableSchema, coerceCtx *TypeCoercionContext) string {
	if isBooleanLikeColumn(col, table, coerceCtx) {
		return fmt.Sprintf(`CASE WHEN %q IN (1, '1') THEN '1' WHEN %q IN (0, '0') THEN '0' ELSE '' END`, col.Name, col.Name)
	}
	if isJSONText(col) && coerceCtx != nil && samplesAllowJSON(table.Name, col.Name, coerceCtx) {
		return fmt.Sprintf(`COALESCE(json(%q), CAST(%q AS TEXT), '')`, col.Name, col.Name)
	}
	if isBlobColumn(col) {
		return fmt.Sprintf(`COALESCE(hex(%q), '')`, col.Name)
	}
	if isTimestampText(col) && coerceCtx != nil && samplesAllowTimestamp(table.Name, col.Name, coerceCtx) {
		return fmt.Sprintf(`COALESCE(strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', %q), COALESCE(CAST(%q AS TEXT), ''))`, col.Name, col.Name)
	}
	return fmt.Sprintf(`COALESCE(CAST(%q AS TEXT), '')`, col.Name)
}

func postgresSignatureColumnExpr(col ColumnSchema, table TableSchema, all []TableSchema, coerceCtx *TypeCoercionContext) string {
	pgType := sqliteTypeToPostgres(col, table, all, coerceCtx)
	switch pgType {
	case "BOOLEAN":
		name := postgres.QuoteIdentifier(col.Name)
		return fmt.Sprintf(`CASE WHEN %s IS TRUE THEN '1' WHEN %s IS FALSE THEN '0' ELSE '' END`, name, name)
	case "TIMESTAMPTZ":
		name := postgres.QuoteIdentifier(col.Name)
		return fmt.Sprintf(`COALESCE(to_char(%s AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')`, name)
	case "JSONB":
		name := postgres.QuoteIdentifier(col.Name)
		return fmt.Sprintf(`COALESCE(%s::jsonb::text, '')`, name)
	case "BYTEA":
		name := postgres.QuoteIdentifier(col.Name)
		return fmt.Sprintf(`COALESCE(encode(%s, 'hex'), '')`, name)
	default:
		return fmt.Sprintf(`COALESCE(%s::text, '')`, postgres.QuoteIdentifier(col.Name))
	}
}

func rowSignaturesMatch(src, dest string, table TableSchema, all []TableSchema, coerceCtx *TypeCoercionContext) bool {
	srcParts := strings.Split(src, "|")
	destParts := strings.Split(dest, "|")
	if len(srcParts) != len(destParts) || len(srcParts) != len(table.Columns) {
		return src == dest
	}
	for i, col := range table.Columns {
		pgType := sqliteTypeToPostgres(col, table, all, coerceCtx)
		switch pgType {
		case "JSONB":
			if !jsonValuesEqual(srcParts[i], destParts[i]) {
				return false
			}
		case "BYTEA":
			if !byteaValuesEqual(srcParts[i], destParts[i]) {
				return false
			}
		case "TIMESTAMPTZ":
			if !timestampValuesEqual(srcParts[i], destParts[i]) {
				return false
			}
		default:
			if looksLikeJSON(srcParts[i]) && looksLikeJSON(destParts[i]) {
				if !jsonValuesEqual(srcParts[i], destParts[i]) {
					return false
				}
				continue
			}
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
	if errA == nil && errB == nil {
		return ca == cb
	}
	if errA != nil && errB != nil {
		if looksLikeJSON(a) || looksLikeJSON(b) {
			return false
		}
		return a == b
	}
	return false
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
	if strings.EqualFold(sqliteText, pgText) {
		return true
	}
	if a, okA := decodeByteaSignature(sqliteText); okA {
		if b, okB := decodeByteaSignature(pgText); okB {
			return string(a) == string(b)
		}
	}
	if strings.HasPrefix(pgText, `\x`) {
		decoded, err := hex.DecodeString(strings.TrimPrefix(pgText, `\x`))
		if err == nil && sqliteText == string(decoded) {
			return true
		}
	}
	return false
}

func decodeByteaSignature(s string) ([]byte, bool) {
	s = strings.TrimPrefix(s, `\x`)
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, false
	}
	return b, true
}

func isBlobColumn(col ColumnSchema) bool {
	return strings.Contains(strings.ToUpper(col.Type), "BLOB")
}

func timestampValuesEqual(a, b string) bool {
	if a == b {
		return true
	}
	if a == "" || b == "" {
		return false
	}
	return normalizeTimestamp(a) == normalizeTimestamp(b)
}

func normalizeTimestamp(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Replace(s, " ", "T", 1)
	if strings.HasSuffix(s, "Z") {
		return s
	}
	if idx := strings.LastIndexAny(s, "+-"); idx > 10 {
		return s
	}
	if strings.Contains(s, "T") {
		return s + "Z"
	}
	return s
}

func sqliteRowSignature(ctx context.Context, sqlitePath string, table TableSchema, pkCol, pkVal string, coerceCtx *TypeCoercionContext) (string, error) {
	cols := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		cols = append(cols, sqliteSignatureColumnExpr(col, table, coerceCtx))
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

func postgresRowSignature(ctx context.Context, db *sql.DB, table TableSchema, pkCol, pkVal string, all []TableSchema, coerceCtx *TypeCoercionContext) (string, error) {
	cols := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		cols = append(cols, postgresSignatureColumnExpr(col, table, all, coerceCtx))
	}
	query := fmt.Sprintf(
		`SELECT %s FROM %s WHERE %s = $1 LIMIT 1`,
		strings.Join(cols, " || '|' || "),
		postgres.QuoteIdentifier(table.Name),
		postgres.QuoteIdentifier(pkCol),
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
	if val != "" && isSQLiteIntegerLiteral(val) {
		return val
	}
	return "'" + strings.ReplaceAll(val, "'", "''") + "'"
}

func isSQLiteIntegerLiteral(val string) bool {
	if val == "" || val[0] == '-' {
		if len(val) <= 1 {
			return false
		}
		val = val[1:]
	}
	for i := 0; i < len(val); i++ {
		if val[i] < '0' || val[i] > '9' {
			return false
		}
	}
	return true
}

func runSQLiteQuery(ctx context.Context, sqlite3, sqlitePath, query string) ([]byte, error) {
	cmd := execabs.CommandContext(ctx, sqlite3, sqlitePath, query)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}
