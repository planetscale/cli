package importcmd

import (
	"testing"
)

func TestD1StartCmdDryRun(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	ch, buf := newD1TestHelper(t)
	fixture := d1FixturePath(t)

	cmd := d1StartCmd(ch)
	if err := executeD1Cmd(t, cmd, "mydb", "--input", fixture, "--dry-run", "--force"); err != nil {
		t.Fatalf("execute: %v", err)
	}

	assertJSONField(t, buf, "command", "start")
	assertJSONField(t, buf, "status", "dry_run")
}

func TestD1StartCmdRequiresInput(t *testing.T) {
	ch, _ := newD1TestHelper(t)

	cmd := d1StartCmd(ch)
	if err := executeD1Cmd(t, cmd, "mydb"); err == nil {
		t.Fatal("expected error when --input is missing")
	}
}
