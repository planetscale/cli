package d1

import "testing"

func TestCLICommandTarget(t *testing.T) {
	if got := CLICommandTarget("mydb", "main"); got != "mydb" {
		t.Fatalf("got %q, want mydb", got)
	}
	if got := CLICommandTarget("mydb", "dev"); got != "mydb dev" {
		t.Fatalf("got %q, want mydb dev", got)
	}
}

func TestStartNextStepsUsesPositionalTarget(t *testing.T) {
	steps := StartNextSteps("abc123", "mydb", "dev", "pgloader", "./d1-export.sql", false)
	if len(steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(steps))
	}
	want := `pscale import d1 verify mydb dev --migration-id abc123 --input "./d1-export.sql"`
	if steps[0].Command != want {
		t.Fatalf("command = %q, want %q", steps[0].Command, want)
	}
}

func TestStartNextStepsDryRunOmitsForce(t *testing.T) {
	steps := StartNextSteps("abc123", "mydb", "main", "pgloader", "./d1-export.sql", true)
	if len(steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(steps))
	}
	want := `pscale import d1 start mydb --migration-id abc123 --input "./d1-export.sql" --method pgloader`
	if steps[0].Command != want {
		t.Fatalf("command = %q, want %q", steps[0].Command, want)
	}
}

func TestStatusNextStepsImported(t *testing.T) {
	steps := StatusNextSteps(&MigrationState{
		MigrationID: "abc123",
		Database:    "import-9gb",
		Branch:      "main",
		InputPath:   "/tmp/export.sql",
		Phase:       PhaseImported,
	})
	if len(steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(steps))
	}
	want := `pscale import d1 verify import-9gb --migration-id abc123 --input "/tmp/export.sql"`
	if steps[0].Command != want {
		t.Fatalf("command = %q, want %q", steps[0].Command, want)
	}
}

func TestStatusNextStepsCompleteHasNoSteps(t *testing.T) {
	steps := StatusNextSteps(&MigrationState{
		MigrationID: "abc123",
		Database:    "import-9gb",
		Branch:      "main",
		Phase:       PhaseComplete,
	})
	if len(steps) != 0 {
		t.Fatalf("steps = %d, want 0", len(steps))
	}
}
