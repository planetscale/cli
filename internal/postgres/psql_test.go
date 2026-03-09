package postgres

import (
	"os"
	"testing"
)

// Note: Testing exported functions only.
// parsePsqlVersion is unexported and tested indirectly through CheckPsqlVersion.

func TestFindPsqlPath(t *testing.T) {
	// This test just ensures the function is callable
	// Actual psql binary may or may not exist depending on system
	path, err := FindPsqlPath()
	if err != nil {
		// psql not installed - that's ok for test
		if path != "" {
			t.Error("FindPsqlPath should return empty path on error")
		}
		return
	}

	// If psql found, verify it's a valid path
	if path == "" {
		t.Error("FindPsqlPath should return non-empty path on success")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("FindPsqlPath returned invalid path: %v", err)
	}
}

func TestFindPgDumpPath(t *testing.T) {
	// This test just ensures the function is callable
	path, err := FindPgDumpPath()
	if err != nil {
		// pg_dump not installed - that's ok for test
		if path != "" {
			t.Error("FindPgDumpPath should return empty path on error")
		}
		return
	}

	// If pg_dump found, verify it's a valid path
	if path == "" {
		t.Error("FindPgDumpPath should return non-empty path on success")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("FindPgDumpPath returned invalid path: %v", err)
	}
}
