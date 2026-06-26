package d1

import (
	"fmt"

	"github.com/planetscale/cli/internal/printer"
)

// ImportPrepareResult is lint + plan output used before and during import.
type ImportPrepareResult struct {
	MigrationID   string      `json:"migration_id"`
	Method        string      `json:"method"`
	Lint          *LintResult `json:"lint"`
	Plan          *PlanResult `json:"plan"`
	CanProceed    bool        `json:"can_proceed"`
	BlockedReason string      `json:"blocked_reason,omitempty"`
}

// PrepareImport runs lint and resolves or creates a migration plan without touching Postgres.
func PrepareImport(opts ImportOptions) (*ImportPrepareResult, error) {
	if _, err := ValidateInputPath(opts.InputPath); err != nil {
		return nil, err
	}
	if opts.MigrationID != "" {
		if _, err := LoadState(opts.Org, opts.Database, opts.Branch, opts.MigrationID); err != nil {
			return nil, err
		}
	}
	if _, err := FindPgloader(); err != nil {
		return nil, err
	}

	lintResult, err := Lint(opts.InputPath)
	if err != nil {
		return nil, err
	}

	method := opts.Method
	if method == "" {
		size, err := FileSize(opts.InputPath)
		if err != nil {
			return nil, err
		}
		method = recommendMethod(size)
	}

	plan, err := resolvePlan(opts, method, lintResult)
	if err != nil {
		return nil, err
	}

	if opts.Method != "" {
		plan.RecommendedMethod = opts.Method
	}
	method = plan.RecommendedMethod

	out := &ImportPrepareResult{
		MigrationID: plan.MigrationID,
		Method:      method,
		Lint:        lintResult,
		Plan:        plan,
		CanProceed:  lintResult.ErrorCount == 0,
	}
	if !out.CanProceed {
		out.BlockedReason = lintBlockedReason(lintResult.ErrorCount)
	}
	return out, nil
}

func resolvePlan(opts ImportOptions, method string, lint *LintResult) (*PlanResult, error) {
	if opts.MigrationID == "" {
		return createAndSavePlan(PlanOptions{
			InputPath: opts.InputPath,
			Org:       opts.Org,
			Database:  opts.Database,
			Branch:    opts.Branch,
			Method:    method,
			Lint:      lint,
		})
	}

	state, err := LoadState(opts.Org, opts.Database, opts.Branch, opts.MigrationID)
	if err != nil {
		return nil, err
	}

	if opts.InputPath != "" && state.InputPath != "" && state.InputPath != opts.InputPath {
		return nil, newMigrationError(
			ErrCodeInvalidInput,
			fmt.Sprintf("input path %q does not match planned import %q", opts.InputPath, state.InputPath),
			"Use the same --input as a prior start preview or omit --migration-id to start fresh",
		)
	}

	inputPath := opts.InputPath
	if inputPath == "" {
		inputPath = state.InputPath
	}

	plan, err := Plan(PlanOptions{
		InputPath:   inputPath,
		Org:         opts.Org,
		Database:    opts.Database,
		Branch:      opts.Branch,
		Method:      method,
		MigrationID: state.MigrationID,
		Lint:        lint,
	})
	if err != nil {
		return nil, err
	}
	if state.Method != "" {
		plan.RecommendedMethod = state.Method
	}
	return plan, nil
}

func createAndSavePlan(opts PlanOptions) (*PlanResult, error) {
	plan, err := Plan(opts)
	if err != nil {
		return nil, err
	}
	if err := SavePlan(plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func importResultFromPrepare(prepared *ImportPrepareResult, dryRun bool) *ImportResult {
	return &ImportResult{
		MigrationID: prepared.MigrationID,
		Method:      prepared.Method,
		DryRun:      dryRun,
		Lint:        prepared.Lint,
		Plan:        prepared.Plan,
		CanProceed:  prepared.CanProceed,
	}
}

// BlockedStartResponse builds the start error envelope when lint blocks import.
func BlockedStartResponse(prepared *ImportPrepareResult, dryRun bool) Response {
	resp := ErrorResponse("start", ErrLintBlocked(prepared.BlockedReason))
	if prepared.Lint != nil {
		resp.Issues = prepared.Lint.Issues
	}
	resp.Data = ImportResult{
		MigrationID: prepared.MigrationID,
		Method:      prepared.Method,
		DryRun:      dryRun,
		Lint:        prepared.Lint,
		Plan:        prepared.Plan,
		CanProceed:  false,
	}
	resp.MigrationID = prepared.MigrationID
	return resp
}

// PrintStartPreview writes a human-readable lint/plan summary before import confirmation.
func PrintStartPreview(p *printer.Printer, prepared *ImportPrepareResult) {
	if prepared == nil {
		return
	}
	p.Println("\nImport preview")
	if prepared.Lint != nil {
		p.Printf("  Lint: %d error(s), %d warning(s)\n", prepared.Lint.ErrorCount, prepared.Lint.WarningCount)
		for _, issue := range prepared.Lint.Issues {
			if issue.Severity != SeverityError && issue.Severity != SeverityWarning {
				continue
			}
			loc := issue.Table
			if issue.Column != "" {
				loc += "." + issue.Column
			}
			if loc != "" {
				loc = " " + loc
			}
			p.Printf("    [%s] %s%s: %s\n", issue.Severity, issue.Code, loc, previewMessage(issue))
		}
	}
	if prepared.Plan != nil {
		sizeMB := float64(prepared.Plan.EstimatedSizeBytes) / (1024 * 1024)
		p.Printf("  Plan: migration_id %s, method %s, %.1f MB, %d tables\n",
			prepared.Plan.MigrationID,
			prepared.Plan.RecommendedMethod,
			sizeMB,
			len(prepared.Plan.Tables),
		)
	}
	if prepared.BlockedReason != "" {
		p.Printf("  Blocked: %s\n", prepared.BlockedReason)
	}
	p.Println()
}

func previewMessage(issue Issue) string {
	if issue.Message != "" {
		return issue.Message
	}
	return issue.Remediation
}
