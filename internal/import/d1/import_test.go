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

func TestIsDefaultPostgresRole(t *testing.T) {
	if !isDefaultPostgresRole("postgres") {
		t.Fatal("expected postgres to match")
	}
	if !isDefaultPostgresRole("postgres.qi39g1yfjxyj") {
		t.Fatal("expected routed postgres role to match")
	}
	if isDefaultPostgresRole("pscale_api_abc123") {
		t.Fatal("pscale_api should not match")
	}
}
