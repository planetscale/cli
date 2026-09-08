package d1

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// sqliteOnlyCheckFuncs are SQLite built-ins that Postgres does not provide under the
// same name. convertCheckExpr rewrites the ones we can translate; lint flags the rest
// so import fails with a clear message instead of a psql "function does not exist".
var sqliteOnlyCheckFuncs = map[string]struct{}{
	"changes": {}, "char": {}, "glob": {}, "hex": {}, "ifnull": {}, "iif": {},
	"instr": {}, "json": {}, "jsonb": {}, "json_error_position": {},
	"json_extract": {}, "json_group_array": {}, "json_group_object": {},
	"json_insert": {}, "json_patch": {}, "json_pretty": {}, "json_quote": {},
	"json_remove": {}, "json_replace": {}, "json_set": {}, "json_type": {},
	"json_valid": {}, "julianday": {}, "last_insert_rowid": {},
	"likelihood": {}, "likely": {}, "load_extension": {}, "printf": {},
	"quote": {}, "randomblob": {}, "soundex": {}, "sqlite_compileoption_get": {},
	"sqlite_compileoption_used": {}, "sqlite_offset": {}, "sqlite_source_id": {},
	"sqlite_version": {}, "strftime": {}, "timediff": {}, "total_changes": {},
	"typeof": {}, "unhex": {}, "unicode": {}, "unlikely": {}, "unixepoch": {},
	"zeroblob": {}, "datetime": {}, "json_array_length": {},
}

var doubleEqRe = regexp.MustCompile(`==`)

// rewriteSQLiteCheckFunctions maps SQLite-only function calls and a few operators
// inside an already identifier-rewritten CHECK/GENERATED expression.
func rewriteSQLiteCheckFunctions(expr string) string {
	expr = rewriteFunctionCalls(expr)
	expr = rewritePrefixJSONValidCompares(expr)
	expr = rewriteGlobAndRegexpOperators(expr)
	return rewriteOutsideStringLiterals(expr, func(sql string) string {
		return doubleEqRe.ReplaceAllString(sql, "=")
	})
}

func rewriteFunctionCalls(expr string) string {
	var out strings.Builder
	n := len(expr)
	for i := 0; i < n; {
		c := expr[i]
		switch {
		case c == '\'':
			j := quotedEnd(expr, i, '\'')
			out.WriteString(expr[i:j])
			i = j
		case c == '"':
			j := quotedEnd(expr, i, '"')
			out.WriteString(expr[i:j])
			i = j
		case isIdentStartByte(c):
			j := i + 1
			for j < n && isSQLIdentChar(expr[j]) {
				j++
			}
			name := expr[i:j]
			k := j
			for k < n && (expr[k] == ' ' || expr[k] == '\t') {
				k++
			}
			if k < n && expr[k] == '(' {
				if _, isKeyword := checkExprKeywords[strings.ToLower(name)]; isKeyword {
					out.WriteString(name)
					i = j
					continue
				}
				end, ok := matchingParenEnd(expr, k)
				if !ok {
					out.WriteString(expr[i:])
					return out.String()
				}
				args := rewriteFunctionCalls(expr[k+1 : end])
				if mapped, converted := mapSQLiteCheckFunction(name, args); converted {
					if strings.EqualFold(name, "json_valid") {
						if consumed, notJSON, ok := consumeJSONValidCompare(expr[end+1:]); ok {
							if notJSON {
								mapped = strings.Replace(mapped, " IS JSON", " IS NOT JSON", 1)
							}
							end += consumed
						}
					}
					out.WriteString(mapped)
				} else {
					out.WriteString(name)
					out.WriteByte('(')
					out.WriteString(args)
					out.WriteByte(')')
				}
				i = end + 1
				continue
			}
			out.WriteString(name)
			i = j
		default:
			out.WriteByte(c)
			i++
		}
	}
	return out.String()
}

// mapSQLiteCheckFunction translates one SQLite function call. converted is true only
// when the result is valid Postgres. Unknown names return converted=false so callers
// pass the original call through (length, coalesce, …).
func mapSQLiteCheckFunction(name, args string) (mapped string, converted bool) {
	switch strings.ToLower(name) {
	case "json_valid":
		parts := splitFunctionArgs(args)
		if len(parts) >= 1 && parts[0] != "" {
			return "(" + parts[0] + ") IS JSON", true
		}
	case "ifnull":
		parts := splitFunctionArgs(args)
		if len(parts) == 2 {
			return "coalesce(" + parts[0] + ", " + parts[1] + ")", true
		}
	case "iif":
		parts := splitFunctionArgs(args)
		switch len(parts) {
		case 2:
			return "CASE WHEN (" + parts[0] + ") THEN (" + parts[1] + ") ELSE NULL END", true
		case 3:
			return "CASE WHEN (" + parts[0] + ") THEN (" + parts[1] + ") ELSE (" + parts[2] + ") END", true
		}
	case "instr":
		parts := splitFunctionArgs(args)
		if len(parts) == 2 {
			return "strpos(" + parts[0] + ", " + parts[1] + ")", true
		}
	case "likely", "unlikely":
		if strings.TrimSpace(args) != "" {
			return "(" + args + ")", true
		}
	case "likelihood":
		parts := splitFunctionArgs(args)
		if len(parts) >= 1 && parts[0] != "" {
			return "(" + parts[0] + ")", true
		}
	case "hex":
		if strings.TrimSpace(args) != "" {
			return "upper(encode(convert_to((" + args + ")::text, 'UTF8'), 'hex'))", true
		}
	case "unhex":
		parts := splitFunctionArgs(args)
		if len(parts) == 1 && parts[0] != "" {
			return "decode((" + parts[0] + "), 'hex')", true
		}
	case "quote":
		if strings.TrimSpace(args) != "" {
			return "quote_literal(" + args + ")", true
		}
	case "unicode":
		if strings.TrimSpace(args) != "" {
			return "ascii(" + args + ")", true
		}
	case "char":
		parts := splitFunctionArgs(args)
		if len(parts) == 0 {
			break
		}
		chrs := make([]string, len(parts))
		for i, p := range parts {
			chrs[i] = "chr(" + p + ")"
		}
		return strings.Join(chrs, " || "), true
	case "typeof":
		if strings.TrimSpace(args) != "" {
			return sqliteTypeofExpr(args), true
		}
	case "json":
		if strings.TrimSpace(args) != "" {
			return "((" + args + ")::json)", true
		}
	case "jsonb":
		if strings.TrimSpace(args) != "" {
			return "((" + args + ")::jsonb)", true
		}
	case "json_extract":
		if mapped, ok := mapJSONExtract(args); ok {
			return mapped, true
		}
	case "json_array_length":
		parts := splitFunctionArgs(args)
		switch len(parts) {
		case 1:
			return "json_array_length((" + parts[0] + ")::json)", true
		case 2:
			if path, ok := sqliteJSONPathLiteral(parts[1]); ok {
				return "jsonb_array_length(((" + parts[0] + ")::jsonb #> " + path + "))", true
			}
		}
	case "json_pretty":
		if strings.TrimSpace(args) != "" {
			return "jsonb_pretty((" + args + ")::jsonb)", true
		}
	case "json_quote":
		if strings.TrimSpace(args) != "" {
			return "to_jsonb(" + args + ")", true
		}
	case "json_array":
		return "json_build_array(" + args + ")", true
	case "json_object":
		return "json_build_object(" + args + ")", true
	case "randomblob":
		if n := randomblobArgN("randomblob(" + args + ")"); n > 0 {
			return randomBytesExpr(n), true
		}
	case "zeroblob":
		if n, err := parsePositiveIntArg(args); err == nil && n > 0 {
			return fmt.Sprintf("decode(repeat('00', %d), 'hex')", n), true
		}
	case "date", "time", "datetime", "julianday", "unixepoch", "strftime":
		return mapSQLiteCheckDateFunction(name, args)
	}
	return "", false
}

// mapSQLiteCheckDateFunction translates SQLite date/time calls only when they
// mean "current time". The DEFAULT mapper cannot be reused here: it maps any
// datetime(...) / date(...) / time(...) to now()/CURRENT_DATE/CURRENT_TIME.
func mapSQLiteCheckDateFunction(name, args string) (string, bool) {
	parts := splitFunctionArgs(args)
	switch strings.ToLower(name) {
	case "strftime":
		if mapped := mapSQLiteDefaultFunction("strftime("+args+")", "TIMESTAMPTZ"); mapped != "" {
			return mapped, true
		}
		return "", false
	case "unixepoch":
		arg := ""
		if len(parts) > 0 {
			arg = parts[0]
		}
		if len(parts) > 1 {
			return "", false
		}
		modifier := strings.ToUpper(strings.Trim(strings.TrimSpace(arg), `'"`))
		if modifier != "" && modifier != "NOW" && modifier != "SUBSEC" {
			return "", false
		}
		if mapped := mapUnixEpochDefault(arg, "TIMESTAMPTZ"); mapped != "" {
			return mapped, true
		}
		return "", false
	case "julianday":
		arg := ""
		if len(parts) > 0 {
			arg = parts[0]
		}
		if len(parts) > 1 || !isSQLiteCurrentTimeValue(arg) {
			return "", false
		}
		return "(extract(epoch from now()) / 86400.0 + 2440587.5)", true
	case "datetime", "date", "time":
		timeArg := ""
		if len(parts) > 0 {
			timeArg = parts[0]
		}
		if !isSQLiteCurrentTimeValue(timeArg) {
			return "", false
		}
		base, ok := strftimeTimeValueExpr(currentTimeArgOrNow(timeArg))
		if !ok {
			return "", false
		}
		if len(parts) > 1 {
			base, ok = applyStrftimeModifiers(base, parts[1:])
			if !ok {
				return "", false
			}
		}
		switch strings.ToLower(name) {
		case "date":
			if len(parts) <= 1 {
				return "CURRENT_DATE", true
			}
			return utcDateTrunc("day", base), true
		case "time":
			if len(parts) <= 1 {
				return "CURRENT_TIME", true
			}
		}
		return base, true
	}
	return "", false
}

func currentTimeArgOrNow(arg string) string {
	if strings.TrimSpace(arg) == "" {
		return "'now'"
	}
	return arg
}

func isSQLiteCurrentTimeValue(arg string) bool {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return true
	}
	if strings.HasPrefix(arg, "'") || strings.HasPrefix(arg, `"`) {
		return strings.EqualFold(strings.Trim(arg, `'" `), "now")
	}
	switch strings.ToUpper(arg) {
	case "NOW", "CURRENT_TIMESTAMP", "CURRENT_DATE", "CURRENT_TIME":
		return true
	}
	return false
}

func consumeJSONValidCompare(rest string) (consumed int, notJSON bool, ok bool) {
	i := 0
	for i < len(rest) && (rest[i] == ' ' || rest[i] == '\t') {
		i++
	}
	op := ""
	switch {
	case strings.HasPrefix(rest[i:], "=="):
		op = "="
		i += 2
	case strings.HasPrefix(rest[i:], "!="), strings.HasPrefix(rest[i:], "<>"):
		op = "!="
		i += 2
	case i < len(rest) && rest[i] == '=':
		op = "="
		i++
	default:
		return 0, false, false
	}
	for i < len(rest) && (rest[i] == ' ' || rest[i] == '\t') {
		i++
	}
	val, n := readJSONValidCompareValue(rest[i:])
	if n == 0 {
		return 0, false, false
	}
	i += n
	truthy := val == "1" || val == "true"
	if op == "!=" {
		truthy = !truthy
	}
	return i, !truthy, true
}

func readJSONValidCompareValue(s string) (string, int) {
	if s == "" {
		return "", 0
	}
	lower := strings.ToLower(s)
	switch {
	case strings.HasPrefix(lower, "true") && (len(s) == 4 || !isSQLIdentChar(s[4])):
		return "true", 4
	case strings.HasPrefix(lower, "false") && (len(s) == 5 || !isSQLIdentChar(s[5])):
		return "false", 5
	case (s[0] == '0' || s[0] == '1') && (len(s) == 1 || !isSQLIdentChar(s[1]) && s[1] != '.'):
		return string(s[0]), 1
	}
	return "", 0
}

func rewritePrefixJSONValidCompares(expr string) string {
	var out strings.Builder
	n := len(expr)
	for i := 0; i < n; {
		if expr[i] == '\'' || expr[i] == '"' {
			j := quotedEnd(expr, i, expr[i])
			out.WriteString(expr[i:j])
			i = j
			continue
		}
		if consumed, replacement, ok := matchPrefixJSONValidCompare(expr, i); ok {
			out.WriteString(replacement)
			i += consumed
			continue
		}
		out.WriteByte(expr[i])
		i++
	}
	return out.String()
}

func matchPrefixJSONValidCompare(expr string, i int) (int, string, bool) {
	if i > 0 && isSQLIdentChar(expr[i-1]) {
		return 0, "", false
	}
	val, n := readJSONValidCompareValue(expr[i:])
	if n == 0 {
		return 0, "", false
	}
	j := i + n
	for j < len(expr) && (expr[j] == ' ' || expr[j] == '\t') {
		j++
	}
	op := ""
	switch {
	case strings.HasPrefix(expr[j:], "=="):
		op = "="
		j += 2
	case strings.HasPrefix(expr[j:], "!="), strings.HasPrefix(expr[j:], "<>"):
		op = "!="
		j += 2
	case j < len(expr) && expr[j] == '=':
		op = "="
		j++
	default:
		return 0, "", false
	}
	for j < len(expr) && (expr[j] == ' ' || expr[j] == '\t') {
		j++
	}
	pred, end, ok := parseLeadingIsJSON(expr[j:])
	if !ok {
		return 0, "", false
	}
	truthy := val == "1" || val == "true"
	if op == "!=" {
		truthy = !truthy
	}
	if pred.not {
		truthy = !truthy
	}
	out := pred.operand + " IS JSON"
	if !truthy {
		out = pred.operand + " IS NOT JSON"
	}
	return (j - i) + end, out, true
}

type isJSONPred struct {
	operand string
	not     bool
}

func parseLeadingIsJSON(s string) (isJSONPred, int, bool) {
	if !strings.HasPrefix(s, "(") {
		return isJSONPred{}, 0, false
	}
	end, ok := matchingParenEnd(s, 0)
	if !ok {
		return isJSONPred{}, 0, false
	}
	k := end + 1
	for k < len(s) && (s[k] == ' ' || s[k] == '\t') {
		k++
	}
	rest := s[k:]
	upper := strings.ToUpper(rest)
	not := false
	switch {
	case strings.HasPrefix(upper, "IS NOT JSON"):
		if len(rest) > 11 && isSQLIdentChar(rest[11]) {
			return isJSONPred{}, 0, false
		}
		not = true
		k += len("IS NOT JSON")
	case strings.HasPrefix(upper, "IS JSON"):
		if len(rest) > 7 && isSQLIdentChar(rest[7]) {
			return isJSONPred{}, 0, false
		}
		k += len("IS JSON")
	default:
		return isJSONPred{}, 0, false
	}
	return isJSONPred{operand: s[:end+1], not: not}, k, true
}

func sqliteTypeofExpr(arg string) string {
	return "CASE" +
		" WHEN (" + arg + ") IS NULL THEN 'null'" +
		" WHEN pg_typeof(" + arg + ")::text IN ('integer', 'bigint', 'smallint') THEN 'integer'" +
		" WHEN pg_typeof(" + arg + ")::text IN ('double precision', 'real', 'numeric') THEN 'real'" +
		" WHEN pg_typeof(" + arg + ")::text = 'bytea' THEN 'blob'" +
		" ELSE 'text' END"
}

func mapJSONExtract(args string) (string, bool) {
	parts := splitFunctionArgs(args)
	if len(parts) != 2 {
		return "", false
	}
	path, ok := sqliteJSONPathLiteral(parts[1])
	if !ok {
		return "", false
	}
	return "((" + parts[0] + ")::jsonb #>> " + path + ")", true
}

// sqliteJSONPathLiteral converts a SQL string literal holding a SQLite JSON path
// ($.foo.bar, $[0].foo) into a Postgres text[] literal for #> / #>>.
func sqliteJSONPathLiteral(lit string) (string, bool) {
	path, ok := unquoteSQLString(lit)
	if !ok {
		return "", false
	}
	if path == "$" {
		return "'{}'", true
	}
	keys, ok := parseSQLiteJSONPath(path)
	if !ok || len(keys) == 0 {
		return "", false
	}
	return postgresTextArrayLiteral(keys), true
}

func parseSQLiteJSONPath(path string) ([]string, bool) {
	if path == "$" || !strings.HasPrefix(path, "$") {
		return nil, false
	}
	var keys []string
	i := 1
	for i < len(path) {
		switch path[i] {
		case '.':
			i++
			if i >= len(path) {
				return nil, false
			}
			if path[i] == '"' {
				key, next, ok := readJSONPathQuotedKey(path, i)
				if !ok {
					return nil, false
				}
				keys = append(keys, key)
				i = next
				continue
			}
			start := i
			for i < len(path) && path[i] != '.' && path[i] != '[' {
				i++
			}
			if start == i {
				return nil, false
			}
			keys = append(keys, path[start:i])
		case '[':
			end := strings.IndexByte(path[i:], ']')
			if end < 0 {
				return nil, false
			}
			idx := path[i+1 : i+end]
			if idx == "" || strings.HasPrefix(idx, "#") {
				return nil, false
			}
			keys = append(keys, idx)
			i = i + end + 1
		default:
			return nil, false
		}
	}
	return keys, true
}

func readJSONPathQuotedKey(path string, quoteAt int) (string, int, bool) {
	var key strings.Builder
	for j := quoteAt + 1; j < len(path); j++ {
		if path[j] == '\\' && j+1 < len(path) {
			key.WriteByte(path[j+1])
			j++
			continue
		}
		if path[j] == '"' {
			return key.String(), j + 1, true
		}
		key.WriteByte(path[j])
	}
	return "", 0, false
}

func postgresTextArrayLiteral(keys []string) string {
	var b strings.Builder
	b.WriteString("'{")
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		if arrayElementNeedsQuotes(k) {
			b.WriteByte('"')
			b.WriteString(strings.ReplaceAll(strings.ReplaceAll(k, `\`, `\\`), `"`, `\"`))
			b.WriteByte('"')
		} else {
			b.WriteString(k)
		}
	}
	b.WriteString("}'")
	return b.String()
}

func arrayElementNeedsQuotes(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		if r == ',' || r == '{' || r == '}' || r == '"' || r == '\\' || r == ' ' || r == '\t' {
			return true
		}
	}
	return false
}

func splitFunctionArgs(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var args []string
	var current strings.Builder
	depth := 0
	inQuote := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote != 0 {
			current.WriteByte(c)
			if c == inQuote {
				if i+1 < len(s) && s[i+1] == inQuote {
					current.WriteByte(s[i+1])
					i++
					continue
				}
				inQuote = 0
			}
			continue
		}
		switch c {
		case '\'', '"':
			inQuote = c
			current.WriteByte(c)
		case '(':
			depth++
			current.WriteByte(c)
		case ')':
			depth--
			current.WriteByte(c)
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(current.String()))
				current.Reset()
				continue
			}
			current.WriteByte(c)
		default:
			current.WriteByte(c)
		}
	}
	args = append(args, strings.TrimSpace(current.String()))
	return args
}

func unquoteSQLString(lit string) (string, bool) {
	lit = strings.TrimSpace(lit)
	if len(lit) < 2 {
		return "", false
	}
	q := lit[0]
	if (q != '\'' && q != '"') || lit[len(lit)-1] != q {
		return "", false
	}
	var b strings.Builder
	for i := 1; i < len(lit)-1; i++ {
		if lit[i] == q && i+1 < len(lit)-1 && lit[i+1] == q {
			b.WriteByte(q)
			i++
			continue
		}
		if lit[i] == q {
			return "", false
		}
		b.WriteByte(lit[i])
	}
	return b.String(), true
}

func parsePositiveIntArg(args string) (int, error) {
	parts := splitFunctionArgs(args)
	if len(parts) != 1 {
		return 0, fmt.Errorf("want 1 arg")
	}
	n := 0
	for _, r := range strings.TrimSpace(parts[0]) {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not an integer")
		}
		n = n*10 + int(r-'0')
		if n > 1<<20 {
			return 0, fmt.Errorf("too large")
		}
	}
	if n == 0 {
		return 0, fmt.Errorf("zero")
	}
	return n, nil
}

func rewriteGlobAndRegexpOperators(expr string) string {
	var out strings.Builder
	n := len(expr)
	for i := 0; i < n; {
		c := expr[i]
		if c == '\'' || c == '"' {
			j := quotedEnd(expr, i, c)
			out.WriteString(expr[i:j])
			i = j
			continue
		}
		if !isIdentStartByte(c) {
			out.WriteByte(c)
			i++
			continue
		}
		j := i + 1
		for j < n && isSQLIdentChar(expr[j]) {
			j++
		}
		word := expr[i:j]
		lower := strings.ToLower(word)
		if lower == "not" {
			if rewritten, next, ok := tryRewritePatternOp(expr, j, true); ok {
				out.WriteString(rewritten)
				i = next
				continue
			}
		}
		if lower == "glob" || lower == "regexp" {
			if rewritten, next, ok := rewritePatternOpAt(expr, j, lower, false); ok {
				out.WriteString(rewritten)
				i = next
				continue
			}
		}
		out.WriteString(word)
		i = j
	}
	return out.String()
}

func tryRewritePatternOp(expr string, afterNot int, negated bool) (string, int, bool) {
	n := len(expr)
	k := afterNot
	for k < n && (expr[k] == ' ' || expr[k] == '\t') {
		k++
	}
	if k >= n || !isIdentStartByte(expr[k]) {
		return "", 0, false
	}
	j := k + 1
	for j < n && isSQLIdentChar(expr[j]) {
		j++
	}
	op := strings.ToLower(expr[k:j])
	if op != "glob" && op != "regexp" {
		return "", 0, false
	}
	return rewritePatternOpAt(expr, j, op, negated)
}

func rewritePatternOpAt(expr string, opEnd int, op string, negated bool) (string, int, bool) {
	n := len(expr)
	k := opEnd
	for k < n && (expr[k] == ' ' || expr[k] == '\t') {
		k++
	}
	if k >= n || (expr[k] != '\'' && expr[k] != '"') {
		return "", 0, false
	}
	end := quotedEnd(expr, k, expr[k])
	lit := expr[k:end]
	sym := "~"
	if negated {
		sym = "!~"
	}
	if op == "glob" {
		pattern, ok := unquoteSQLString(lit)
		if !ok {
			return "", 0, false
		}
		return sym + " " + quotePostgresLiteral(globToPOSIXRegex(pattern)), end, true
	}
	if q, ok := unquoteSQLString(lit); ok {
		lit = quotePostgresLiteral(q)
	}
	return sym + " " + lit, end, true
}

func globToPOSIXRegex(pattern string) string {
	var b strings.Builder
	b.WriteByte('^')
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteByte('.')
		case '[':
			j := i + 1
			if j < len(pattern) && (pattern[j] == '!' || pattern[j] == '^') {
				j++
			}
			if j < len(pattern) && pattern[j] == ']' {
				j++
			}
			for j < len(pattern) && pattern[j] != ']' {
				j++
			}
			if j >= len(pattern) {
				b.WriteString(`\[`)
				continue
			}
			class := pattern[i+1 : j]
			b.WriteByte('[')
			if strings.HasPrefix(class, "!") {
				b.WriteByte('^')
				b.WriteString(class[1:])
			} else {
				b.WriteString(class)
			}
			b.WriteByte(']')
			i = j
		default:
			if strings.ContainsRune(`.+()|{}^$\\`, rune(pattern[i])) {
				b.WriteByte('\\')
			}
			b.WriteByte(pattern[i])
		}
	}
	b.WriteByte('$')
	return b.String()
}

type sqliteFuncCall struct {
	name string
	args string
}

func findFunctionCalls(expr string) []sqliteFuncCall {
	var calls []sqliteFuncCall
	n := len(expr)
	for i := 0; i < n; {
		c := expr[i]
		switch {
		case c == '\'':
			i = quotedEnd(expr, i, '\'')
		case c == '"':
			i = quotedEnd(expr, i, '"')
		case c == '[':
			if end := strings.IndexByte(expr[i+1:], ']'); end >= 0 {
				i = i + 1 + end + 1
			} else {
				i = n
			}
		case c == '`':
			i = quotedEnd(expr, i, '`')
		case isIdentStartByte(c):
			j := i + 1
			for j < n && isSQLIdentChar(expr[j]) {
				j++
			}
			name := expr[i:j]
			k := j
			for k < n && unicode.IsSpace(rune(expr[k])) {
				k++
			}
			if k < n && expr[k] == '(' {
				end, ok := matchingParenEnd(expr, k)
				if !ok {
					return calls
				}
				args := expr[k+1 : end]
				calls = append(calls, sqliteFuncCall{name: name, args: args})
				calls = append(calls, findFunctionCalls(args)...)
				i = end + 1
				continue
			}
			i = j
		default:
			i++
		}
	}
	return calls
}

func unconvertedSQLiteFunctions(expr string) []string {
	var names []string
	seen := map[string]struct{}{}
	for _, call := range findFunctionCalls(expr) {
		lower := strings.ToLower(call.name)
		if _, ok := sqliteOnlyCheckFuncs[lower]; !ok {
			continue
		}
		if _, converted := mapSQLiteCheckFunction(call.name, call.args); converted {
			continue
		}
		if _, dup := seen[lower]; dup {
			continue
		}
		seen[lower] = struct{}{}
		names = append(names, call.name)
	}
	if leftoverPatternOp(expr, "GLOB") {
		names = append(names, "GLOB")
	}
	if leftoverPatternOp(expr, "REGEXP") {
		names = append(names, "REGEXP")
	}
	return names
}

func leftoverPatternOp(expr, op string) bool {
	n := len(expr)
	want := strings.ToLower(op)
	for i := 0; i < n; {
		c := expr[i]
		if c == '\'' || c == '"' {
			i = quotedEnd(expr, i, c)
			continue
		}
		if !isIdentStartByte(c) {
			i++
			continue
		}
		j := i + 1
		for j < n && isSQLIdentChar(expr[j]) {
			j++
		}
		if strings.ToLower(expr[i:j]) == want {
			k := j
			for k < n && (expr[k] == ' ' || expr[k] == '\t') {
				k++
			}
			if k >= n || (expr[k] != '\'' && expr[k] != '"') {
				return true
			}
		}
		i = j
	}
	return false
}
