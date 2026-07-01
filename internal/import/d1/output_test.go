package d1

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/printer"
)

func TestLintResponseSetsErrorEnvelope(t *testing.T) {
	result := &LintResult{
		InputPath:    "/tmp/export.sql",
		TableCount:   1,
		ErrorCount:   1,
		WarningCount: 2,
		Issues: []Issue{{
			Code:        "VIRTUAL_TABLE",
			Severity:    SeverityError,
			Table:       "fts",
			Remediation: "Virtual tables are not supported",
		}},
	}

	resp := LintResponse(result)
	if resp.Status != "error" {
		t.Fatalf("status = %q, want error", resp.Status)
	}
	if resp.Error == nil {
		t.Fatal("expected structured error")
	}
	if resp.Error.Code != ErrCodeLintBlocked {
		t.Fatalf("error code = %q, want %q", resp.Error.Code, ErrCodeLintBlocked)
	}
	if len(resp.Issues) != 1 {
		t.Fatalf("issues = %d, want 1", len(resp.Issues))
	}
}

func TestDoctorResponseIncludesChecksWhenNotReady(t *testing.T) {
	result := &DoctorResult{
		Ready: false,
		Checks: []DoctorCheck{{
			Name:        "pgloader",
			Status:      checkFail,
			Message:     "pgloader not found",
			Remediation: pgloaderInstallRemediation,
		}},
	}

	resp := DoctorResponse(result)
	if resp.Status != "error" {
		t.Fatalf("status = %q, want error", resp.Status)
	}
	if resp.Error == nil || resp.Error.Code != ErrCodePrereqFailed {
		t.Fatalf("error = %#v, want prereq_failed", resp.Error)
	}
	data, ok := resp.Data.(*DoctorResult)
	if !ok || data == nil {
		t.Fatalf("data = %T, want *DoctorResult", resp.Data)
	}
	if len(data.Checks) != 1 || data.Checks[0].Name != "pgloader" {
		t.Fatalf("checks = %#v", data.Checks)
	}
}

func TestPrintHumanResponseDoctorFailureIncludesChecks(t *testing.T) {
	resp := DoctorResponse(&DoctorResult{
		Ready: false,
		Checks: []DoctorCheck{{
			Name:    "pgloader",
			Status:  checkFail,
			Message: "pgloader not found",
		}},
	})

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)
	PrintHumanResponse(p, resp)

	out := buf.String()
	for _, want := range []string{
		"Status: error",
		"pgloader: fail",
		"Ready: false",
		ErrCodePrereqFailed,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestPrintHumanResponseIncludesLintIssuesOnError(t *testing.T) {
	resp := LintResponse(&LintResult{
		TableCount: 1,
		ErrorCount: 1,
		Issues: []Issue{{
			Code:        "VIRTUAL_TABLE",
			Severity:    SeverityError,
			Table:       "fts",
			Remediation: "Virtual tables are not supported",
		}},
	})

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)
	PrintHumanResponse(p, resp)

	out := buf.String()
	for _, want := range []string{
		"Status: error",
		"Errors: 1",
		"[error] VIRTUAL_TABLE",
		"Virtual tables are not supported",
		ErrCodeLintBlocked,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestPrintHumanResponseStatusShowsMigrationPhase(t *testing.T) {
	state := &MigrationState{
		MigrationID:  "abc123",
		Database:     "mydb",
		Branch:       "main",
		Phase:        PhaseImported,
		Method:       "pgloader",
		InputPath:    "/tmp/export.sql",
		LoadedTables: []string{"users", "posts"},
		UpdatedAt:    time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC),
	}

	resp := StatusResponse(state)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)
	PrintHumanResponse(p, resp)

	out := buf.String()
	for _, want := range []string{
		"Status: ok (imported)",
		"Migration ID: abc123",
		"Method: pgloader",
		"Tables loaded: 2",
		"Input: /tmp/export.sql",
		"Updated: 2026-06-29T12:00:00Z",
		"Next steps:",
		"pscale import d1 verify mydb --migration-id abc123 --input \"/tmp/export.sql\"",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestStatusResponseSetsPhase(t *testing.T) {
	state := &MigrationState{
		MigrationID: "abc123",
		Database:    "mydb",
		Branch:      "main",
		Phase:       PhaseVerified,
	}

	resp := StatusResponse(state)
	if len(resp.NextSteps) != 1 {
		t.Fatalf("next_steps = %d, want 1", len(resp.NextSteps))
	}
	if !strings.Contains(resp.NextSteps[0].Command, "import d1 complete mydb") {
		t.Fatalf("next step = %q, want complete command", resp.NextSteps[0].Command)
	}
	if resp.Phase != PhaseVerified {
		t.Fatalf("phase = %q, want %q", resp.Phase, PhaseVerified)
	}
	if resp.Command != "status" {
		t.Fatalf("command = %q, want status", resp.Command)
	}
	if resp.MigrationID != "abc123" {
		t.Fatalf("migration_id = %q, want abc123", resp.MigrationID)
	}
}
