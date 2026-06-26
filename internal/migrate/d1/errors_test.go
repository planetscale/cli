package d1

import "testing"

func requireMigrationErr(t *testing.T, err error, code string) {
	t.Helper()
	me, ok := migrationErr(err)
	if !ok {
		t.Fatalf("expected MigrationError, got %T: %v", err, err)
	}
	if me.Info.Code != code {
		t.Fatalf("code = %q, want %q", me.Info.Code, code)
	}
}
