package tui

import (
	"regexp"
	"strings"
)

var (
	sqlClauseBoundary = regexp.MustCompile(`(?i)\s+(from|left\s+join|right\s+join|inner\s+join|outer\s+join|cross\s+join|full\s+join|join|where|group\s+by|having|order\s+by|limit|offset|returning|set|values|union\s+all|union|intersect|except)\s+`)
	sqlBoolBoundary   = regexp.MustCompile(`(?i)\s+(and|or)\s+`)
	dollarQuote       = regexp.MustCompile(`\$[A-Za-z_][A-Za-z0-9_]*\$|\$\$`)
)

// This is light display formatting for pg_stat_activity text, not SQL parsing.
// It turns common generated one-liners such as
// "select id from t where owner_id = $1 order by id limit 100" into a short
// vertical scan, and leaves risky input such as quoted text, comments,
// parentheses, dollar quotes, and authored multiline line breaks alone.
// formatQueryForDisplay normalizes a query into display lines. Authored
// newlines are preserved, common one-line statements are split at major SQL
// clauses, and uncertain input falls back to the original sanitized text.
func formatQueryForDisplay(query string) []string {
	raw := strings.TrimSpace(query)
	if raw == "" {
		return nil
	}
	if strings.Contains(raw, "\n") {
		raw = strings.TrimSuffix(raw, "\n")
		lines := strings.Split(raw, "\n")
		for i, line := range lines {
			lines[i] = sanitizeFooterText(line)
		}
		return lines
	}
	if !canFormatOneLineQuery(raw) {
		return []string{sanitizeFooterText(raw)}
	}
	query = strings.Join(strings.Fields(sanitizeFooterText(raw)), " ")
	return formatOneLineQuery(query)
}

func canFormatOneLineQuery(query string) bool {
	lower := strings.ToLower(query)
	if strings.ContainsAny(query, "'\"`()") {
		return false
	}
	if strings.Contains(lower, "--") || strings.Contains(lower, "#") || strings.Contains(lower, "/*") || strings.Contains(lower, "*/") {
		return false
	}
	if strings.Contains(lower, " between ") || dollarQuote.MatchString(query) {
		return false
	}
	return len(sqlClauseBoundary.FindAllStringSubmatchIndex(query, -1)) > 0
}

func formatOneLineQuery(query string) []string {
	matches := sqlClauseBoundary.FindAllStringSubmatchIndex(query, -1)
	if len(matches) == 0 {
		return []string{query}
	}

	lines := []string{}
	first := strings.TrimSpace(query[:matches[0][0]])
	if first != "" {
		lines = append(lines, first)
	}
	for i, match := range matches {
		end := len(query)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		clause := strings.TrimSpace(query[match[2]:end])
		if strings.HasPrefix(strings.ToLower(clause), "where ") {
			lines = append(lines, splitWhereClause(clause)...)
			continue
		}
		lines = append(lines, clause)
	}
	return lines
}

func splitWhereClause(clause string) []string {
	space := strings.IndexByte(clause, ' ')
	if space < 0 {
		return []string{clause}
	}
	prefix := clause[:space]
	body := strings.TrimSpace(clause[space+1:])
	matches := sqlBoolBoundary.FindAllStringSubmatchIndex(body, -1)
	if len(matches) == 0 {
		return []string{clause}
	}

	lines := []string{prefix + " " + strings.TrimSpace(body[:matches[0][0]])}
	for i, match := range matches {
		end := len(body)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		operator := body[match[2]:match[3]]
		condition := strings.TrimSpace(body[match[1]:end])
		lines = append(lines, "  "+operator+" "+condition)
	}
	return lines
}

// queryDisplayLines is the normalized-then-wrapped line set actually shown in
// the Query tab. Both the scroll clamp and the renderer derive their bounds
// from this exact slice, so they can never disagree about how many lines exist.
func queryDisplayLines(query string, width int) []string {
	formatted := formatQueryForDisplay(query)
	if len(formatted) == 0 {
		return nil
	}
	wrapped := make([]string, 0, len(formatted))
	for _, line := range formatted {
		if line == "" {
			wrapped = append(wrapped, "")
			continue
		}
		wrapped = append(wrapped, wrapLines(line, width)...)
	}
	return wrapped
}

// maxQueryOffset is the largest scroll offset that still fills the viewport.
// bodyHeight includes the row reserved for the "query lines X-Y/Z" indicator,
// so the visible query rows are bodyHeight-1 and the last line is reachable.
func maxQueryOffset(totalLines, bodyHeight int) int {
	if bodyHeight <= 1 || totalLines <= bodyHeight-1 {
		return 0
	}
	return totalLines - (bodyHeight - 1)
}
