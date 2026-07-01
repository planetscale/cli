package d1

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateInputPath ensures a user-supplied path is safe to read.
func ValidateInputPath(path string) (string, error) {
	if path == "" {
		return "", newMigrationError(ErrCodeMissingInput, "input path is required", "Pass --input with a D1 SQL export file")
	}
	if strings.ContainsAny(path, "\x00\n\r;") {
		return "", newMigrationError(ErrCodeInvalidInput, "invalid characters in input path", "Use a simple file path without newlines or semicolons")
	}

	clean := filepath.Clean(path)
	info, err := os.Stat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errMissingInput(clean)
		}
		return "", fmt.Errorf("stat input: %w", err)
	}
	if info.IsDir() {
		return "", newMigrationError(ErrCodeInvalidInput, "input path is a directory", "Pass a .sql export file path")
	}
	return clean, nil
}

// NormalizeInputPath validates path and returns an absolute path for stable state comparisons.
func NormalizeInputPath(path string) (string, error) {
	clean, err := ValidateInputPath(path)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return clean, nil
	}
	return abs, nil
}

func normalizePathForCompare(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return filepath.Clean(abs)
	}
	return eval
}

func validateInputPathAgainstState(provided, saved string) error {
	if provided == "" || saved == "" {
		return nil
	}
	if normalizePathForCompare(provided) != normalizePathForCompare(saved) {
		return newMigrationError(
			ErrCodeInvalidInput,
			fmt.Sprintf("input path %q does not match migration state %q", provided, saved),
			"Use the same --input as the original import or omit --input to use saved state",
		)
	}
	return nil
}

// DefaultSQLitePath returns a sqlite path adjacent to the dump.
func DefaultSQLitePath(dumpPath string) string {
	base := filepath.Base(dumpPath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	return filepath.Join(filepath.Dir(dumpPath), name+".sqlite")
}
