package importcmd

import (
	"testing"
)

func TestD1VerifyCmdRequiresMigrationID(t *testing.T) {
	ch, _ := newD1TestHelper(t)

	cmd := d1VerifyCmd(ch)
	if err := executeD1Cmd(t, cmd, "mydb"); err == nil {
		t.Fatal("expected error when --migration-id is missing")
	}
}
