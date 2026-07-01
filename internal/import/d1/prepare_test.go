package d1

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/planetscale/cli/internal/printer"
)

func TestPrepareImport(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	prepared, err := PrepareImport(ImportOptions{
		InputPath: testFixture(t),
		Org:       "acme",
		Database:  "mydb",
		Branch:    "main",
	})
	if err != nil {
		t.Fatalf("PrepareImport: %v", err)
	}
	if !prepared.CanProceed {
		t.Fatalf("expected can proceed, blocked: %s", prepared.BlockedReason)
	}
	if prepared.MigrationID == "" {
		t.Fatal("expected migration id")
	}
	if prepared.Lint == nil || prepared.Plan == nil {
		t.Fatal("expected lint and plan in prepare result")
	}
	if prepared.Method != prepared.Plan.RecommendedMethod {
		t.Fatalf("method mismatch: %q vs %q", prepared.Method, prepared.Plan.RecommendedMethod)
	}
}

func TestImport_BlocksOnLintErrors(t *testing.T) {
	prepared := &ImportPrepareResult{
		MigrationID:   "mig-test",
		Method:        MethodPgloader,
		CanProceed:    false,
		BlockedReason: "lint reported 1 error(s); fix or use import d1 lint for details",
		Lint: &LintResult{
			ErrorCount: 1,
			Issues: []Issue{{
				Code:     "TEST",
				Severity: SeverityError,
				Message:  "blocked for test",
			}},
		},
	}

	result, err := Import(context.Background(), nil, nil, ImportOptions{DryRun: true}, prepared)
	if err == nil {
		t.Fatal("expected lint blocked error")
	}
	requireMigrationErr(t, err, ErrCodeLintBlocked)
	if result == nil || result.CanProceed {
		t.Fatal("expected result with can_proceed false")
	}
}

func TestPrepareImportRejectsMissingMigrationState(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	_, err := PrepareImport(ImportOptions{
		InputPath:   testFixture(t),
		Org:         "acme",
		Database:    "mydb",
		Branch:      "main",
		MigrationID: "missing-migration-id",
	})
	requireMigrationErr(t, err, ErrCodeNotFound)
}

func TestPrepareImportAcceptsEquivalentInputPath(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	org, database, branch := "acme", "mydb", "main"
	absFixture := testFixture(t)

	prepared, err := PrepareImport(ImportOptions{
		InputPath: absFixture,
		Org:       org,
		Database:  database,
		Branch:    branch,
	})
	if err != nil {
		t.Fatalf("initial PrepareImport: %v", err)
	}

	dir := filepath.Dir(absFixture)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	again, err := PrepareImport(ImportOptions{
		InputPath:   "./" + filepath.Base(absFixture),
		Org:         org,
		Database:    database,
		Branch:      branch,
		MigrationID: prepared.MigrationID,
	})
	if err != nil {
		t.Fatalf("PrepareImport with relative path: %v", err)
	}
	if again.MigrationID != prepared.MigrationID {
		t.Fatalf("migration = %q, want %q", again.MigrationID, prepared.MigrationID)
	}
}

func TestPrepareImportRejectsCorruptMigrationState(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	store, err := NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	migrationID := "corrupt-migration-id"
	path := store.statePath("acme", "mydb", "main", migrationID)
	if err := os.WriteFile(path, []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	_, err = PrepareImport(ImportOptions{
		InputPath:   testFixture(t),
		Org:         "acme",
		Database:    "mydb",
		Branch:      "main",
		MigrationID: migrationID,
	})
	if err == nil {
		t.Fatal("expected corrupt migration state to fail")
	}
}

func TestPrintStartPreview(t *testing.T) {
	prepared, err := PrepareImport(ImportOptions{
		InputPath: testFixture(t),
		Org:       "acme",
		Database:  "mydb",
	})
	if err != nil {
		t.Fatalf("PrepareImport: %v", err)
	}

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)
	PrintStartPreview(p, prepared)
	out := buf.String()
	for _, want := range []string{"Import preview", "Lint:", "Plan:", prepared.MigrationID} {
		if !strings.Contains(out, want) {
			t.Fatalf("preview missing %q:\n%s", want, out)
		}
	}
}
