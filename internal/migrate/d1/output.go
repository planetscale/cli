package d1

import (
	"fmt"
	"time"

	"github.com/planetscale/cli/internal/printer"
)

// PrintHumanResponse writes a human-readable success response via the shared printer.
func PrintHumanResponse(p *printer.Printer, resp Response) {
	p.Printf("Status: %s", resp.Status)
	if resp.Phase != "" {
		p.Printf(" (%s)", resp.Phase)
	}
	p.Println()

	if resp.MigrationID != "" {
		p.Printf("Migration ID: %s\n", resp.MigrationID)
	}

	printHumanData(p, resp.Phase, resp.Data)

	if resp.Error != nil {
		p.Printf("\nError [%s]: %s\n", resp.Error.Code, resp.Error.Message)
		if resp.Error.Remediation != "" {
			p.Printf("%s\n", resp.Error.Remediation)
		}
	}

	if len(resp.Issues) > 0 {
		p.Printf("\nIssues (%d):\n", len(resp.Issues))
		for _, issue := range resp.Issues {
			loc := issue.Table
			if issue.Column != "" {
				loc += "." + issue.Column
			}
			p.Printf("  [%s] %s %s: %s\n", issue.Severity, issue.Code, loc, issue.Remediation)
		}
	}

	if len(resp.NextSteps) > 0 {
		p.Println("\nNext steps:")
		for _, step := range resp.NextSteps {
			if step.Command != "" {
				p.Printf("  - %s (%s)\n", step.Command, step.Reason)
			} else {
				p.Printf("  - %s (%s)\n", step.Tool, step.Reason)
			}
		}
	}
}

func printImportResultHuman(p *printer.Printer, r ImportResult) {
	p.Printf("\nMethod: %s", r.Method)
	if r.DryRun {
		p.Print(" (dry run)")
	}
	p.Println()
	if r.Plan != nil {
		sizeMB := float64(r.Plan.EstimatedSizeBytes) / (1024 * 1024)
		p.Printf("Plan: %d tables, %.1f MB estimated\n", len(r.Plan.Tables), sizeMB)
	}
	if r.TablesLoaded > 0 {
		p.Printf("Tables loaded: %d\n", r.TablesLoaded)
	}
	if r.Timings != nil && r.Timings.TotalMs > 0 {
		p.Printf("Total time: %.1fs\n", float64(r.Timings.TotalMs)/1000)
	}
}

func printHumanData(p *printer.Printer, phase string, data any) {
	if data == nil {
		return
	}

	switch phase {
	case "doctor":
		if r, ok := data.(DoctorResult); ok {
			p.Println("\nChecks:")
			for _, c := range r.Checks {
				line := fmt.Sprintf("  %s: %s", c.Name, c.Status)
				if c.Version != "" {
					line += fmt.Sprintf(" (%s)", c.Version)
				}
				p.Println(line)
			}
			p.Printf("Ready: %v\n", r.Ready)
		}
	case "export":
		if r, ok := data.(ExportResult); ok {
			p.Printf("\nExported to %s (%d bytes)\n", r.OutputPath, r.SizeBytes)
		}
	case "lint":
		switch r := data.(type) {
		case LintResult:
			p.Printf("\nTables: %d | Errors: %d | Warnings: %d\n", r.TableCount, r.ErrorCount, r.WarningCount)
		case *LintResult:
			if r != nil {
				p.Printf("\nTables: %d | Errors: %d | Warnings: %d\n", r.TableCount, r.ErrorCount, r.WarningCount)
			}
		}
	case "start":
		switch r := data.(type) {
		case ImportResult:
			printImportResultHuman(p, r)
		case *ImportResult:
			if r != nil {
				printImportResultHuman(p, *r)
			}
		}
	case "verify":
		if r, ok := data.(VerifyResult); ok {
			matched := "no"
			if r.Matched {
				matched = "yes"
			}
			p.Printf("\nMatched: %s\n", matched)
		}
	case "status":
		if r, ok := data.(MigrationState); ok {
			p.Printf("\nPhase: %s | Updated: %s\n", r.Phase, r.UpdatedAt.Format(time.RFC3339))
		}
	case "convert-schema":
		if m, ok := data.(map[string]any); ok {
			p.Println()
			p.Printf("  Input: %v\n", m["input"])
			p.Printf("  Output: %v\n", m["output"])
			p.Printf("  Tables: %v\n", m["table_count"])
		}
	case "complete":
		if m, ok := data.(map[string]string); ok {
			p.Println()
			p.Printf("  Migration ID: %s\n", m["migration_id"])
			p.Printf("  Status: %s\n", m["status"])
		}
	}
}

// OKResponse builds a success response.
func OKResponse(phase string, data any, next []NextStep) Response {
	return Response{
		Status:    "ok",
		Phase:     phase,
		Data:      data,
		NextSteps: next,
	}
}

// ErrorResponse builds an error response from an error.
func ErrorResponse(phase string, err error) Response {
	resp := Response{
		Status: "error",
		Phase:  phase,
	}
	if me, ok := migrationErr(err); ok {
		resp.Error = &me.Info
	} else {
		resp.Error = &ErrorInfo{
			Code:    ErrCodeImportFailed,
			Message: err.Error(),
		}
	}
	return resp
}

// LintResponse builds the lint command envelope with status derived from issue severity.
func LintResponse(result *LintResult) Response {
	resp := OKResponse("lint", result, LintNextSteps(result))
	resp.Issues = result.Issues
	if result.ErrorCount > 0 {
		resp.Status = "error"
		resp.Error = &ErrorInfo{
			Code:        ErrCodeLintBlocked,
			Message:     lintBlockedReason(result.ErrorCount),
			Remediation: lintBlockedRemediation,
		}
	} else if result.WarningCount > 0 {
		resp.Status = "warning"
	}
	return resp
}
