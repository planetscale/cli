package d1

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateInputPathAgainstStateEquivalentPaths(t *testing.T) {
	dir := t.TempDir()
	dump := filepath.Join(dir, "export.sql")
	if err := os.WriteFile(dump, []byte("SELECT 1;\n"), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	rel := "./export.sql"
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	if err := validateInputPathAgainstState(rel, dump); err != nil {
		t.Fatalf("expected equivalent paths to match: %v", err)
	}
}

func TestNormalizeInputPathReturnsAbsolute(t *testing.T) {
	dir := t.TempDir()
	dump := filepath.Join(dir, "export.sql")
	if err := os.WriteFile(dump, []byte("SELECT 1;\n"), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	got, err := NormalizeInputPath(dump)
	if err != nil {
		t.Fatalf("NormalizeInputPath: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("path = %q, want absolute", got)
	}
}
