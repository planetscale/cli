package d1

import (
	"regexp"
	"slices"
	"strings"

	"github.com/planetscale/cli/internal/postgres"
)

var (
	foreignKeyConstraintRe = regexp.MustCompile(`(?is)^FOREIGN\s+KEY\s*\(\s*([^)]+)\)\s*(REFERENCES\s+.+)$`)
	primaryKeyConstraintRe = regexp.MustCompile(`(?is)^PRIMARY\s+KEY\s*\(\s*([^)]+)\)\s*(?:ON\s+CONFLICT\s+\w+)?\s*$`)
	uniqueConstraintRe     = regexp.MustCompile(`(?is)^UNIQUE\s*\(\s*([^)]+)\)\s*(?:ON\s+CONFLICT\s+\w+)?\s*$`)
	createIndexRe          = regexp.MustCompile(`(?is)^CREATE\s+(UNIQUE\s+)?INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:"([^"]+)"|'([^']+)'|` + "`" + `([^` + "`" + `]+)` + "`" + `|([a-zA-Z_][\w]*))\s+ON\s+(?:"([^"]+)"|'([^']+)'|` + "`" + `([^` + "`" + `]+)` + "`" + `|([a-zA-Z_][\w]*))\s*\(\s*([^)]+)\)\s*(?:WHERE\b.+)?;?\s*$`)
	partialIndexRe         = regexp.MustCompile(`(?is)\)\s*WHERE\b`)
)

func isPartialIndexDDL(raw string) bool {
	return partialIndexRe.MatchString(raw)
}

// IndexSchema holds a parsed CREATE INDEX statement from a dump.
type IndexSchema struct {
	Name    string
	Table   string
	Unique  bool
	Columns string
	RawDDL  string
}

func isTableConstraint(part string) bool {
	upper := strings.ToUpper(strings.TrimSpace(part))
	return strings.HasPrefix(upper, "PRIMARY KEY") ||
		strings.HasPrefix(upper, "FOREIGN KEY") ||
		strings.HasPrefix(upper, "UNIQUE(") ||
		strings.HasPrefix(upper, "UNIQUE (") ||
		strings.HasPrefix(upper, "CHECK(") ||
		strings.HasPrefix(upper, "CHECK (") ||
		strings.HasPrefix(upper, "CONSTRAINT ")
}

func referencesClause(colDef string) string {
	idx := indexOfIgnoreCase(colDef, "REFERENCES")
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(colDef[idx:])
}

func convertTableConstraint(clause string, table TableSchema, all []TableSchema, ctx *TypeCoercionContext) string {
	return convertTableConstraintWithOptions(clause, table, all, ctx, tableConvertOptions{})
}

func convertTableConstraintWithOptions(clause string, table TableSchema, all []TableSchema, ctx *TypeCoercionContext, opts tableConvertOptions) string {
	clause = strings.TrimSpace(clause)
	clause = strings.TrimSuffix(clause, ",")
	if clause == "" {
		return ""
	}

	upper := strings.ToUpper(clause)
	switch {
	case strings.HasPrefix(upper, "CONSTRAINT "):
		return convertNamedConstraintWithOptions(clause, table, all, ctx, opts)
	case strings.HasPrefix(upper, "FOREIGN KEY"):
		if opts.omitForeignKeys {
			return ""
		}
		return convertForeignKeyConstraint(clause, table, all)
	case strings.HasPrefix(upper, "PRIMARY KEY"):
		return convertPrimaryKeyConstraint(clause, table)
	case strings.HasPrefix(upper, "UNIQUE"):
		return convertUniqueConstraint(clause, table)
	case strings.HasPrefix(upper, "CHECK"):
		return convertCheckConstraint(clause, table, ctx)
	default:
		return clause
	}
}

// convertNamedConstraintWithOptions converts a `CONSTRAINT <name> <body>` clause by re-quoting the
// constraint name and running the body through the same conversion as unnamed constraints,
// so named constraints get identical quoting/canonicalization fixes.
func convertNamedConstraintWithOptions(clause string, table TableSchema, all []TableSchema, ctx *TypeCoercionContext, opts tableConvertOptions) string {
	rest := strings.TrimSpace(clause[len("CONSTRAINT"):])
	name, body := parseColumnNameAndRest(rest)
	if name == "" || body == "" {
		return clause
	}
	converted := convertTableConstraintWithOptions(body, table, all, ctx, opts)
	if converted == "" {
		return ""
	}
	return "CONSTRAINT " + postgres.QuoteIdentifier(name) + " " + converted
}

// convertCheckConstraint converts a table-level `CHECK (expr)` clause, re-quoting any
// identifiers inside expr that reference the table's columns (so mixed-case column names
// survive Postgres's case-folding of unquoted identifiers) and converting SQLite's
// double-quoted string-literal fallback into proper single-quoted literals.
func convertCheckConstraint(clause string, table TableSchema, ctx *TypeCoercionContext) string {
	clause = strings.TrimSpace(clause)
	clause = strings.TrimSuffix(clause, ",")

	upper := strings.ToUpper(clause)
	if !strings.HasPrefix(upper, "CHECK") {
		return clause
	}
	rest := strings.TrimSpace(clause[len("CHECK"):])
	if !strings.HasPrefix(rest, "(") {
		return clause
	}
	end, ok := matchingParenEnd(rest, 0)
	if !ok {
		return clause
	}
	expr := rest[1:end]
	return "CHECK (" + convertCheckExpr(expr, table, ctx) + ")"
}

// checkExprKeywords are SQL keywords that can legitimately appear as bare words inside a
// CHECK/GENERATED expression. They are never treated as column references, even when the
// table has a column with the same name, since quoting them would corrupt the expression
// (e.g. `a > 0 AND b < 5` must not become `"a" > 0 "and" "b" < 5`). The tradeoff is that a
// column named after one of these keywords won't be case-canonicalized when referenced
// bare — such references must already be quoted in the source DDL to be unambiguous.
var checkExprKeywords = map[string]struct{}{
	"and": {}, "or": {}, "not": {}, "in": {}, "is": {}, "null": {},
	"like": {}, "glob": {}, "regexp": {}, "match": {}, "escape": {},
	"between": {}, "case": {}, "when": {}, "then": {}, "else": {}, "end": {},
	"cast": {}, "as": {}, "exists": {}, "distinct": {}, "collate": {},
	"isnull": {}, "notnull": {}, "true": {}, "false": {},
	"current_time": {}, "current_date": {}, "current_timestamp": {},
}

// convertCheckExpr rewrites identifiers and string literals inside a CHECK/GENERATED
// expression for Postgres:
//   - bare or double-quoted tokens that match one of the table's column names (case
//     insensitively) are re-quoted using the column's declared case, so Postgres's
//     case-folding of unquoted identifiers can't cause a "column does not exist" error.
//     Bare tokens that are SQL keywords (see checkExprKeywords) are never treated as
//     column references.
//   - double-quoted tokens that do NOT match a column name are treated as SQLite's
//     double-quoted string-literal fallback and converted to proper single-quoted
//     string literals (Postgres always treats double quotes as identifiers).
//   - bracket-quoted ([col]) and backtick-quoted identifiers — both valid SQLite quoting
//     that Postgres rejects — are converted to double-quoted identifiers, canonicalized
//     to the column's declared case when they match one.
//   - single-quoted string literals and everything else are passed through unchanged.
//   - 0/1 values compared with coerced BOOLEAN columns become false/true.
func convertCheckExpr(expr string, table TableSchema, ctx *TypeCoercionContext) string {
	colMap := make(map[string]string, len(table.Columns))
	for _, col := range table.Columns {
		colMap[strings.ToLower(col.Name)] = col.Name
	}

	var out strings.Builder
	n := len(expr)
	for i := 0; i < n; {
		c := expr[i]
		switch {
		case c == '\'':
			j := i + 1
			for j < n {
				if expr[j] == '\'' {
					if j+1 < n && expr[j+1] == '\'' {
						j += 2
						continue
					}
					j++
					break
				}
				j++
			}
			out.WriteString(expr[i:j])
			i = j
		case c == '"':
			j := i + 1
			var raw strings.Builder
			for j < n {
				if expr[j] == '"' {
					if j+1 < n && expr[j+1] == '"' {
						raw.WriteByte('"')
						j += 2
						continue
					}
					j++
					break
				}
				raw.WriteByte(expr[j])
				j++
			}
			inner := raw.String()
			if actual, ok := colMap[strings.ToLower(inner)]; ok {
				out.WriteString(postgres.QuoteIdentifier(actual))
			} else {
				out.WriteString(quotePostgresLiteral(inner))
			}
			i = j
		case c == '[':
			end := strings.IndexByte(expr[i+1:], ']')
			if end < 0 {
				out.WriteString(expr[i:])
				i = n
				break
			}
			inner := expr[i+1 : i+1+end]
			out.WriteString(postgres.QuoteIdentifier(canonicalIdent(inner, colMap)))
			i = i + 1 + end + 1
		case c == '`':
			j := i + 1
			var raw strings.Builder
			for j < n {
				if expr[j] == '`' {
					if j+1 < n && expr[j+1] == '`' {
						raw.WriteByte('`')
						j += 2
						continue
					}
					j++
					break
				}
				raw.WriteByte(expr[j])
				j++
			}
			out.WriteString(postgres.QuoteIdentifier(canonicalIdent(raw.String(), colMap)))
			i = j
		case isIdentStartByte(c):
			j := i + 1
			for j < n && isSQLIdentChar(expr[j]) {
				j++
			}
			word := expr[i:j]
			k := j
			for k < n && (expr[k] == ' ' || expr[k] == '\t') {
				k++
			}
			isFunctionCall := k < n && expr[k] == '('
			_, isKeyword := checkExprKeywords[strings.ToLower(word)]
			if actual, ok := colMap[strings.ToLower(word)]; ok && !isFunctionCall && !isKeyword {
				out.WriteString(postgres.QuoteIdentifier(actual))
			} else {
				out.WriteString(word)
			}
			i = j
		default:
			out.WriteByte(c)
			i++
		}
	}
	result := out.String()
	if boolCols := booleanCoercedColumnNames(table, ctx); len(boolCols) > 0 {
		result = rewriteBooleanCheckLiterals(result, boolCols)
	}
	return result
}

func isIdentStartByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

// canonicalIdent returns the declared-case column name for ident when it matches one of
// the table's columns (colMap keys are lower-cased declared names), or ident unchanged.
func canonicalIdent(ident string, colMap map[string]string) string {
	if actual, ok := colMap[strings.ToLower(ident)]; ok {
		return actual
	}
	return ident
}

func booleanCoercedColumnNames(table TableSchema, ctx *TypeCoercionContext) []string {
	if ctx == nil {
		return nil
	}
	var cols []string
	for _, col := range table.Columns {
		if isBooleanLikeColumn(col, table, ctx) {
			cols = append(cols, col.Name)
		}
	}
	return cols
}

// rewriteBooleanCheckLiterals runs only after column names have been normalized and quoted.
func rewriteBooleanCheckLiterals(expr string, boolCols []string) string {
	idents := make([]string, len(boolCols))
	for i, col := range boolCols {
		idents[i] = regexp.QuoteMeta(postgres.QuoteIdentifier(col))
	}
	ident := `(?:` + strings.Join(idents, `|`) + `)`
	operandStart := `(^|[(,]\s*|(?i:\b(?:AND|OR)\b)\s*)`
	operandEnd := `(\s*(?:(?i:AND|OR)\b|[),]|$))`
	literals := []struct{ from, to string }{{"0", "false"}, {"1", "true"}}
	left := make([]*regexp.Regexp, len(literals))
	right := make([]*regexp.Regexp, len(literals))
	isLeft := make([]*regexp.Regexp, len(literals))
	isRight := make([]*regexp.Regexp, len(literals))
	equalLeft := make([]*regexp.Regexp, len(literals))
	equalRight := make([]*regexp.Regexp, len(literals))
	for i, literal := range literals {
		left[i] = regexp.MustCompile(`(` + ident + `\s*(?:=|<>|!=|<=|>=|<|>)\s*)` + literal.from + operandEnd)
		right[i] = regexp.MustCompile(operandStart + literal.from + `(\s*(?:=|<>|!=|<=|>=|<|>)\s*` + ident + `)`)
		isLeft[i] = regexp.MustCompile(`(` + ident + `\s+(?i:IS(?:\s+NOT)?(?:\s+DISTINCT\s+FROM)?)\s*)` + literal.from + operandEnd)
		isRight[i] = regexp.MustCompile(operandStart + literal.from + `\s+(?i:(IS(?:\s+NOT)?(?:\s+DISTINCT\s+FROM)?))\s*(` + ident + `)`)
		equalLeft[i] = regexp.MustCompile(`(` + ident + `\s*)==(\s*)` + literal.from + operandEnd)
		equalRight[i] = regexp.MustCompile(operandStart + literal.from + `(\s*)==(\s*` + ident + `)`)
	}
	in := regexp.MustCompile(`(` + ident + `\s+(?i:(?:NOT\s+)?IN)\s*\()(\s*[01](?:\s*,\s*[01])*\s*)(\))`)
	between := regexp.MustCompile(`(` + ident + `\s+(?i:(?:NOT\s+)?BETWEEN)\s+)([01])(\s+(?i:AND)\s+)([01])` + operandEnd)
	replaceList := strings.NewReplacer("0", "false", "1", "true")

	return rewriteOutsideStringLiterals(expr, func(sql string) string {
		for i, literal := range literals {
			sql = left[i].ReplaceAllString(sql, `${1}`+literal.to+`${2}`)
			sql = right[i].ReplaceAllString(sql, `${1}`+literal.to+`${2}`)
			sql = isLeft[i].ReplaceAllString(sql, `${1}`+literal.to+`${2}`)
			sql = isRight[i].ReplaceAllString(sql, `${1}${3} ${2} `+literal.to)
			sql = equalLeft[i].ReplaceAllString(sql, `${1}=${2}`+literal.to+`${3}`)
			sql = equalRight[i].ReplaceAllString(sql, `${1}`+literal.to+`${2}=${3}`)
		}
		sql = in.ReplaceAllStringFunc(sql, func(match string) string {
			parts := in.FindStringSubmatch(match)
			return parts[1] + replaceList.Replace(parts[2]) + parts[3]
		})
		return between.ReplaceAllStringFunc(sql, func(match string) string {
			parts := between.FindStringSubmatch(match)
			return parts[1] + replaceList.Replace(parts[2]) + parts[3] + replaceList.Replace(parts[4]) + parts[5]
		})
	})
}

// rewriteOutsideStringLiterals leaves quoted text alone while rewriting SQL around it.
func rewriteOutsideStringLiterals(expr string, rewrite func(string) string) string {
	var out strings.Builder
	for len(expr) > 0 {
		start, end := nextStringLiteral(expr)
		if start < 0 {
			out.WriteString(rewrite(expr))
			break
		}
		out.WriteString(rewrite(expr[:start]))
		out.WriteString(expr[start:end])
		expr = expr[end:]
	}
	return out.String()
}

func nextStringLiteral(expr string) (int, int) {
	for i := 0; i < len(expr); i++ {
		switch expr[i] {
		case '"':
			i = quotedEnd(expr, i, '"') - 1
		case '\'':
			return i, quotedEnd(expr, i, '\'')
		}
	}
	return -1, -1
}

func quotedEnd(s string, start int, quote byte) int {
	for i := start + 1; i < len(s); i++ {
		if s[i] != quote {
			continue
		}
		if i+1 < len(s) && s[i+1] == quote {
			i++
			continue
		}
		return i + 1
	}
	return len(s)
}

func convertForeignKeyConstraint(clause string, table TableSchema, all []TableSchema) string {
	m := foreignKeyConstraintRe.FindStringSubmatch(clause)
	if m == nil {
		return clause
	}
	cols := quoteColumnListFor(m[1], &table)
	refs := convertReferencesClause(strings.TrimSpace(m[2]), all)
	if cols == "" || refs == "" {
		return ""
	}
	return "FOREIGN KEY (" + cols + ") " + refs
}

func convertPrimaryKeyConstraint(clause string, table TableSchema) string {
	m := primaryKeyConstraintRe.FindStringSubmatch(clause)
	if m == nil {
		return clause
	}
	return "PRIMARY KEY (" + quoteColumnListFor(m[1], &table) + ")"
}

func convertUniqueConstraint(clause string, table TableSchema) string {
	m := uniqueConstraintRe.FindStringSubmatch(clause)
	if m == nil {
		return clause
	}
	return "UNIQUE (" + quoteColumnListFor(m[1], &table) + ")"
}

// convertReferencesClause converts a `REFERENCES table(col, ...) [tail]` clause. SQLite
// resolves table/column names in REFERENCES case-insensitively, but Postgres compares
// quoted identifiers case-sensitively, so the referenced table/columns are canonicalized
// to the actual declared case from all (the full set of parsed tables) rather than quoting
// whatever case happens to appear in this clause.
//
// Parsing matches parseReferencedTableName: bracket-, quote-, backtick-, and
// schema-qualified identifiers are accepted. Unrecognized REFERENCES text and unsafe
// action tails (statement terminators, extra parentheses, or tokens outside the FK-action
// grammar) are dropped rather than emitted verbatim into executed DDL.
func convertReferencesClause(refs string, all []TableSchema) string {
	rawTable, colList, tail, ok := parseReferencesParts(refs)
	if !ok {
		return ""
	}
	refTable := tableByName(all, rawTable)
	tableName := rawTable
	if refTable != nil {
		tableName = refTable.Name
	}
	if strings.TrimSpace(colList) == "" {
		if refTable == nil {
			return ""
		}
		pks := primaryKeyColumns(*refTable)
		if len(pks) == 0 {
			return ""
		}
		colList = strings.Join(pks, ", ")
	}
	refCols := quoteColumnListFor(colList, refTable)
	if refCols == "" {
		return ""
	}
	out := "REFERENCES " + postgres.QuoteIdentifier(tableName) + " (" + refCols + ")"
	if safe := sanitizeReferencesTail(tail); safe != "" {
		return out + " " + safe
	}
	return out
}

// parseReferencesParts extracts the referenced table, column list, and action tail from a
// REFERENCES clause (or a larger string containing one). The table name is returned without
// any schema qualifier.
func parseReferencesParts(refs string) (table, colList, tail string, ok bool) {
	refs = strings.TrimSpace(refs)
	if refs == "" {
		return "", "", "", false
	}
	if idx := indexOfIgnoreCase(refs, "REFERENCES"); idx >= 0 {
		refs = strings.TrimSpace(refs[idx+len("REFERENCES"):])
	} else {
		return "", "", "", false
	}

	rawTable, rest := parseQualifiedTableRef(refs)
	if rawTable == "" {
		return "", "", "", false
	}

	params, remainder, found := extractLeadingParenGroup(rest)
	if found && len(params) >= 2 {
		return rawTable, params[1 : len(params)-1], strings.TrimSpace(remainder), true
	}
	// SQLite allows REFERENCES parent with no column list (defaults to the parent's PK).
	return rawTable, "", strings.TrimSpace(rest), true
}

// parseQualifiedTableRef parses table or schema.table from the start of s and returns the
// table name only (schema stripped when present).
//
//   - Bare schema.table is one token ('.' is not an identifier break); the segment after the
//     last '.' is treated as the table name.
//   - Quoted/bracketed schema.table is two tokens joined by '.'; only the second token is kept.
//   - A single quoted/bracketed identifier that itself contains '.' (e.g. "my.table") is kept
//     intact — the dot is part of the table name, not a schema separator.
func parseQualifiedTableRef(s string) (name, rest string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", s
	}
	quoted := s[0] == '"' || s[0] == '[' || s[0] == '`' || s[0] == '\''
	first, rest := parseColumnNameAndRest(s)
	if first == "" {
		return "", s
	}
	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, ".") {
		second, rest2 := parseColumnNameAndRest(strings.TrimSpace(rest[1:]))
		if second != "" {
			return second, rest2
		}
	}
	if !quoted {
		if dot := strings.LastIndex(first, "."); dot >= 0 {
			// Re-parse the segment after the last '.' so bare schema + quoted/bracketed
			// table (main."Parent", main.[Parent]) yields Parent, not quote characters.
			segment := first[dot+1:]
			if name, _ := parseColumnNameAndRest(segment); name != "" {
				return name, rest
			}
			return segment, rest
		}
	}
	return first, rest
}

// fkActionTailRe matches a sequence of SQLite/Postgres foreign-key action clauses only.
// Anything outside this grammar (including ";" / extra ")" that could escape CREATE TABLE)
// is rejected by sanitizeReferencesTail.
var fkActionTailRe = regexp.MustCompile(`(?is)^(?:\s*(?:ON\s+(?:DELETE|UPDATE)\s+(?:CASCADE|SET\s+NULL|SET\s+DEFAULT|RESTRICT|NO\s+ACTION)|MATCH\s+(?:SIMPLE|FULL|PARTIAL)|NOT\s+DEFERRABLE|DEFERRABLE(?:\s+INITIALLY\s+(?:DEFERRED|IMMEDIATE))?|INITIALLY\s+(?:DEFERRED|IMMEDIATE)))+$`)

// sanitizeReferencesTail returns tail when it is a safe FK action clause list, otherwise "".
func sanitizeReferencesTail(tail string) string {
	tail = strings.TrimSpace(tail)
	if tail == "" {
		return ""
	}
	if strings.ContainsAny(tail, ";()") {
		return ""
	}
	if !fkActionTailRe.MatchString(tail) {
		return ""
	}
	return strings.Join(strings.Fields(tail), " ")
}

// quoteColumnList quotes a comma-separated column list, stripping SQLite indexed-column
// modifiers (COLLATE, ASC, DESC) that cannot appear in this position in Postgres.
func quoteColumnList(list string) string {
	return quoteColumnListFor(list, nil)
}

// quoteColumnListFor is like quoteColumnList but additionally canonicalizes each column's
// case against table's declared columns when table is non-nil.
func quoteColumnListFor(list string, table *TableSchema) string {
	parts := splitCommaList(list)
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		name := cleanIndexedColumnName(part)
		if name == "" {
			continue
		}
		quoted = append(quoted, postgres.QuoteIdentifier(canonicalColumnName(name, table)))
	}
	return strings.Join(quoted, ", ")
}

// cleanIndexedColumnName extracts the bare column name from a column-list entry that may
// carry SQLite indexed-column modifiers, e.g. "b DESC" -> "b", `"MixedCase" COLLATE NOCASE
// ASC` -> "MixedCase".
func cleanIndexedColumnName(part string) string {
	part = strings.TrimSpace(part)
	if part == "" {
		return ""
	}
	name, _ := parseColumnNameAndRest(part)
	if name == "" {
		return strings.Trim(part, "`\"'")
	}
	return name
}

// canonicalColumnName looks up name (case-insensitively) among table's declared columns
// and returns the declared case, or name unchanged if table is nil or has no match.
func canonicalColumnName(name string, table *TableSchema) string {
	if table == nil {
		return name
	}
	for _, col := range table.Columns {
		if strings.EqualFold(col.Name, name) {
			return col.Name
		}
	}
	return name
}

func splitCommaList(list string) []string {
	var parts []string
	var current strings.Builder
	depth := 0
	inSingle := false
	inDouble := false

	for _, r := range list {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
			current.WriteRune(r)
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
			current.WriteRune(r)
		case '(':
			if !inSingle && !inDouble {
				depth++
			}
			current.WriteRune(r)
		case ')':
			if !inSingle && !inDouble {
				depth--
			}
			current.WriteRune(r)
		case ',':
			if depth == 0 && !inSingle && !inDouble {
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

func convertIndexDDL(raw string) string {
	if isPartialIndexDDL(raw) {
		return ""
	}
	m := createIndexRe.FindStringSubmatch(raw)
	if m == nil {
		return ""
	}
	if indexColumnsLookExpression(m[10]) {
		return ""
	}
	unique := strings.TrimSpace(m[1]) != ""
	name := postgres.QuoteIdentifier(firstNonEmpty(m[2], m[3], m[4], m[5]))
	table := postgres.QuoteIdentifier(firstNonEmpty(m[6], m[7], m[8], m[9]))
	cols := quoteColumnList(m[10])
	prefix := "CREATE INDEX IF NOT EXISTS "
	if unique {
		prefix = "CREATE UNIQUE INDEX IF NOT EXISTS "
	}
	return prefix + name + " ON " + table + " (" + cols + ");"
}

func isUUIDColumn(col ColumnSchema, table TableSchema, all []TableSchema, ctx *TypeCoercionContext) bool {
	if isExplicitUUIDColumn(col) {
		if ctx == nil {
			return false
		}
		return samplesAllowUUID(table.Name, col.Name, ctx)
	}
	return columnReferencesUUIDKey(col, table, all, ctx)
}

const maxUUIDFKDepth = 32

func columnReferencesUUIDKey(col ColumnSchema, table TableSchema, all []TableSchema, ctx *TypeCoercionContext) bool {
	visited := make(map[string]struct{})
	return columnReferencesUUIDKeyVisited(col, table, all, ctx, visited, 0)
}

func columnReferencesUUIDKeyVisited(col ColumnSchema, table TableSchema, all []TableSchema, ctx *TypeCoercionContext, visited map[string]struct{}, depth int) bool {
	if depth >= maxUUIDFKDepth {
		return false
	}
	key := table.Name + "." + col.Name
	if _, seen := visited[key]; seen {
		return false
	}
	visited[key] = struct{}{}

	refTable, refCol := columnFKTarget(col, table, all)
	if refTable == "" || refCol == "" {
		return false
	}
	ref := tableByName(all, refTable)
	if ref == nil {
		return false
	}
	refColSchema := columnByName(*ref, refCol)
	if isExplicitUUIDColumn(refColSchema) {
		if ctx == nil {
			return false
		}
		return samplesAllowUUID(ref.Name, refColSchema.Name, ctx)
	}
	return columnReferencesUUIDKeyVisited(refColSchema, *ref, all, ctx, visited, depth+1)
}

func isExplicitUUIDColumn(col ColumnSchema) bool {
	name := strings.ToLower(col.Name)
	t := strings.ToUpper(col.Type)

	if !isTextLikeType(t) {
		return false
	}

	if col.PrimaryKey && (name == "id" || name == "uuid") {
		return true
	}
	if strings.HasSuffix(name, "_uuid") {
		return true
	}
	return false
}

func columnFKTarget(col ColumnSchema, table TableSchema, all []TableSchema) (string, string) {
	if col.ForeignKey != "" {
		return parseReferencesTarget(col.ForeignKey, all)
	}
	for _, constraint := range table.Constraints {
		cols, refs := parseTableLevelForeignKey(constraint)
		pos := slices.Index(cols, col.Name)
		if pos < 0 {
			continue
		}
		// Map the local column to the referenced column at the same position, so each
		// part of a composite FK inherits the correct parent column (and type) rather
		// than always the first one.
		return parseReferencesTargetAt(refs, pos, all)
	}
	return "", ""
}

func parseTableLevelForeignKey(constraint string) ([]string, string) {
	m := foreignKeyConstraintRe.FindStringSubmatch(constraint)
	if m == nil {
		return nil, ""
	}
	cols := make([]string, 0)
	for _, part := range splitCommaList(m[1]) {
		part = strings.Trim(strings.TrimSpace(part), "`\"'")
		if part != "" {
			cols = append(cols, part)
		}
	}
	return cols, strings.TrimSpace(m[2])
}

func parseReferencesTarget(refs string, all []TableSchema) (string, string) {
	return parseReferencesTargetAt(refs, 0, all)
}

// parseReferencesTargetAt resolves the referenced table and the referenced column at
// position pos. When the REFERENCES clause omits its column list, it defaults to the
// parent primary key column at the same position (SQLite matches PK columns positionally).
func parseReferencesTargetAt(refs string, pos int, all []TableSchema) (string, string) {
	table, colList, _, ok := parseReferencesParts(refs)
	if !ok {
		return "", ""
	}
	refCol := ""
	if cols := splitCommaList(colList); pos < len(cols) {
		refCol = cleanIndexedColumnName(cols[pos])
	}
	if refCol == "" && strings.TrimSpace(colList) == "" && len(all) > 0 {
		if ref := tableByName(all, table); ref != nil {
			if pks := primaryKeyColumns(*ref); pos < len(pks) {
				refCol = pks[pos]
			}
		}
	}
	return table, refCol
}

func tableByName(all []TableSchema, name string) *TableSchema {
	lower := strings.ToLower(name)
	for i := range all {
		if strings.ToLower(all[i].Name) == lower {
			return &all[i]
		}
	}
	return nil
}

func columnByName(table TableSchema, name string) ColumnSchema {
	lower := strings.ToLower(name)
	for _, col := range table.Columns {
		if strings.ToLower(col.Name) == lower {
			return col
		}
	}
	return ColumnSchema{}
}

func isTextLikeType(t string) bool {
	return t == "" || strings.Contains(t, "CHAR") || strings.Contains(t, "CLOB") || strings.Contains(t, "TEXT")
}
