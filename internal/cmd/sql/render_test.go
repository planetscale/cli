package sql

import (
	"bytes"
	"strings"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/sqlquery"
)

func TestStripVerticalTerminator(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		wantQuery    string
		wantVertical bool
	}{
		{"no terminator", "SELECT 1", "SELECT 1", false},
		{"trailing G", `SHOW REPLICA STATUS\G`, "SHOW REPLICA STATUS", true},
		{"trailing G with whitespace", "SELECT 1 \\G \n", "SELECT 1", true},
		{"lowercase g strips without vertical", `SELECT 1\g`, "SELECT 1", false},
		{"semicolon untouched", "SELECT 1;", "SELECT 1;", false},
		{"backslash G inside string literal", `SELECT '\G stuff' FROM t`, `SELECT '\G stuff' FROM t`, false},
		{"only terminator", `\G`, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotQuery, gotVertical := stripVerticalTerminator(tt.query)
			if gotQuery != tt.wantQuery || gotVertical != tt.wantVertical {
				t.Fatalf("stripVerticalTerminator(%q) = (%q, %v), want (%q, %v)",
					tt.query, gotQuery, gotVertical, tt.wantQuery, tt.wantVertical)
			}
		})
	}
}

func TestRenderTablePreservesColumnOrderAndAlignment(t *testing.T) {
	columns := []string{"id", "name", "deleted_at"}
	rows := []map[string]any{
		{"id": int64(1), "name": "alice", "deleted_at": nil},
		{"id": int64(2), "name": "bo", "deleted_at": "2026-01-02"},
	}

	var b bytes.Buffer
	renderTable(&b, columns, rows)

	want := strings.Join([]string{
		"+----+-------+------------+",
		"| id | name  | deleted_at |",
		"+----+-------+------------+",
		"| 1  | alice | NULL       |",
		"| 2  | bo    | 2026-01-02 |",
		"+----+-------+------------+",
		"",
	}, "\n")
	if b.String() != want {
		t.Fatalf("renderTable output:\n%s\nwant:\n%s", b.String(), want)
	}
}

func TestRenderTableMultiLineValue(t *testing.T) {
	columns := []string{"gtid"}
	rows := []map[string]any{{"gtid": "aaaa:1-5,\nbb:1-2"}}

	var b bytes.Buffer
	renderTable(&b, columns, rows)

	want := strings.Join([]string{
		"+-----------+",
		"| gtid      |",
		"+-----------+",
		"| aaaa:1-5, |",
		"| bb:1-2    |",
		"+-----------+",
		"",
	}, "\n")
	if b.String() != want {
		t.Fatalf("renderTable output:\n%s\nwant:\n%s", b.String(), want)
	}
}

func TestRenderTableMultiLineValueKeepsLaterColumnsAligned(t *testing.T) {
	columns := []string{"id", "gtid", "host"}
	rows := []map[string]any{
		{"id": int64(1), "gtid": "a:1-5,\nb:1-2", "host": "h1"},
	}

	var b bytes.Buffer
	renderTable(&b, columns, rows)

	want := strings.Join([]string{
		"+----+--------+------+",
		"| id | gtid   | host |",
		"+----+--------+------+",
		"| 1  | a:1-5, | h1   |",
		"|    | b:1-2  |      |",
		"+----+--------+------+",
		"",
	}, "\n")
	if b.String() != want {
		t.Fatalf("renderTable output:\n%s\nwant:\n%s", b.String(), want)
	}

	// Every physical line must open and close its border.
	for i, line := range strings.Split(strings.TrimRight(b.String(), "\n"), "\n") {
		if !strings.HasPrefix(line, "+") && (!strings.HasPrefix(line, "| ") || !strings.HasSuffix(line, "|")) {
			t.Fatalf("line %d does not close its border: %q", i+1, line)
		}
	}
}

func TestRenderVertical(t *testing.T) {
	columns := []string{"Replica_IO_Running", "Seconds_Behind_Source", "Last_Error"}
	rows := []map[string]any{
		{"Replica_IO_Running": "Yes", "Seconds_Behind_Source": int64(0), "Last_Error": nil},
		{"Replica_IO_Running": "No", "Seconds_Behind_Source": int64(12), "Last_Error": ""},
	}

	var b bytes.Buffer
	renderVertical(&b, columns, rows)

	want := strings.Join([]string{
		"*************************** 1. row ***************************",
		"   Replica_IO_Running: Yes",
		"Seconds_Behind_Source: 0",
		"           Last_Error: NULL",
		"*************************** 2. row ***************************",
		"   Replica_IO_Running: No",
		"Seconds_Behind_Source: 12",
		"           Last_Error: ",
		"",
	}, "\n")
	if b.String() != want {
		t.Fatalf("renderVertical output:\n%s\nwant:\n%s", b.String(), want)
	}
}

func humanHelper(out *bytes.Buffer) *cmdutil.Helper {
	format := printer.Human
	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
		Config:  &config.Config{},
	}
	ch.Printer.SetHumanOutput(out)
	return ch
}

func TestPrintHumanResultRowsAffected(t *testing.T) {
	var out bytes.Buffer
	printHumanResult(humanHelper(&out), &sqlquery.Result{RowsAffected: 3}, false)
	if got, want := out.String(), "Rows affected: 3\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestPrintHumanResultZeroRows(t *testing.T) {
	var out bytes.Buffer
	printHumanResult(humanHelper(&out), &sqlquery.Result{Columns: []string{"id"}}, false)
	if got, want := out.String(), "Returned 0 row(s)\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestPrintHumanResultTable(t *testing.T) {
	var out bytes.Buffer
	result := &sqlquery.Result{
		RowCount: 1,
		Columns:  []string{"id", "name"},
		Rows:     []map[string]any{{"id": int64(1), "name": "alice"}},
	}
	printHumanResult(humanHelper(&out), result, false)

	got := out.String()
	if !strings.Contains(got, "| id | name  |") {
		t.Fatalf("expected table header in output:\n%s", got)
	}
	if !strings.HasSuffix(got, "Returned 1 row(s)\n") {
		t.Fatalf("expected row count after table:\n%s", got)
	}
	if strings.Contains(got, "map[") {
		t.Fatalf("output must not contain Go map syntax:\n%s", got)
	}
}

func TestPrintHumanResultVertical(t *testing.T) {
	var out bytes.Buffer
	result := &sqlquery.Result{
		RowCount: 1,
		Columns:  []string{"id", "name"},
		Rows:     []map[string]any{{"id": int64(1), "name": "alice"}},
	}
	printHumanResult(humanHelper(&out), result, true)

	got := out.String()
	if !strings.Contains(got, "*************************** 1. row ***************************") {
		t.Fatalf("expected vertical row header in output:\n%s", got)
	}
	if !strings.Contains(got, "name: alice") {
		t.Fatalf("expected column line in output:\n%s", got)
	}
}

func TestSQLCmdHasVerticalFlag(t *testing.T) {
	format := printer.Human
	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
		Config:  &config.Config{},
	}
	cmd := SQLCmd(ch)
	if cmd.Flags().Lookup("vertical") == nil {
		t.Fatal("expected --vertical flag on sql subcommand")
	}
}
