package d1

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

const (
	MethodPgloader = "pgloader"
	MethodPsql     = "psql" // schema via psql; data via pgloader (dumps under 1GB)
)

// PlanOptions configures migration planning.
type PlanOptions struct {
	InputPath   string
	Org         string
	Database    string
	Branch      string
	Method      string
	MigrationID string      // optional: reuse an existing migration ID from plan/start
	Lint        *LintResult // optional: skip re-lint when already computed
}

// Plan builds a migration plan from a SQLite dump.
func Plan(opts PlanOptions) (*PlanResult, error) {
	tables, err := ParseDump(opts.InputPath)
	if err != nil {
		return nil, err
	}

	lintResult := opts.Lint
	if lintResult == nil {
		lintResult, err = Lint(opts.InputPath)
		if err != nil {
			return nil, err
		}
	}

	rowCounts, err := CountInsertRows(opts.InputPath)
	if err != nil {
		return nil, err
	}
	size, err := FileSize(opts.InputPath)
	if err != nil {
		return nil, err
	}

	method := opts.Method
	if method == "" {
		method = recommendMethod(size)
	}

	plan := &PlanResult{
		MigrationID:        opts.planMigrationID(),
		InputPath:          opts.InputPath,
		Org:                opts.Org,
		Database:           opts.Database,
		Branch:             opts.Branch,
		RecommendedMethod:  method,
		EstimatedSizeBytes: size,
		Tables:             make([]TablePlan, 0, len(tables)),
		CastRules:          defaultCastRules(),
		LoadOrder:          topologicalLoadOrder(tables),
		Issues:             lintResult.Issues,
	}

	for _, table := range tables {
		tp := TablePlan{
			Name:        table.Name,
			RowEstimate: rowCounts[table.Name],
		}
		for _, col := range table.Columns {
			if col.ForeignKey != "" {
				tp.HasFK = true
				break
			}
		}
		if !tp.HasFK {
			for _, ref := range parseTableFKReferences(table.RawDDL) {
				if ref != "" {
					tp.HasFK = true
					break
				}
			}
		}
		plan.Tables = append(plan.Tables, tp)
	}

	return plan, nil
}

func (opts PlanOptions) planMigrationID() string {
	if opts.MigrationID != "" {
		return opts.MigrationID
	}
	return gonanoid.MustGenerate("0123456789abcdefghijklmnopqrstuvwxyz", 12)
}

func recommendMethod(sizeBytes int64) string {
	const oneGB = 1024 * 1024 * 1024
	if sizeBytes > 0 && sizeBytes < oneGB {
		return MethodPsql
	}
	return MethodPgloader
}

func defaultCastRules() []CastRule {
	return []CastRule{
		{SourceType: "integer", TargetType: "boolean", Using: "(= 1)", Tables: "match-columns-like '%active%'"},
		{SourceType: "text", TargetType: "timestamptz", Using: "sqlite-timestamp-to-timestamp"},
		{SourceType: "text", TargetType: "jsonb", Using: "sqlite-text-to-jsonb"},
	}
}

func topologicalLoadOrder(tables []TableSchema) []string {
	names := make([]string, 0, len(tables))
	deps := make(map[string][]string)
	nameSet := make(map[string]bool)

	for _, t := range tables {
		names = append(names, t.Name)
		nameSet[t.Name] = true
		for _, col := range t.Columns {
			if ref := parseFKReference(col.ForeignKey); ref != "" && nameSet[ref] {
				deps[t.Name] = append(deps[t.Name], ref)
			}
		}
		for _, ref := range parseTableFKReferences(t.RawDDL) {
			if nameSet[ref] {
				deps[t.Name] = append(deps[t.Name], ref)
			}
		}
	}

	sort.Strings(names)

	visited := make(map[string]bool)
	var order []string

	var visit func(string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		for _, dep := range deps[name] {
			if dep != "" {
				visit(dep)
			}
		}
		order = append(order, name)
	}

	for _, name := range names {
		visit(name)
	}

	return order
}

func parseFKReference(fk string) string {
	if fk == "" {
		return ""
	}
	idx := indexOfIgnoreCase(fk, "REFERENCES")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(fk[idx+len("REFERENCES"):])
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		return ""
	}
	return strings.Trim(parts[0], "`\"'")
}

var tableFKRe = regexp.MustCompile(`(?i)FOREIGN\s+KEY[^)]*\)\s*REFERENCES\s+(?:` + "`" + `([^` + "`" + `]+)` + "`" + `|"([^"]+)"|'([^']+)'|([a-zA-Z_][\w]*))`)

func parseTableFKReferences(ddl string) []string {
	matches := tableFKRe.FindAllStringSubmatch(ddl, -1)
	var refs []string
	for _, m := range matches {
		ref := firstNonEmpty(m[1], m[2], m[3], m[4])
		if ref != "" {
			refs = append(refs, ref)
		}
	}
	return refs
}

func indexOfIgnoreCase(s, sub string) int {
	return strings.Index(strings.ToUpper(s), strings.ToUpper(sub))
}

// SavePlan persists plan state for later import/verify.
func SavePlan(plan *PlanResult) error {
	state := &MigrationState{
		MigrationID: plan.MigrationID,
		Org:         plan.Org,
		Database:    plan.Database,
		Branch:      plan.Branch,
		InputPath:   plan.InputPath,
		Method:      plan.RecommendedMethod,
		Phase:       PhasePlanned,
	}
	return SaveState(state)
}

// StartNextSteps returns agent next steps after start or start --dry-run.
func StartNextSteps(migrationID, database, method string, dryRun bool) []NextStep {
	if dryRun {
		cmd := fmt.Sprintf("pscale import d1 start --migration-id %s --database %s", migrationID, database)
		if method != "" {
			cmd += fmt.Sprintf(" --method %s", method)
		}
		cmd += " --force"
		return []NextStep{
			{
				Tool:    "import_d1_start",
				Command: cmd,
				Reason:  "Run the import after preview",
			},
		}
	}
	return []NextStep{
		{
			Tool:    "import_d1_verify",
			Command: fmt.Sprintf("pscale import d1 verify --migration-id %s --database %s", migrationID, database),
			Reason:  "Verify row counts, sequences, and content after import",
		},
	}
}
