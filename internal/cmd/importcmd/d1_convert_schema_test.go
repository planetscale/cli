package importcmd

import (
	"path/filepath"
	"testing"
)

func TestD1ConvertSchemaCmd(t *testing.T) {
	ch, buf := newD1TestHelper(t)
	fixture := d1FixturePath(t)
	output := filepath.Join(t.TempDir(), "schema.postgres.sql")

	cmd := d1ConvertSchemaCmd(ch)
	if err := executeD1Cmd(t, cmd, "--input", fixture, "--output", output); err != nil {
		t.Fatalf("execute: %v", err)
	}

	assertJSONField(t, buf, "command", "convert-schema")
	assertJSONField(t, buf, "status", "ok")
}

func TestD1ConvertSchemaCmdRequiresInput(t *testing.T) {
	ch, _ := newD1TestHelper(t)

	cmd := d1ConvertSchemaCmd(ch)
	if err := executeD1Cmd(t, cmd); err == nil {
		t.Fatal("expected error when --input is missing")
	}
}
