package d1

import (
	"regexp"
	"strings"
)

var (
	referencesClauseRe = regexp.MustCompile(`(?is)^REFERENCES\s+(?:"([^"]+)"|'([^']+)'|` + "`" + `([^` + "`" + `]+)` + "`" + `|([a-zA-Z_][\w]*))\s*\(\s*([^)]+)\)\s*(.*)$`)
	foreignKeyConstraintRe = regexp.MustCompile(`(?is)^FOREIGN\s+KEY\s*\(\s*([^)]+)\)\s*(REFERENCES\s+.+)$`)
	primaryKeyConstraintRe = regexp.MustCompile(`(?is)^PRIMARY\s+KEY\s*\(\s*([^)]+)\)\s*$`)
	uniqueConstraintRe = regexp.MustCompile(`(?is)^UNIQUE\s*\(\s*([^)]+)\)\s*$`)
	createIndexRe = regexp.MustCompile(`(?is)^CREATE\s+(UNIQUE\s+)?INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:"([^"]+)"|'([^']+)'|` + "`" + `([^` + "`" + `]+)` + "`" + `|([a-zA-Z_][\w]*))\s+ON\s+(?:"([^"]+)"|'([^']+)'|` + "`" + `([^` + "`" + `]+)` + "`" + `|([a-zA-Z_][\w]*))\s*\(\s*([^)]+)\)\s*;?\s*$`)
)

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

func convertTableConstraint(clause string) string {
	clause = strings.TrimSpace(clause)
	clause = strings.TrimSuffix(clause, ",")
	if clause == "" {
		return ""
	}

	upper := strings.ToUpper(clause)
	switch {
	case strings.HasPrefix(upper, "FOREIGN KEY"):
		return convertForeignKeyConstraint(clause)
	case strings.HasPrefix(upper, "PRIMARY KEY"):
		return convertPrimaryKeyConstraint(clause)
	case strings.HasPrefix(upper, "UNIQUE"):
		return convertUniqueConstraint(clause)
	default:
		return clause
	}
}

func convertForeignKeyConstraint(clause string) string {
	m := foreignKeyConstraintRe.FindStringSubmatch(clause)
	if m == nil {
		return clause
	}
	cols := quoteColumnList(m[1])
	refs := convertReferencesClause(strings.TrimSpace(m[2]))
	return "FOREIGN KEY (" + cols + ") " + refs
}

func convertPrimaryKeyConstraint(clause string) string {
	m := primaryKeyConstraintRe.FindStringSubmatch(clause)
	if m == nil {
		return clause
	}
	return "PRIMARY KEY (" + quoteColumnList(m[1]) + ")"
}

func convertUniqueConstraint(clause string) string {
	m := uniqueConstraintRe.FindStringSubmatch(clause)
	if m == nil {
		return clause
	}
	return "UNIQUE (" + quoteColumnList(m[1]) + ")"
}

func convertReferencesClause(refs string) string {
	m := referencesClauseRe.FindStringSubmatch(refs)
	if m == nil {
		return refs
	}
	table := quoteIdent(firstNonEmpty(m[1], m[2], m[3], m[4]))
	refCols := quoteColumnList(m[5])
	tail := strings.TrimSpace(m[6])
	if tail != "" {
		return "REFERENCES " + table + " (" + refCols + ") " + tail
	}
	return "REFERENCES " + table + " (" + refCols + ")"
}

func quoteColumnList(list string) string {
	parts := splitCommaList(list)
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		quoted = append(quoted, quoteIdent(strings.Trim(part, "`\"'")))
	}
	return strings.Join(quoted, ", ")
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
	m := createIndexRe.FindStringSubmatch(raw)
	if m == nil {
		return raw
	}
	unique := strings.TrimSpace(m[1]) != ""
	name := quoteIdent(firstNonEmpty(m[2], m[3], m[4], m[5]))
	table := quoteIdent(firstNonEmpty(m[6], m[7], m[8], m[9]))
	cols := quoteColumnList(m[10])
	prefix := "CREATE INDEX IF NOT EXISTS "
	if unique {
		prefix = "CREATE UNIQUE INDEX IF NOT EXISTS "
	}
	return prefix + name + " ON " + table + " (" + cols + ");"
}

func isUUIDColumn(col ColumnSchema, table TableSchema, all []TableSchema) bool {
	return isExplicitUUIDColumn(col) || columnReferencesUUIDKey(col, table, all)
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

func columnReferencesUUIDKey(col ColumnSchema, table TableSchema, all []TableSchema) bool {
	refTable, refCol := columnFKTarget(col, table)
	if refTable == "" {
		return false
	}
	ref := tableByName(all, refTable)
	if ref == nil {
		return false
	}
	refColSchema := columnByName(*ref, refCol)
	return isExplicitUUIDColumn(refColSchema)
}

func columnFKTarget(col ColumnSchema, table TableSchema) (string, string) {
	if col.ForeignKey != "" {
		return parseReferencesTarget(col.ForeignKey)
	}
	for _, constraint := range table.Constraints {
		cols, refs := parseTableLevelForeignKey(constraint)
		for _, name := range cols {
			if name == col.Name {
				return parseReferencesTarget(refs)
			}
		}
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

func parseReferencesTarget(refs string) (string, string) {
	m := referencesClauseRe.FindStringSubmatch(strings.TrimSpace(refs))
	if m == nil {
		return "", ""
	}
	table := firstNonEmpty(m[1], m[2], m[3], m[4])
	refCols := splitCommaList(m[5])
	refCol := ""
	if len(refCols) > 0 {
		refCol = strings.Trim(strings.TrimSpace(refCols[0]), "`\"'")
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
