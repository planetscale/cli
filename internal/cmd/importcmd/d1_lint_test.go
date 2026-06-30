package importcmd

import (
	"testing"
)

func TestD1LintCmd(t *testing.T) {
	ch, buf := newD1TestHelper(t)
	fixture := d1FixturePath(t)

	cmd := d1LintCmd(ch)
	if err := executeD1Cmd(t, cmd, "--input", fixture); err != nil {
		t.Fatalf("execute: %v", err)
	}

	assertJSONField(t, buf, "command", "lint")
	if status := jsonStatus(t, buf); status != "ok" && status != "warning" {
		t.Fatalf("status = %q, want ok or warning", status)
	}
}

func TestD1LintCmdRequiresInput(t *testing.T) {
	ch, _ := newD1TestHelper(t)

	cmd := d1LintCmd(ch)
	if err := executeD1Cmd(t, cmd); err == nil {
		t.Fatal("expected error when --input is missing")
	}
}
