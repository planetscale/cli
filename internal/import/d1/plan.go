package d1

import (
	"fmt"
	"regexp"
	"slices"
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
	nameSet := make(map[string]bool)
	for _, t := range tables {
		names = append(names, t.Name)
		nameSet[t.Name] = true
	}

	deps := make(map[string][]string)
	for _, t := range tables {
		for _, col := range t.Columns {
			if ref := parseFKReference(col.ForeignKey); ref != "" && nameSet[ref] && !slices.Contains(deps[t.Name], ref) {
				deps[t.Name] = append(deps[t.Name], ref)
			}
		}
		for _, ref := range parseTableFKReferences(t.RawDDL) {
			if nameSet[ref] && !slices.Contains(deps[t.Name], ref) {
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
	ref := strings.Trim(parts[0], "`\"'")
	if paren := strings.Index(ref, "("); paren >= 0 {
		ref = ref[:paren]
	}
	return ref
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
func StartNextSteps(migrationID, database, branch, method, inputPath string, dryRun bool) []NextStep {
	target := CLICommandTarget(database, branch)
	if dryRun {
		cmd := fmt.Sprintf("pscale import d1 start %s --migration-id %s", target, migrationID)
		if inputPath != "" {
			cmd += fmt.Sprintf(" --input %q", inputPath)
		}
		if method != "" {
			cmd += fmt.Sprintf(" --method %s", method)
		}
		return []NextStep{
			{
				Command: cmd,
				Reason:  "Run the import after preview",
			},
		}
	}
	verifyCmd := fmt.Sprintf("pscale import d1 verify %s --migration-id %s", target, migrationID)
	if inputPath != "" {
		verifyCmd += fmt.Sprintf(" --input %q", inputPath)
	}
	return []NextStep{
		{
			Command: verifyCmd,
			Reason:  "Verify row counts, sequences, and content after import",
		},
	}
}

// StatusNextSteps returns the recommended next command for the current migration phase.
func StatusNextSteps(state *MigrationState) []NextStep {
	if state == nil {
		return nil
	}

	target := CLICommandTarget(state.Database, state.Branch)
	migrationID := state.MigrationID

	switch state.Phase {
	case PhasePlanned:
		cmd := fmt.Sprintf("pscale import d1 start %s --migration-id %s", target, migrationID)
		if state.InputPath != "" {
			cmd += fmt.Sprintf(" --input %q", state.InputPath)
		}
		if state.Method != "" {
			cmd += fmt.Sprintf(" --method %s", state.Method)
		}
		return []NextStep{{
			Command: cmd,
			Reason:  "Run the import after dry-run preview",
		}}
	case PhaseImporting:
		return []NextStep{{
			Command: fmt.Sprintf("pscale import d1 status %s --migration-id %s", target, migrationID),
			Reason:  "Import in progress; check status again when it finishes",
		}}
	case PhaseImported:
		cmd := fmt.Sprintf("pscale import d1 verify %s --migration-id %s", target, migrationID)
		if state.InputPath != "" {
			cmd += fmt.Sprintf(" --input %q", state.InputPath)
		}
		return []NextStep{{
			Command: cmd,
			Reason:  "Verify row counts and content after import",
		}}
	case PhaseVerified:
		return []NextStep{{
			Command: fmt.Sprintf("pscale import d1 complete %s --migration-id %s", target, migrationID),
			Reason:  "Mark migration complete after successful verify",
		}}
	case PhaseFailed:
		cmd := fmt.Sprintf("pscale import d1 start %s --migration-id %s", target, migrationID)
		if state.InputPath != "" {
			cmd += fmt.Sprintf(" --input %q", state.InputPath)
		}
		return []NextStep{{
			Command: cmd,
			Reason:  "Retry or resume the failed import",
		}}
	default:
		return nil
	}
}

// VerifyNextSteps returns next steps after a successful verify.
func VerifyNextSteps(migrationID, database, branch string) []NextStep {
	target := CLICommandTarget(database, branch)
	return []NextStep{{
		Command: fmt.Sprintf("pscale import d1 complete %s --migration-id %s", target, migrationID),
		Reason:  "Mark migration complete after successful verify",
	}}
}

// CLICommandTarget formats database and branch for pscale import d1 command examples.
func CLICommandTarget(database, branch string) string {
	if branch == "" || branch == "main" {
		return database
	}
	return database + " " + branch
}
