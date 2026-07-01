package importcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/import/d1"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func newD1TestHelper(t *testing.T) (*cmdutil.Helper, *bytes.Buffer) {
	t.Helper()

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{}, nil
		},
	}
	return ch, &buf
}

func d1FixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Clean(filepath.Join("..", "..", "import", "d1", "testdata", "sample_d1_export.sql"))
}

func executeD1Cmd(t *testing.T, cmd *cobra.Command, args ...string) error {
	t.Helper()
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	return cmd.Execute()
}

func assertJSONField(t *testing.T, buf *bytes.Buffer, field string, want any) {
	t.Helper()
	got := jsonField(t, buf, field)
	if got != want {
		t.Fatalf("%s = %v, want %v", field, got, want)
	}
}

func jsonField(t *testing.T, buf *bytes.Buffer, field string) any {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal output: %v\nbody: %s", err, buf.String())
	}
	got, ok := resp[field]
	if !ok {
		t.Fatalf("response missing %q\nbody: %s", field, buf.String())
	}
	return got
}

func jsonStatus(t *testing.T, buf *bytes.Buffer) string {
	t.Helper()
	got, ok := jsonField(t, buf, "status").(string)
	if !ok {
		t.Fatalf("status field is %T, want string", jsonField(t, buf, "status"))
	}
	return got
}

func TestParseDatabaseBranch(t *testing.T) {
	database, branch := parseDatabaseBranch([]string{"mydb"})
	if database != "mydb" || branch != "main" {
		t.Fatalf("got (%q, %q), want (mydb, main)", database, branch)
	}

	database, branch = parseDatabaseBranch([]string{"mydb", "dev"})
	if database != "mydb" || branch != "dev" {
		t.Fatalf("got (%q, %q), want (mydb, dev)", database, branch)
	}
}

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

func TestCountDataTablesSkipsORMMetadata(t *testing.T) {
	got := countDataTables([]d1.TablePlan{
		{Name: "users"},
		{Name: "_prisma_migrations"},
		{Name: "posts"},
	})
	if got != 2 {
		t.Fatalf("countDataTables = %d, want 2", got)
	}
}
