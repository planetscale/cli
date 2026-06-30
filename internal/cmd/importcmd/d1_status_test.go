package importcmd

import (
	"testing"

	"github.com/planetscale/cli/internal/import/d1"
)

func TestD1StatusCmd(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	const migrationID = "statuscmd123"
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

	ch, buf := newD1TestHelper(t)
	cmd := d1StatusCmd(ch)
	if err := executeD1Cmd(t, cmd, "mydb", "--migration-id", migrationID); err != nil {
		t.Fatalf("execute: %v", err)
	}

	assertJSONField(t, buf, "command", "status")
	assertJSONField(t, buf, "status", "ok")
	assertJSONField(t, buf, "migration_id", migrationID)
}

func TestD1StatusCmdRequiresMigrationID(t *testing.T) {
	ch, _ := newD1TestHelper(t)

	cmd := d1StatusCmd(ch)
	if err := executeD1Cmd(t, cmd, "mydb"); err == nil {
		t.Fatal("expected error when --migration-id is missing")
	}
}
