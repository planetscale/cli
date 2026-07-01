package d1

import (
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
)

var (
	uuidValueRe      = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	timestampValueRe = regexp.MustCompile(`(?i)^\d{4}-\d{2}-\d{2}(?:[ T]\d{2}:\d{2}(?::\d{2}(?:\.\d+)?)?(?:Z|[+-]\d{2}:?\d{2})?)?$`)
)

// ColumnSamples holds sampled INSERT values per table/column.
type ColumnSamples map[string]map[string][]string

// TypeCoercionContext carries sampled values used to validate name-based coercions.
type TypeCoercionContext struct {
	Samples ColumnSamples
}

func BuildTypeCoercionContext(inputPath string, tables []TableSchema) (*TypeCoercionContext, error) {
	samples, err := SampleColumnValues(inputPath, tables)
	if err != nil {
		return nil, err
	}
	return &TypeCoercionContext{Samples: samples}, nil
}

func (ctx *TypeCoercionContext) samplesFor(table, column string) []string {
	if ctx == nil || ctx.Samples == nil {
		return nil
	}
	if cols, ok := ctx.Samples[table]; ok {
		return cols[column]
	}
	return nil
}

// SampleColumnValues reads INSERT statements and collects non-null literal values.
func SampleColumnValues(path string, tables []TableSchema) (ColumnSamples, error) {
	clean, err := ValidateInputPath(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(clean)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tableCols := make(map[string][]string, len(tables))
	for _, t := range tables {
		cols := make([]string, 0, len(t.Columns))
		for _, c := range t.Columns {
			cols = append(cols, c.Name)
		}
		tableCols[t.Name] = cols
	}

	samples := make(ColumnSamples)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	var pendingInsert strings.Builder

	flushInsert := func() {
		line := strings.TrimSpace(pendingInsert.String())
		pendingInsert.Reset()
		if line == "" {
			return
		}
		m := insertRe.FindStringSubmatch(line)
		if m == nil {
			return
		}
		table := firstNonEmpty(m[1], m[2], m[3], m[4])
		columns, valueGroups, ok := parseInsertColumnsAndValues(line)
		if !ok || len(valueGroups) == 0 {
			return
		}
		if len(columns) == 0 {
			columns = tableCols[table]
		}
		if len(columns) == 0 {
			return
		}
		if samples[table] == nil {
			samples[table] = make(map[string][]string)
		}
		for _, values := range valueGroups {
			for i, col := range columns {
				if i >= len(values) {
					break
				}
				val := values[i]
				if val == "" || strings.EqualFold(val, "NULL") {
					continue
				}
				val = unquoteSQLLiteral(val)
				samples[table][col] = appendUniqueSample(samples[table][col], val, 32)
			}
		}
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		if pendingInsert.Len() > 0 {
			pendingInsert.WriteString(" ")
			pendingInsert.WriteString(line)
			if strings.HasSuffix(line, ";") {
				flushInsert()
			}
			continue
		}

		m := insertRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if strings.HasSuffix(line, ";") {
			pendingInsert.WriteString(line)
			flushInsert()
			continue
		}
		pendingInsert.WriteString(line)
	}
	flushInsert()

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return samples, nil
}

func appendUniqueSample(existing []string, val string, max int) []string {
	if slices.Contains(existing, val) {
		return existing
	}
	if len(existing) >= max {
		return existing
	}
	return append(existing, val)
}

func parseInsertColumnsAndValues(line string) (columns []string, valueGroups [][]string, ok bool) {
	upper := strings.ToUpper(line)
	valuesIdx := strings.Index(upper, " VALUES")
	if valuesIdx < 0 {
		return nil, nil, false
	}
	head := line[:valuesIdx]
	valuesPart := strings.TrimSpace(line[valuesIdx+len(" VALUES"):])
	valuesPart = strings.TrimSuffix(valuesPart, ";")

	openParen := strings.Index(head, "(")
	closeParen := strings.LastIndex(head, ")")
	if openParen >= 0 && closeParen > openParen {
		colPart := head[openParen+1 : closeParen]
		for _, part := range splitCommaList(colPart) {
			name, _ := parseColumnNameAndRest(strings.TrimSpace(part))
			if name != "" {
				columns = append(columns, name)
			}
		}
	}

	valuesPart = strings.TrimSpace(valuesPart)
	if !strings.HasPrefix(valuesPart, "(") {
		return columns, nil, false
	}

	// Split one or more (...) value tuples.
	var tuples []string
	var current strings.Builder
	depth := 0
	inSingle := false
	inDouble := false
	for i := 0; i < len(valuesPart); i++ {
		c := valuesPart[i]
		switch c {
		case '\'':
			if !inDouble {
				if inSingle && i+1 < len(valuesPart) && valuesPart[i+1] == '\'' {
					current.WriteByte(c)
					current.WriteByte(valuesPart[i+1])
					i++
					continue
				}
				inSingle = !inSingle
			}
			current.WriteByte(c)
		case '"':
			if !inSingle {
				if inDouble && i+1 < len(valuesPart) && valuesPart[i+1] == '"' {
					current.WriteByte(c)
					current.WriteByte(valuesPart[i+1])
					i++
					continue
				}
				inDouble = !inDouble
			}
			current.WriteByte(c)
		case '(':
			if !inSingle && !inDouble {
				if depth == 0 {
					current.Reset()
				}
				depth++
			}
			if depth > 0 {
				current.WriteByte(c)
			}
		case ')':
			if !inSingle && !inDouble {
				if depth > 0 {
					current.WriteByte(c)
				}
				depth--
				if depth == 0 {
					inner := strings.TrimSpace(current.String())
					inner = strings.TrimPrefix(inner, "(")
					inner = strings.TrimSuffix(inner, ")")
					tuples = append(tuples, inner)
					current.Reset()
				}
				continue
			}
			current.WriteByte(c)
		default:
			if depth > 0 {
				current.WriteByte(c)
			}
		}
	}

	for _, tuple := range tuples {
		valueGroups = append(valueGroups, splitInsertValues(tuple))
	}
	return columns, valueGroups, len(valueGroups) > 0
}

func splitInsertValues(s string) []string {
	var parts []string
	var current strings.Builder
	depth := 0
	inSingle := false
	inDouble := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\'':
			if !inDouble {
				if inSingle && i+1 < len(s) && s[i+1] == '\'' {
					current.WriteByte(c)
					current.WriteByte(s[i+1])
					i++
					continue
				}
				inSingle = !inSingle
			}
			current.WriteByte(c)
		case '"':
			if !inSingle {
				if inDouble && i+1 < len(s) && s[i+1] == '"' {
					current.WriteByte(c)
					current.WriteByte(s[i+1])
					i++
					continue
				}
				inDouble = !inDouble
			}
			current.WriteByte(c)
		case '(':
			if !inSingle && !inDouble {
				depth++
			}
			current.WriteByte(c)
		case ')':
			if !inSingle && !inDouble {
				depth--
			}
			current.WriteByte(c)
		case ',':
			if depth == 0 && !inSingle && !inDouble {
				parts = append(parts, strings.TrimSpace(current.String()))
				current.Reset()
				continue
			}
			current.WriteByte(c)
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, strings.TrimSpace(current.String()))
	}
	return parts
}

func unquoteSQLLiteral(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		inner := s[1 : len(s)-1]
		return strings.ReplaceAll(inner, "''", "'")
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		inner := s[1 : len(s)-1]
		return strings.ReplaceAll(inner, `""`, `"`)
	}
	return s
}

func looksLikeUUID(s string) bool {
	return uuidValueRe.MatchString(strings.TrimSpace(s))
}

func looksLikeJSON(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	switch s[0] {
	case '{', '[':
		var v any
		return json.Unmarshal([]byte(s), &v) == nil
	default:
		return false
	}
}

func looksLikeTimestamp(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	return timestampValueRe.MatchString(s)
}

func samplesLookBoolean(table, column string, ctx *TypeCoercionContext) bool {
	vals := ctx.samplesFor(table, column)
	if len(vals) == 0 {
		return false
	}
	for _, v := range vals {
		if v == "" {
			continue
		}
		if v != "0" && v != "1" {
			return false
		}
	}
	return true
}

func samplesAllow(table, column string, ctx *TypeCoercionContext, allow func(string) bool) bool {
	vals := ctx.samplesFor(table, column)
	if len(vals) == 0 {
		return false
	}
	for _, v := range vals {
		if v != "" && !allow(v) {
			return false
		}
	}
	return true
}

func samplesAllowUUID(table, column string, ctx *TypeCoercionContext) bool {
	return samplesAllow(table, column, ctx, looksLikeUUID)
}

func samplesAllowJSON(table, column string, ctx *TypeCoercionContext) bool {
	return samplesAllow(table, column, ctx, looksLikeJSON)
}

func samplesAllowTimestamp(table, column string, ctx *TypeCoercionContext) bool {
	return samplesAllow(table, column, ctx, looksLikeTimestamp)
}
