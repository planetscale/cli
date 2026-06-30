package importcmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/import/d1"
	"github.com/planetscale/cli/internal/printer"
)

func TestD1CompleteCmd(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	client, err := ps.NewClient(
		ps.WithBaseURL(srv.URL),
		ps.WithAccessToken("token"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	const migrationID = "completecmd123"
	fixture := d1FixturePath(t)
	if err := d1.SavePlan(&d1.PlanResult{
		MigrationID: migrationID,
		Org:         "acme",
		Database:    "mydb",
		Branch:      "main",
		InputPath:   fixture,
	}); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if err := d1.SetMigrationPhase("acme", "mydb", "main", migrationID, d1.PhaseVerified); err != nil {
		t.Fatalf("SetMigrationPhase: %v", err)
	}

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return client, nil
		},
	}

	cmd := d1CompleteCmd(ch)
	cmd.SetArgs([]string{"mydb", "--migration-id", migrationID, "--force"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	assertJSONField(t, &buf, "command", "complete")
	assertJSONField(t, &buf, "status", "ok")
	assertJSONField(t, &buf, "migration_id", migrationID)
	if !strings.Contains(buf.String(), "reminder") {
		t.Fatalf("expected reminder in complete JSON output:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "next_steps") {
		t.Fatalf("expected next_steps in complete JSON output:\n%s", buf.String())
	}
}

func TestD1CompleteCmdRequiresMigrationID(t *testing.T) {
	ch, _ := newD1TestHelper(t)

	cmd := d1CompleteCmd(ch)
	if err := executeD1Cmd(t, cmd, "mydb"); err == nil {
		t.Fatal("expected error when --migration-id is missing")
	}
}
