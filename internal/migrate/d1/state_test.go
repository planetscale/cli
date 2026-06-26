package d1

import (
	"testing"
)

func TestMigrationPhaseTransitions(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	org, database, branch := "acme", "mydb", "main"
	migrationID := "testphase123"

	plan := &PlanResult{
		MigrationID: migrationID,
		Org:         org,
		Database:    database,
		Branch:      branch,
		InputPath:   testFixture(t),
	}
	if err := SavePlan(plan); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	state, err := LoadState(org, database, branch, migrationID)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase != PhasePlanned {
		t.Fatalf("phase = %q, want %q", state.Phase, PhasePlanned)
	}

	opts := ImportOptions{
		Org:         org,
		Database:    database,
		Branch:      branch,
		MigrationID: migrationID,
		InputPath:   plan.InputPath,
		Method:      MethodPgloader,
	}
	if err := saveImportMigrationState(opts, PhaseImporting, ""); err != nil {
		t.Fatalf("saveImportMigrationState importing: %v", err)
	}
	state, err = LoadState(org, database, branch, migrationID)
	if err != nil {
		t.Fatalf("LoadState importing: %v", err)
	}
	if state.Phase != PhaseImporting {
		t.Fatalf("phase = %q, want %q", state.Phase, PhaseImporting)
	}

	if err := SetMigrationPhase(org, database, branch, migrationID, PhaseImported); err != nil {
		t.Fatalf("SetMigrationPhase imported: %v", err)
	}
	if err := SetMigrationPhase(org, database, branch, migrationID, PhaseVerified); err != nil {
		t.Fatalf("SetMigrationPhase verified: %v", err)
	}
	if err := Complete(org, database, branch, migrationID); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	state, err = LoadState(org, database, branch, migrationID)
	if err != nil {
		t.Fatalf("LoadState complete: %v", err)
	}
	if state.Phase != PhaseComplete {
		t.Fatalf("phase = %q, want %q", state.Phase, PhaseComplete)
	}
}

func TestSaveImportMigrationStateFailed(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	org, database, branch := "acme", "mydb", "main"
	migrationID := "testfailed456"
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
	}
	if err := saveImportMigrationState(opts, PhaseFailed, "/tmp/test.sqlite"); err != nil {
		t.Fatalf("saveImportMigrationState failed: %v", err)
	}

	state, err := LoadState(org, database, branch, migrationID)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase != PhaseFailed {
		t.Fatalf("phase = %q, want %q", state.Phase, PhaseFailed)
	}
	if state.SQLitePath != "/tmp/test.sqlite" {
		t.Fatalf("sqlite path = %q", state.SQLitePath)
	}
}
