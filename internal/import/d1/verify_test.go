package d1

import (
	"context"
	"strings"
	"testing"
)

func TestVerifyFailsWhenSourceCountsUnavailable(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	_, err := Verify(context.Background(), VerifyOptions{
		DestURI:     "postgresql://u:p@localhost:5432/postgres?sslmode=disable",
		InputPath:   testFixture(t),
		SQLitePath:  "/nonexistent/staging.sqlite",
		MigrationID: "verify001",
	})
	if err == nil {
		t.Fatal("expected error for missing sqlite staging db")
	}
	migrationErr, ok := err.(*MigrationError)
	if !ok {
		t.Fatalf("expected MigrationError, got %T: %v", err, err)
	}
	if migrationErr.Info.Code != ErrCodeVerifyFailed {
		t.Fatalf("code = %q, want %q", migrationErr.Info.Code, ErrCodeVerifyFailed)
	}
	if !strings.Contains(migrationErr.Info.Message, "count source rows") {
		t.Fatalf("message = %q", migrationErr.Info.Message)
	}
}

func TestVerifyUsesDBNameFromState(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	org, database, branch := "acme", "mydb", "main"
	migrationID := "verify002"
	if err := SavePlan(&PlanResult{
		MigrationID: migrationID,
		Org:         org,
		Database:    database,
		Branch:      branch,
		InputPath:   testFixture(t),
	}); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if err := updateMigrationState(org, database, branch, migrationID, func(state *MigrationState) {
		state.DBName = "customdb"
	}); err != nil {
		t.Fatalf("update state: %v", err)
	}

	opts := VerifyOptions{
		Org:         org,
		Database:    database,
		Branch:      branch,
		MigrationID: migrationID,
		DestURI:     "postgresql://u:p@localhost:5432/postgres?sslmode=disable",
		InputPath:   testFixture(t),
		SQLitePath:  "/nonexistent/staging.sqlite",
	}
	if got := ResolveVerifyDBName(opts, false); got != "customdb" {
		t.Fatalf("ResolveVerifyDBName = %q, want customdb", got)
	}
	if got := ResolveVerifyDBName(opts, true); got != "postgres" {
		t.Fatalf("explicit ResolveVerifyDBName = %q, want postgres", got)
	}
}

func TestResolveVerifySQLitePathDefaultsFromInput(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	org, database, branch := "acme", "mydb", "main"
	migrationID := "verify003"
	input := testFixture(t)
	if err := SavePlan(&PlanResult{
		MigrationID: migrationID,
		Org:         org,
		Database:    database,
		Branch:      branch,
		InputPath:   input,
	}); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	_, sqlitePath, err := resolveVerifySQLitePath(VerifyOptions{
		Org:         org,
		Database:    database,
		Branch:      branch,
		MigrationID: migrationID,
	})
	if err != nil {
		t.Fatalf("resolveVerifySQLitePath: %v", err)
	}
	want := DefaultSQLitePath(input)
	if sqlitePath != want {
		t.Fatalf("sqlite path = %q, want %q", sqlitePath, want)
	}
}

func TestResolveVerifySQLitePathBackfillsInputPathWhenSQLiteExplicit(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	org, database, branch := "acme", "mydb", "main"
	migrationID := "verify004"
	input := testFixture(t)
	if err := SavePlan(&PlanResult{
		MigrationID: migrationID,
		Org:         org,
		Database:    database,
		Branch:      branch,
		InputPath:   input,
	}); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	gotOpts, sqlitePath, err := resolveVerifySQLitePath(VerifyOptions{
		Org:         org,
		Database:    database,
		Branch:      branch,
		MigrationID: migrationID,
		SQLitePath:  "/nonexistent/staging.sqlite",
	})
	if err != nil {
		t.Fatalf("resolveVerifySQLitePath: %v", err)
	}
	if sqlitePath != "/nonexistent/staging.sqlite" {
		t.Fatalf("sqlite path = %q, want explicit --sqlite path", sqlitePath)
	}
	if gotOpts.InputPath != input {
		t.Fatalf("InputPath = %q, want backfilled %q", gotOpts.InputPath, input)
	}
}

func TestResolveVerifySQLitePathFailsOnBadMigrationIDWithInput(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	_, _, err := resolveVerifySQLitePath(VerifyOptions{
		Org:         "acme",
		Database:    "mydb",
		Branch:      "main",
		MigrationID: "missing-migration",
		InputPath:   testFixture(t),
	})
	requireMigrationErr(t, err, ErrCodeNotFound)
}
