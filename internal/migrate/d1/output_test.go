package d1

import (
	"bytes"
	"strings"
	"testing"

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
