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
	valueTupleSepRe = regexp.MustCompile(`\)\s*,\s*\(`)
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
		current.RawDDL = strings.Join(ddlLines, "\n")
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
	end := strings.LastIndex(ddl, ")")
	if start < 0 || end <= start {
		return nil, nil
	}
	body := ddl[start+1 : end]
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

	// Strip trailing comma
	def = strings.TrimSuffix(def, ",")

	tokens := strings.Fields(def)
	if len(tokens) == 0 {
		return ColumnSchema{}
	}

	name := strings.Trim(tokens[0], "`\"'")
	colType := ""
	if len(tokens) > 1 {
		colType = strings.ToUpper(tokens[1])
	}

	col := ColumnSchema{
		Name: name,
		Type: colType,
	}

	upper := strings.ToUpper(def)
	if strings.Contains(upper, "NOT NULL") {
		col.NotNull = true
	}
	if strings.Contains(upper, "PRIMARY KEY") {
		col.PrimaryKey = true
	}
	if columnUniqueRe.MatchString(def) {
		col.Unique = true
	}
	if autoincrementRe.MatchString(def) {
		col.AutoIncrement = true
	}
	if idx := strings.Index(strings.ToUpper(def), "DEFAULT"); idx >= 0 {
		col.DefaultValue = strings.TrimSpace(def[idx+7:])
		col.DefaultValue = strings.TrimSuffix(col.DefaultValue, ",")
		if refIdx := indexOfIgnoreCase(col.DefaultValue, "REFERENCES"); refIdx >= 0 {
			col.DefaultValue = strings.TrimSpace(col.DefaultValue[:refIdx])
			col.DefaultValue = strings.TrimSuffix(col.DefaultValue, ",")
		}
	}
	if strings.Contains(upper, "REFERENCES") {
		col.ForeignKey = referencesClause(def)
	}

	return col
}

// ParseIndexes extracts CREATE INDEX statements from a SQLite dump.
func ParseIndexes(path string) ([]IndexSchema, error) {
	clean, err := ValidateInputPath(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(clean)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var indexes []IndexSchema
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		if !strings.HasPrefix(strings.ToUpper(line), "CREATE") {
			continue
		}
		upper := strings.ToUpper(line)
		if !strings.Contains(upper, " INDEX ") {
			continue
		}
		m := createIndexRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		indexes = append(indexes, IndexSchema{
			Name:    firstNonEmpty(m[2], m[3], m[4], m[5]),
			Table:   firstNonEmpty(m[6], m[7], m[8], m[9]),
			Unique:  strings.TrimSpace(m[1]) != "",
			Columns: m[10],
			RawDDL:  line,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read dump indexes: %w", err)
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
		rows := len(valueTupleSepRe.FindAllString(sql, -1)) + 1
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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
