package d1

import (
	"context"
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

	if enabled, err := importSchemaResumeEnabled(context.Background(), opts, ""); err != nil || enabled {
		t.Fatalf("expected no schema resume before schema is applied (enabled=%v err=%v)", enabled, err)
	}
	if enabled, err := importDataResumeEnabled(context.Background(), opts, ""); err != nil || enabled {
		t.Fatalf("expected no data resume before any tables loaded (enabled=%v err=%v)", enabled, err)
	}

	if err := updateMigrationState(org, database, branch, migrationID, func(state *MigrationState) {
		state.Phase = PhaseFailed
		state.SchemaApplied = true
	}); err != nil {
		t.Fatalf("update state schema applied: %v", err)
	}
	if enabled, err := importSchemaResumeEnabled(context.Background(), opts, ""); err != nil || enabled {
		t.Fatalf("expected schema resume deferred until destination is known (enabled=%v err=%v)", enabled, err)
	}
	if !shouldPreserveImportProgress(context.Background(), opts, "") {
		t.Fatal("expected import progress preserved when failed after schema applied")
	}

	if err := updateMigrationState(org, database, branch, migrationID, func(state *MigrationState) {
		state.SchemaApplied = false
		state.LoadedTables = []string{"users"}
	}); err != nil {
		t.Fatalf("update state: %v", err)
	}
	if enabled, err := importDataResumeEnabled(context.Background(), opts, ""); err != nil || enabled {
		t.Fatalf("expected data resume deferred until destination is known (enabled=%v err=%v)", enabled, err)
	}
	if !shouldPreserveImportProgress(context.Background(), opts, "") {
		t.Fatal("expected import progress preserved when failed with loaded tables")
	}

	if err := SetMigrationPhase(org, database, branch, migrationID, PhasePlanned); err != nil {
		t.Fatalf("SetMigrationPhase: %v", err)
	}
	if enabled, err := importDataResumeEnabled(context.Background(), opts, ""); err != nil || enabled {
		t.Fatalf("expected no data resume from planned phase even with loaded_tables (enabled=%v err=%v)", enabled, err)
	}
}

func TestSkipLoadedTablesForResume(t *testing.T) {
	loaded := []string{"users", "posts", "comments"}
	withRows := map[string]struct{}{
		"users": {},
		"posts": {},
	}
	got := skipLoadedTablesForResume(loaded, withRows)
	want := []string{"users", "posts"}
	if len(got) != len(want) {
		t.Fatalf("skip tables = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("skip tables = %v, want %v", got, want)
		}
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
