package d1

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	createTableRe   = regexp.MustCompile(`(?is)^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:"([^"]+)"|'([^']+)'|` + "`" + `([^` + "`" + `]+)` + "`" + `|([a-zA-Z_][\w]*))\s*\(`)
	virtualTableRe  = regexp.MustCompile(`(?is)^CREATE\s+VIRTUAL\s+TABLE`)
	autoincrementRe = regexp.MustCompile(`(?i)AUTOINCREMENT`)
	columnUniqueRe  = regexp.MustCompile(`(?i)\bUNIQUE\b`)
	insertRe        = regexp.MustCompile(`(?is)^INSERT\s+INTO\s+(?:` + "`" + `([^` + "`" + `]+)` + "`" + `|"([^"]+)"|'([^']+)'|([a-zA-Z_][\w]*))`)
)

// TableSchema holds parsed SQLite table metadata from a dump file.
type TableSchema struct {
	Name        string
	Columns     []ColumnSchema
	Constraints []string
	RawDDL      string
}

// ColumnSchema holds parsed column metadata.
type ColumnSchema struct {
	Name          string
	Type          string
	PrimaryKey    bool
	AutoIncrement bool
	NotNull       bool
	Unique        bool
	DefaultValue  string
	ForeignKey    string
	// CheckExprs holds raw (unconverted) column-level CHECK expressions, e.g.
	// ["age >= 0"] for `age INTEGER CHECK (age >= 0)`.
	CheckExprs []string
	// GeneratedExpr holds the raw (unconverted) expression of a
	// `GENERATED ALWAYS AS (...)` computed column, empty if the column is not generated.
	GeneratedExpr string
	// GeneratedMode is "STORED" or "VIRTUAL" as declared in the source DDL, empty if
	// GeneratedExpr is empty. SQLite defaults to VIRTUAL when omitted.
	GeneratedMode string
}

// ParseDump reads a SQLite SQL dump and extracts table definitions.
func ParseDump(path string) ([]TableSchema, error) {
	clean, err := ValidateInputPath(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(clean)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var tables []TableSchema
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	var current *TableSchema
	var ddlLines []string
	parenDepth := 0

	flush := func() {
		if current == nil {
			return
		}
		raw := strings.Join(ddlLines, "\n")
		// Truncate at the CREATE TABLE's balanced column-list close so trailing
		// attacker statements on the same line never become part of RawDDL.
		if start := strings.Index(raw, "("); start >= 0 {
			if end, ok := matchingParenEnd(raw, start); ok {
				raw = strings.TrimSpace(raw[:end+1])
				if !strings.HasSuffix(raw, ";") {
					raw += ";"
				}
			}
		}
		current.RawDDL = raw
		current.Columns, current.Constraints = parseTableBody(current.RawDDL)
		tables = append(tables, *current)
		current = nil
		ddlLines = nil
		parenDepth = 0
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		if virtualTableRe.MatchString(line) {
			return nil, newMigrationError(
				ErrCodeVirtualTable,
				"dump contains CREATE VIRTUAL TABLE statements",
				"Remove or recreate FTS5/virtual tables manually in Postgres after migration",
			)
		}

		if current == nil {
			m := createTableRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			name := firstNonEmpty(m[1], m[2], m[3], m[4])
			current = &TableSchema{Name: name}
			ddlLines = append(ddlLines, line)
			parenDepth += strings.Count(line, "(") - strings.Count(line, ")")
			if parenDepth <= 0 && strings.HasSuffix(line, ";") {
				flush()
			}
			continue
		}

		ddlLines = append(ddlLines, line)
		parenDepth += strings.Count(line, "(") - strings.Count(line, ")")
		if parenDepth <= 0 && strings.HasSuffix(line, ";") {
			flush()
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read dump: %w", err)
	}
	flush()

	if len(tables) == 0 {
		return nil, newMigrationError(
			ErrCodeInvalidInput,
			"no CREATE TABLE statements found in dump",
			"Ensure the input is a wrangler d1 export SQL file with schema definitions",
		)
	}

	return tables, nil
}

func parseTableBody(ddl string) ([]ColumnSchema, []string) {
	start := strings.Index(ddl, "(")
	if start < 0 {
		return nil, nil
	}
	// Match the opening paren of the column list — never strings.LastIndex.
	// A dump that smuggles "); DROP ...; CREATE TABLE ..." after the real close
	// would otherwise pull attacker SQL into a column/REFERENCES fragment.
	end, ok := matchingParenEnd(ddl, start)
	if !ok {
		return nil, nil
	}
	body := stripSQLComments(ddl[start+1 : end])
	parts := splitColumnDefs(body)
	cols := make([]ColumnSchema, 0, len(parts))
	var constraints []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if isTableConstraint(part) {
			constraints = append(constraints, part)
			continue
		}
		col := parseColumn(part)
		if col.Name != "" {
			cols = append(cols, col)
		}
	}
	return cols, constraints
}

func parseColumn(def string) ColumnSchema {
	def = strings.TrimSpace(def)
	if def == "" {
		return ColumnSchema{}
	}

	def = strings.TrimSuffix(def, ",")

	name, rest := parseColumnNameAndRest(def)
	if name == "" {
		return ColumnSchema{}
	}

	colType := firstToken(rest)
	constraints := restAfterFirstToken(rest)
	// Preserve precision/scale on NUMERIC/DECIMAL declarations (e.g. "DECIMAL(10,2)") so
	// the Postgres type mapping can produce NUMERIC(10,2) instead of losing it.
	if isNumericAffinityTypeName(colType) {
		if params, remainder, ok := extractLeadingParenGroup(constraints); ok {
			colType += params
			constraints = remainder
		}
	}
	col := ColumnSchema{
		Name: name,
		Type: colType,
	}

	if idx := indexSQLKeyword(constraints, "DEFAULT"); idx >= 0 {
		afterDefault := strings.TrimSpace(constraints[idx+len("DEFAULT"):])
		col.DefaultValue = trimDefaultClause(afterDefault)
		trailing := strings.TrimSpace(afterDefault[len(col.DefaultValue):])
		constraints = strings.TrimSpace(constraints[:idx])
		if trailing != "" {
			constraints = strings.TrimSpace(constraints + " " + trailing)
		}
	}
	constraints, checkExprs := extractCheckClauses(constraints)
	col.CheckExprs = checkExprs

	constraints, generatedExpr, generatedMode := extractGeneratedClause(constraints)
	col.GeneratedExpr = generatedExpr
	col.GeneratedMode = generatedMode

	if indexSQLKeyword(constraints, "NOT NULL") >= 0 {
		col.NotNull = true
	}
	if indexSQLKeyword(constraints, "PRIMARY KEY") >= 0 {
		col.PrimaryKey = true
	}
	if columnUniqueRe.MatchString(constraints) {
		col.Unique = true
	}
	if autoincrementRe.MatchString(rest) {
		col.AutoIncrement = true
	}
	if indexSQLKeyword(rest, "REFERENCES") >= 0 {
		col.ForeignKey = referencesClause(rest)
	}

	return col
}

// foreachDumpStatement invokes fn for each semicolon-terminated SQL statement in a dump.
func foreachDumpStatement(path string, fn func(stmt string) error) error {
	clean, err := ValidateInputPath(path)
	if err != nil {
		return err
	}
	f, err := os.Open(clean)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	var stmt strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		if stmt.Len() > 0 {
			stmt.WriteByte(' ')
		}
		stmt.WriteString(line)
		if !strings.HasSuffix(line, ";") {
			continue
		}
		full := stmt.String()
		stmt.Reset()
		if err := fn(full); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read dump: %w", err)
	}
	return nil
}

// ParseIndexes extracts CREATE INDEX statements from a SQLite dump.
func ParseIndexes(path string) ([]IndexSchema, error) {
	var indexes []IndexSchema
	err := foreachDumpStatement(path, func(full string) error {
		if !strings.HasPrefix(strings.ToUpper(full), "CREATE") {
			return nil
		}
		if !strings.Contains(strings.ToUpper(full), " INDEX ") {
			return nil
		}
		m := createIndexRe.FindStringSubmatch(full)
		if m == nil {
			return nil
		}
		indexes = append(indexes, IndexSchema{
			Name:    firstNonEmpty(m[2], m[3], m[4], m[5]),
			Table:   firstNonEmpty(m[6], m[7], m[8], m[9]),
			Unique:  strings.TrimSpace(m[1]) != "",
			Columns: m[10],
			RawDDL:  full,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return indexes, nil
}

func splitColumnDefs(body string) []string {
	var parts []string
	var current strings.Builder
	depth := 0
	for _, r := range body {
		switch r {
		case '(':
			depth++
			current.WriteRune(r)
		case ')':
			depth--
			current.WriteRune(r)
		case ',':
			if depth == 0 {
				parts = append(parts, current.String())
				current.Reset()
				continue
			}
			current.WriteRune(r)
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// CountInsertRows estimates row counts per table from INSERT statements.
func CountInsertRows(path string) (map[string]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	counts := make(map[string]int)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	var pendingTable string
	var pendingSQL strings.Builder

	flush := func() {
		if pendingTable == "" {
			return
		}
		sql := pendingSQL.String()
		rows := countInsertValueGroups(sql)
		if rows == 0 {
			rows = 1
		}
		counts[pendingTable] += rows
		pendingTable = ""
		pendingSQL.Reset()
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		m := insertRe.FindStringSubmatch(line)
		if m != nil {
			flush()
			pendingTable = firstNonEmpty(m[1], m[2], m[3], m[4])
			pendingSQL.WriteString(line)
			if strings.HasSuffix(line, ";") {
				flush()
			}
			continue
		}

		if pendingTable != "" {
			pendingSQL.WriteString(" ")
			pendingSQL.WriteString(line)
			if strings.HasSuffix(line, ";") {
				flush()
			}
		}
	}
	flush()

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

// FileSize returns the size of a file in bytes.
func FileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func countInsertValueGroups(line string) int {
	_, groups, ok := parseInsertColumnsAndValues(line)
	if !ok || len(groups) == 0 {
		return 0
	}
	return len(groups)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// stripSQLComments removes -- line and /* block */ comments outside quoted strings.
func stripSQLComments(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	inSingle := false
	inDouble := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if inSingle {
			b.WriteByte(c)
			if c == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' {
					b.WriteByte(s[i+1])
					i++
					continue
				}
				inSingle = false
			}
			continue
		}
		if inDouble {
			b.WriteByte(c)
			if c == '"' {
				if i+1 < len(s) && s[i+1] == '"' {
					b.WriteByte(s[i+1])
					i++
					continue
				}
				inDouble = false
			}
			continue
		}

		switch c {
		case '\'':
			inSingle = true
			b.WriteByte(c)
		case '"':
			inDouble = true
			b.WriteByte(c)
		case '-':
			if i+1 < len(s) && s[i+1] == '-' {
				i += 2
				for i < len(s) && s[i] != '\n' {
					i++
				}
				continue
			}
			b.WriteByte(c)
		case '/':
			if i+1 < len(s) && s[i+1] == '*' {
				i += 2
				for i+1 < len(s) && (s[i] != '*' || s[i+1] != '/') {
					i++
				}
				if i+1 < len(s) {
					i++
				}
				continue
			}
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}

	return b.String()
}

func parseColumnNameAndRest(def string) (name, rest string) {
	def = strings.TrimSpace(def)
	def = strings.TrimSuffix(def, ",")
	if def == "" {
		return "", ""
	}

	switch def[0] {
	case '"':
		end := 1
		var raw strings.Builder
		for end < len(def) {
			if def[end] == '"' {
				if end+1 < len(def) && def[end+1] == '"' {
					raw.WriteByte('"')
					end += 2
					continue
				}
				return raw.String(), strings.TrimSpace(def[end+1:])
			}
			raw.WriteByte(def[end])
			end++
		}
		return "", def
	case '[':
		end := strings.Index(def, "]")
		if end <= 1 {
			return "", def
		}
		return def[1:end], strings.TrimSpace(def[end+1:])
	case '`':
		end := strings.Index(def[1:], "`")
		if end < 0 {
			return "", def
		}
		return def[1 : end+1], strings.TrimSpace(def[end+2:])
	case '\'':
		end := 1
		var raw strings.Builder
		for end < len(def) {
			if def[end] == '\'' {
				if end+1 < len(def) && def[end+1] == '\'' {
					raw.WriteByte('\'')
					end += 2
					continue
				}
				return raw.String(), strings.TrimSpace(def[end+1:])
			}
			raw.WriteByte(def[end])
			end++
		}
		return "", def
	default:
		i := 0
		for i < len(def) && !isIdentBreak(def[i]) {
			i++
		}
		if i == 0 {
			return "", def
		}
		return def[:i], strings.TrimSpace(def[i:])
	}
}

func trimDefaultClause(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ",")
	stopPatterns := []string{
		" NOT NULL",
		" NULL",
		" UNIQUE",
		" PRIMARY KEY",
		" REFERENCES",
		" CHECK",
		" COLLATE",
		" GENERATED",
	}
	best := len(s)
	upper := strings.ToUpper(s)
	for _, pat := range stopPatterns {
		if i := indexOutsideQuotes(upper, pat); i >= 0 && i < best {
			best = i
		}
	}
	if best < len(s) {
		s = strings.TrimSpace(s[:best])
	}
	return strings.TrimSuffix(strings.TrimSpace(s), ",")
}

func isNumericAffinityTypeName(t string) bool {
	upper := strings.ToUpper(t)
	return upper == "NUMERIC" || upper == "DECIMAL"
}

// extractLeadingParenGroup extracts a "(...)" group from the start of s (allowing leading
// whitespace before the opening paren), returning the group text, the remainder of s after
// it, and whether a group was found.
func extractLeadingParenGroup(s string) (params, remainder string, ok bool) {
	trimmed := strings.TrimLeft(s, " \t")
	if !strings.HasPrefix(trimmed, "(") {
		return "", s, false
	}
	end, matched := matchingParenEnd(trimmed, 0)
	if !matched {
		return "", s, false
	}
	return trimmed[:end+1], strings.TrimSpace(trimmed[end+1:]), true
}

func restAfterFirstToken(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	i := 0
	for i < len(s) && !isIdentBreak(s[i]) {
		i++
	}
	return strings.TrimSpace(s[i:])
}

func indexOutsideQuotes(s, pattern string) int {
	if pattern == "" {
		return -1
	}
	inQuote := byte(0)
	for i := 0; i+len(pattern) <= len(s); i++ {
		switch {
		case inQuote != 0:
			if s[i] == inQuote {
				inQuote = 0
			}
		case s[i] == '\'' || s[i] == '"':
			inQuote = s[i]
		case strings.EqualFold(s[i:i+len(pattern)], pattern):
			return i
		}
	}
	return -1
}

func indexSQLKeyword(s, keyword string) int {
	if s == "" || keyword == "" {
		return -1
	}
	upper := strings.ToUpper(s)
	kw := strings.ToUpper(keyword)
	for i := 0; i+len(kw) <= len(upper); i++ {
		if upper[i:i+len(kw)] != kw {
			continue
		}
		if i > 0 && isSQLIdentChar(upper[i-1]) {
			continue
		}
		end := i + len(kw)
		if end < len(upper) && isSQLIdentChar(upper[end]) {
			continue
		}
		return i
	}
	return -1
}

func isSQLIdentChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_'
}

// extractCheckClauses removes column-level CHECK(...) clauses from s, returning the
// cleaned string (so remaining constraint keywords like NOT NULL can be detected without
// confusion from text inside the CHECK expression) along with the raw, unconverted
// expressions that were found inside each CHECK(...).
func extractCheckClauses(s string) (string, []string) {
	upper := strings.ToUpper(s)
	var out strings.Builder
	var checks []string
	for i := 0; i < len(s); {
		if strings.HasPrefix(upper[i:], "CHECK") && (i == 0 || !isSQLIdentChar(upper[i-1])) && (i+5 == len(s) || !isSQLIdentChar(upper[i+5])) {
			j := i + 5
			for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
				j++
			}
			if j < len(s) && s[j] == '(' {
				if end, ok := matchingParenEnd(s, j); ok {
					checks = append(checks, strings.TrimSpace(s[j+1:end]))
					i = end + 1
					continue
				}
			}
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String(), checks
}

// generatedAsRe matches the `[GENERATED ALWAYS] AS (` prefix of a computed column
// definition, up to (but not including) the opening paren of the generation expression.
var generatedAsRe = regexp.MustCompile(`(?i)(?:GENERATED\s+ALWAYS\s+)?\bAS\b\s*\(`)

// extractGeneratedClause removes a `[GENERATED ALWAYS] AS (expr) [STORED|VIRTUAL]` computed
// column clause from s, returning the cleaned string along with the raw (unconverted)
// generation expression and its storage mode. Returns s unchanged with an empty expr/mode
// if no generated-column clause is present.
func extractGeneratedClause(s string) (cleaned, expr, mode string) {
	loc := generatedAsRe.FindStringIndex(s)
	if loc == nil {
		return s, "", ""
	}
	openParen := loc[1] - 1
	end, ok := matchingParenEnd(s, openParen)
	if !ok {
		return s, "", ""
	}
	expr = strings.TrimSpace(s[openParen+1 : end])

	rest := s[end+1:]
	trimmedRest := strings.TrimLeft(rest, " \t")
	leadingWS := len(rest) - len(trimmedRest)
	upperTrimmedRest := strings.ToUpper(trimmedRest)
	switch {
	case strings.HasPrefix(upperTrimmedRest, "STORED"):
		mode = "STORED"
		rest = rest[leadingWS+len("STORED"):]
	case strings.HasPrefix(upperTrimmedRest, "VIRTUAL"):
		mode = "VIRTUAL"
		rest = rest[leadingWS+len("VIRTUAL"):]
	default:
		// SQLite defaults to VIRTUAL when the storage mode is omitted.
		mode = "VIRTUAL"
	}
	cleaned = strings.TrimSpace(s[:loc[0]] + " " + rest)
	return cleaned, expr, mode
}

func matchingParenEnd(s string, open int) (int, bool) {
	if open >= len(s) || s[open] != '(' {
		return 0, false
	}
	depth := 0
	inQuote := byte(0)
	for i := open; i < len(s); i++ {
		c := s[i]
		if inQuote != 0 {
			if c == inQuote {
				// Doubled quote ('', "", ``) is the escape for a literal quote.
				// Backslash is not an escape here under standard_conforming_strings
				// (Postgres) or in SQLite string literals.
				if i+1 < len(s) && s[i+1] == inQuote {
					i++
					continue
				}
				inQuote = 0
			}
			continue
		}
		switch c {
		case '\'', '"', '`':
			inQuote = c
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func isIdentBreak(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', '(', ')', ',':
		return true
	default:
		return false
	}
}

func firstToken(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	i := 0
	for i < len(s) && !isIdentBreak(s[i]) {
		i++
	}
	return strings.ToUpper(s[:i])
}
