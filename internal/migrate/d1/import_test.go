package d1

import (
	"testing"
)

func TestIsEphemeralImportRole(t *testing.T) {
	if !isEphemeralImportRole("pscale_api_abc123") {
		t.Fatal("expected pscale_api role to match")
	}
	if isEphemeralImportRole("postgres") {
		t.Fatal("postgres should not match")
	}
}
