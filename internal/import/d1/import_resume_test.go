package d1

import (
	"testing"
)

func TestImportResumeEnabled(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	org, database, branch := "acme", "mydb", "main"
	migrationID := "resume001"
	if err := SavePlan(&PlanResult{
		MigrationID: migrationID,
		Org:         org,
		Database:    database,
		Branch:      branch,
		InputPath:   testFixture(t),
	}); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	opts := ImportOptions{
		Org:         org,
		Database:    database,
		Branch:      branch,
		MigrationID: migrationID,
	}

	if importResumeEnabled(opts) {
		t.Fatal("expected no resume before any tables loaded")
	}

	if err := updateMigrationState(org, database, branch, migrationID, func(state *MigrationState) {
		state.Phase = PhaseFailed
		state.SchemaApplied = true
	}); err != nil {
		t.Fatalf("update state schema applied: %v", err)
	}
	if !importResumeEnabled(opts) {
		t.Fatal("expected resume when failed after schema applied")
	}

	if err := updateMigrationState(org, database, branch, migrationID, func(state *MigrationState) {
		state.SchemaApplied = false
		state.LoadedTables = []string{"users"}
	}); err != nil {
		t.Fatalf("update state: %v", err)
	}
	if !importResumeEnabled(opts) {
		t.Fatal("expected resume when failed with loaded tables")
	}

	if err := SetMigrationPhase(org, database, branch, migrationID, PhasePlanned); err != nil {
		t.Fatalf("SetMigrationPhase: %v", err)
	}
	if importResumeEnabled(opts) {
		t.Fatal("expected no resume from planned phase even with loaded_tables")
	}
}

func TestAppendLoadedTableAndReset(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	org, database, branch := "acme", "mydb", "main"
	migrationID := "resume002"
	if err := SavePlan(&PlanResult{
		MigrationID: migrationID,
		Org:         org,
		Database:    database,
		Branch:      branch,
		InputPath:   testFixture(t),
	}); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	opts := ImportOptions{
		Org:         org,
		Database:    database,
		Branch:      branch,
		MigrationID: migrationID,
		InputPath:   testFixture(t),
		Method:      MethodPgloader,
		DBName:      "myapp",
	}
	if err := appendLoadedTable(opts, "users"); err != nil {
		t.Fatalf("appendLoadedTable: %v", err)
	}
	if err := appendLoadedTable(opts, "users"); err != nil {
		t.Fatalf("appendLoadedTable duplicate: %v", err)
	}
	if err := appendLoadedTable(opts, "posts"); err != nil {
		t.Fatalf("appendLoadedTable posts: %v", err)
	}

	state, err := LoadState(org, database, branch, migrationID)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if len(state.LoadedTables) != 2 {
		t.Fatalf("loaded_tables = %v, want [users posts]", state.LoadedTables)
	}

	if err := resetImportProgress(opts, PhaseImporting, "/tmp/stage.sqlite"); err != nil {
		t.Fatalf("resetImportProgress: %v", err)
	}
	state, err = LoadState(org, database, branch, migrationID)
	if err != nil {
		t.Fatalf("LoadState after reset: %v", err)
	}
	if len(state.LoadedTables) != 0 {
		t.Fatalf("expected loaded_tables cleared, got %v", state.LoadedTables)
	}
	if state.SchemaApplied {
		t.Fatal("expected schema_applied cleared on reset")
	}
	if state.DBName != "myapp" {
		t.Fatalf("db_name = %q, want myapp", state.DBName)
	}
	if state.SQLitePath != "/tmp/stage.sqlite" {
		t.Fatalf("sqlite_path = %q", state.SQLitePath)
	}
}
