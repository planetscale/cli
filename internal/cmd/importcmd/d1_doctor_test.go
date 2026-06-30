package importcmd

import (
	"encoding/json"
	"testing"
)

func TestD1DoctorCmd(t *testing.T) {
	ch, buf := newD1TestHelper(t)

	cmd := d1DoctorCmd(ch)
	err := executeD1Cmd(t, cmd)
	if buf.Len() == 0 {
		t.Fatal("expected JSON output")
	}
	assertJSONField(t, buf, "command", "doctor")
	if err == nil {
		assertJSONField(t, buf, "status", "ok")
		return
	}

	var resp map[string]any
	if unmarshalErr := json.Unmarshal(buf.Bytes(), &resp); unmarshalErr != nil {
		t.Fatalf("unmarshal output: %v\nbody: %s", unmarshalErr, buf.String())
	}
	if resp["status"] != "error" {
		t.Fatalf("status = %v, want error", resp["status"])
	}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("data = %T, want object", resp["data"])
	}
	checks, ok := data["checks"].([]any)
	if !ok || len(checks) == 0 {
		t.Fatalf("checks = %v, want non-empty array", data["checks"])
	}
}
