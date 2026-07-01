package d1

import (
	"context"
	"testing"
)

func TestDoctor_RequiresPgloader(t *testing.T) {
	if _, err := FindPgloader(); err == nil {
		t.Skip("pgloader installed")
	}

	result, err := Doctor(context.Background())
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	if result.Ready {
		t.Fatal("expected doctor not ready without pgloader")
	}

	var pgloaderCheck DoctorCheck
	for _, c := range result.Checks {
		if c.Name == "pgloader" {
			pgloaderCheck = c
			break
		}
	}
	if pgloaderCheck.Status != checkFail {
		t.Fatalf("pgloader check status = %q, want %q", pgloaderCheck.Status, checkFail)
	}

	if err := DoctorReadinessError(result); err == nil {
		t.Fatal("expected readiness error")
	} else {
		requireMigrationErr(t, err, ErrCodePrereqFailed)
	}
}

func TestImport_RequiresPgloader(t *testing.T) {
	if _, err := FindPgloader(); err == nil {
		t.Skip("pgloader installed")
	}

	result, err := Import(context.Background(), nil, nil, ImportOptions{
		InputPath: testFixture(t),
		Org:       "acme",
		Database:  "mydb",
	}, nil)
	if err == nil {
		t.Fatal("expected missing pgloader error")
	}
	requireMigrationErr(t, err, ErrCodeMissingTool)
	if result == nil {
		t.Fatal("expected import result on failure")
	}
	if result.MigrationID == "" {
		t.Fatal("expected migration_id in failure result")
	}
	if result.Lint == nil || result.Plan == nil {
		t.Fatal("expected lint and plan in failure result")
	}
}
