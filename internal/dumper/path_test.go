package dumper

import (
	"path/filepath"
	"testing"
)

func TestDumpOutputPathRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	if _, err := dumpOutputPath(dir, "db", "../../etc/passwd", "-schema.sql"); err == nil {
		t.Fatal("expected traversal table name to be rejected")
	}
	if _, err := dumpOutputPath(dir, "../db", "t", "-schema.sql"); err == nil {
		t.Fatal("expected traversal database name to be rejected")
	}

	got, err := dumpOutputPath(dir, "db", "users", "-schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "db.users-schema.sql")
	if got != want {
		t.Fatalf("dumpOutputPath = %q, want %q", got, want)
	}
	if filepath.Dir(got) != dir {
		t.Fatalf("path escaped outdir: %q", got)
	}
}
