package importcmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/migrate/d1"
	"github.com/planetscale/cli/internal/printer"
)

func TestWriteD1ErrorUsesConsistentExitCode(t *testing.T) {
	resp := d1.LintResponse(&d1.LintResult{
		TableCount: 1,
		ErrorCount: 1,
		Issues: []d1.Issue{{
			Code:        "VIRTUAL_TABLE",
			Severity:    d1.SeverityError,
			Table:       "fts",
			Remediation: "Virtual tables are not supported",
		}},
	})

	for _, format := range []printer.Format{printer.Human, printer.JSON} {
		t.Run(format.String(), func(t *testing.T) {
			var buf bytes.Buffer
			p := printer.NewPrinter(&format)
			if format == printer.Human {
				p.SetHumanOutput(&buf)
			} else {
				p.SetResourceOutput(&buf)
			}

			err := writeD1(&cmdutil.Helper{Printer: p}, resp)
			if err == nil {
				t.Fatal("expected error")
			}

			var cmdErr *cmdutil.Error
			if !errors.As(err, &cmdErr) {
				t.Fatalf("expected *cmdutil.Error, got %T: %v", err, err)
			}
			if cmdErr.ExitCode != cmdutil.ActionRequestedExitCode {
				t.Fatalf("exit code = %d, want %d", cmdErr.ExitCode, cmdutil.ActionRequestedExitCode)
			}
			if !cmdErr.Printed {
				t.Fatal("expected output to be marked printed")
			}
			if buf.Len() == 0 {
				t.Fatal("expected response output")
			}
		})
	}
}
