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
	} else if resp.Command != "" {
		p.Printf(" (%s)", resp.Command)
	}
	p.Println()

	if resp.MigrationID != "" {
		p.Printf("Migration ID: %s\n", resp.MigrationID)
	}

	printHumanData(p, resp.Command, resp.Data)

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

func printVerifyResultHuman(p *printer.Printer, r VerifyResult) {
	matched := "no"
	if r.Matched {
		matched = "yes"
	}
	p.Printf("\nMatched: %s\n", matched)
}

func printMigrationStateHuman(p *printer.Printer, r MigrationState) {
	if r.Method != "" {
		p.Printf("Method: %s\n", r.Method)
	}
	if len(r.LoadedTables) > 0 {
		p.Printf("Tables loaded: %d\n", len(r.LoadedTables))
	}
	if r.InputPath != "" {
		p.Printf("Input: %s\n", r.InputPath)
	}
	if !r.UpdatedAt.IsZero() {
		p.Printf("Updated: %s\n", r.UpdatedAt.Format(time.RFC3339))
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

func printDoctorResultHuman(p *printer.Printer, r DoctorResult) {
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

func printHumanData(p *printer.Printer, command string, data any) {
	if data == nil {
		return
	}

	switch command {
	case "doctor":
		switch r := data.(type) {
		case DoctorResult:
			printDoctorResultHuman(p, r)
		case *DoctorResult:
			if r != nil {
				printDoctorResultHuman(p, *r)
			}
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
		switch r := data.(type) {
		case VerifyResult:
			printVerifyResultHuman(p, r)
		case *VerifyResult:
			if r != nil {
				printVerifyResultHuman(p, *r)
			}
		}
	case "status":
		switch r := data.(type) {
		case MigrationState:
			printMigrationStateHuman(p, r)
		case *MigrationState:
			if r != nil {
				printMigrationStateHuman(p, *r)
			}
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

// StatusResponse builds the status command envelope.
func StatusResponse(state *MigrationState) Response {
	var next []NextStep
	if state != nil {
		next = StatusNextSteps(state)
	}
	resp := OKResponse("status", state, next)
	if state != nil {
		resp.MigrationID = state.MigrationID
		resp.Phase = state.Phase
	}
	return resp
}

// OKResponse builds a success response.
func OKResponse(command string, data any, next []NextStep) Response {
	return Response{
		Status:    "ok",
		Command:   command,
		Data:      data,
		NextSteps: next,
	}
}

// ErrorResponse builds an error response from an error.
func ErrorResponse(command string, err error) Response {
	resp := Response{
		Status:  "error",
		Command: command,
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

// DoctorResponse builds the doctor command envelope, including check details when not ready.
func DoctorResponse(result *DoctorResult) Response {
	resp := OKResponse("doctor", result, DoctorNextSteps(result))
	if result != nil && !result.Ready {
		resp.Status = "error"
		if err := DoctorReadinessError(result); err != nil {
			if me, ok := migrationErr(err); ok {
				resp.Error = &me.Info
			} else {
				resp.Error = &ErrorInfo{
					Code:    ErrCodePrereqFailed,
					Message: err.Error(),
				}
			}
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
