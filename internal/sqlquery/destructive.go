package sqlquery

import (
	"slices"
	"strings"
)

var destructiveWords = []string{"DELETE", "DROP", "TRUNCATE"}

// DestructiveQueryError is returned when a query would delete or drop data or
// schema objects and --force was not passed.
type DestructiveQueryError struct{}

func (e *DestructiveQueryError) Error() string {
	return "destructive SQL requires explicit user approval (ask the user, then re-run with --force)"
}

// IsDestructiveQuery reports whether the query deletes or drops resources.
// This is a best-effort guardrail for agents, not a SQL parser. Statements
// led by DELETE, DROP, or TRUNCATE (including after CTEs) require approval.
func IsDestructiveQuery(query string) bool {
	return slices.ContainsFunc(splitSQLStatements(stripSQLGuardIgnoredText(query)), isDestructiveStatement)
}

func splitSQLStatements(query string) []string {
	out := make([]string, 0, strings.Count(query, ";")+1)
	start := 0
	quote := byte(0)
	for i := 0; i < len(query); i++ {
		c := query[i]
		if quote != 0 {
			if c == quote {
				if i+1 < len(query) && query[i+1] == quote {
					i++
					continue
				}
				quote = 0
			}
			continue
		}
		switch c {
		case '\'', '"', '`':
			quote = c
		case ';':
			if trimmed := strings.TrimSpace(query[start:i]); trimmed != "" {
				out = append(out, trimmed)
			}
			start = i + 1
		}
	}
	if trimmed := strings.TrimSpace(query[start:]); trimmed != "" {
		out = append(out, trimmed)
	}
	return out
}

func isDestructiveStatement(stmt string) bool {
	return slices.ContainsFunc(splitDestructiveSegments(stmt), isDestructiveSegment)
}

func splitDestructiveSegments(stmt string) []string {
	var out []string
	start := 0
	for i := 0; i < len(stmt); i++ {
		if stmt[i] != '\n' {
			continue
		}
		j := i + 1
		for j < len(stmt) && (stmt[j] == ' ' || stmt[j] == '\t' || stmt[j] == '\r') {
			j++
		}
		if j >= len(stmt) {
			continue
		}
		rest := strings.ToUpper(stmt[j:])
		if !slices.ContainsFunc(destructiveWords, func(word string) bool {
			return hasKeywordPrefix(rest, word)
		}) {
			continue
		}
		if trimmed := strings.TrimSpace(stmt[start:i]); trimmed != "" {
			out = append(out, trimmed)
		}
		start = j
		i = j - 1
	}
	if trimmed := strings.TrimSpace(stmt[start:]); trimmed != "" {
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return []string{stmt}
	}
	return out
}

func isDestructiveSegment(stmt string) bool {
	trimmed := strings.TrimSpace(stmt)
	upper := strings.ToUpper(trimmed)

	if hasKeywordPrefix(upper, "WITH") {
		if cteContainsDestructive(trimmed) {
			return true
		}
		after, ok := queryAfterCTEs(trimmed)
		if !ok {
			return false
		}
		return isDestructiveSegment(after)
	}

	switch leadingStatementKeyword(trimmed) {
	case "DELETE", "DROP", "TRUNCATE":
		return true
	case "EXPLAIN":
		inner, analyze := parseExplainAnalyze(trimmed)
		if !analyze || inner == "" {
			return false
		}
		return isDestructiveSegment(inner)
	case "ALTER":
		upperNorm := strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(upper)
		return containsWord(upperNorm, "DROP")
	case "MERGE":
		upperNorm := strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(upper)
		return strings.Contains(" "+upperNorm+" ", " THEN DELETE ")
	default:
		return false
	}
}

func leadingStatementKeyword(stmt string) string {
	trimmed := strings.TrimSpace(stmt)
	for i := 0; i < len(trimmed); {
		for i < len(trimmed) && trimmed[i] <= ' ' {
			i++
		}
		if i >= len(trimmed) {
			break
		}
		start := i
		for i < len(trimmed) && isIdentifierChar(trimmed[i]) {
			i++
		}
		if start < i {
			return strings.ToUpper(trimmed[start:i])
		}
		i++
	}
	return ""
}

func stripSQLGuardIgnoredText(stmt string) string {
	var out strings.Builder
	out.Grow(len(stmt))
	quote := byte(0)
	lineComment := false
	blockComment := false
	execComment := false
	for i := 0; i < len(stmt); i++ {
		c := stmt[i]
		if lineComment {
			if c == '\n' {
				lineComment = false
				out.WriteByte(c)
			} else {
				out.WriteByte(' ')
			}
			continue
		}
		if blockComment {
			out.WriteByte(' ')
			if c == '*' && i+1 < len(stmt) && stmt[i+1] == '/' {
				i++
				out.WriteByte(' ')
				blockComment = false
			}
			continue
		}
		if quote != 0 {
			out.WriteByte(' ')
			if c == quote {
				if i+1 < len(stmt) && stmt[i+1] == quote {
					i++
					out.WriteByte(' ')
					continue
				}
				quote = 0
			}
			continue
		}
		// MySQL executable comments /*! ... */ and /*!NNNNN ... */ are live SQL.
		// Blank the delimiters (and optional version digits) but keep the body.
		if execComment && c == '*' && i+1 < len(stmt) && stmt[i+1] == '/' {
			out.WriteByte(' ')
			i++
			out.WriteByte(' ')
			execComment = false
			continue
		}
		switch c {
		case '#':
			lineComment = true
			out.WriteByte(' ')
		case '-':
			if i+1 < len(stmt) && stmt[i+1] == '-' {
				lineComment = true
				out.WriteByte(' ')
				i++
				out.WriteByte(' ')
			} else {
				out.WriteByte(c)
			}
		case '/':
			if i+1 < len(stmt) && stmt[i+1] == '*' {
				if i+2 < len(stmt) && stmt[i+2] == '!' {
					out.WriteByte(' ')
					out.WriteByte(' ')
					out.WriteByte(' ')
					i += 2
					for i+1 < len(stmt) && stmt[i+1] >= '0' && stmt[i+1] <= '9' {
						i++
						out.WriteByte(' ')
					}
					execComment = true
				} else {
					blockComment = true
					out.WriteByte(' ')
					i++
					out.WriteByte(' ')
				}
			} else {
				out.WriteByte(c)
			}
		case '\'', '"', '`':
			quote = c
			out.WriteByte(' ')
		default:
			out.WriteByte(c)
		}
	}
	return out.String()
}

func cteContainsDestructive(query string) bool {
	bodies, _, ok := walkCTEs(query)
	if !ok {
		return false
	}
	return slices.ContainsFunc(bodies, isDestructiveSegment)
}

func parseExplainAnalyze(stmt string) (inner string, analyze bool) {
	rest := strings.TrimSpace(stmt)
	if leadingStatementKeyword(rest) != "EXPLAIN" {
		return "", false
	}
	i := 0
	for i < len(rest) && isIdentifierChar(rest[i]) {
		i++
	}
	rest = strings.TrimSpace(rest[i:])
	if strings.HasPrefix(rest, "(") {
		end := matchingParenIndex(rest)
		if end < 0 {
			return "", false
		}
		opts := strings.ToUpper(rest[1:end])
		analyze = containsWord(strings.NewReplacer(",", " ", "(", " ", ")", " ").Replace(opts), "ANALYZE")
		return strings.TrimSpace(rest[end+1:]), analyze
	}
	for {
		upper := strings.ToUpper(rest)
		switch {
		case hasKeywordPrefix(upper, "ANALYZE"):
			analyze = true
			rest = strings.TrimSpace(rest[len("ANALYZE"):])
		case hasKeywordPrefix(upper, "VERBOSE"):
			rest = strings.TrimSpace(rest[len("VERBOSE"):])
		default:
			return rest, analyze
		}
	}
}

func containsWord(q, word string) bool {
	return strings.Contains(" "+q+" ", " "+word+" ")
}
