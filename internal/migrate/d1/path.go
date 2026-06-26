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
