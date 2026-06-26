package d1

import (
	"errors"
	"fmt"
	"strings"
)

// ErrCode constants for structured errors.
const (
	ErrCodeVirtualTable        = "VIRTUAL_TABLE"
	ErrCodeMissingInput        = "MISSING_INPUT"
	ErrCodeMissingTool         = "MISSING_TOOL"
	ErrCodeInvalidInput        = "INVALID_INPUT"
	ErrCodeImportFailed        = "IMPORT_FAILED"
	ErrCodeVerifyFailed        = "VERIFY_FAILED"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodePrereqFailed        = "PREREQ_FAILED"
	ErrCodeLintBlocked         = "LINT_BLOCKED"
	ErrCodeDestinationConflict = "DESTINATION_CONFLICT"
)

const (
	wranglerMissingRemediation = "Install wrangler, use npx wrangler d1 export, or pass --input if you already have a dump."
	pgloaderInstallRemediation = "Install pgloader (brew install pgloader on macOS; see https://pgloader.readthedocs.io/en/latest/install.html for other platforms)"
	lintBlockedRemediation     = "Fix lint errors or run `pscale import d1 lint` for details; use `import d1 start --dry-run` for a read-only preview"
)

type MigrationError struct {
	Info ErrorInfo
}

func (e *MigrationError) Error() string {
	return e.Info.Message
}

func migrationErr(err error) (*MigrationError, bool) {
	var me *MigrationError
	if errors.As(err, &me) {
		return me, true
	}
	return nil, false
}

func newMigrationError(code, message, remediation string) *MigrationError {
	return &MigrationError{
		Info: ErrorInfo{
			Code:        code,
			Message:     message,
			Remediation: remediation,
		},
	}
}

func lintBlockedReason(errorCount int) string {
	return fmt.Sprintf("lint reported %d error(s); fix or use import d1 lint for details", errorCount)
}

func errMissingInput(path string) error {
	return newMigrationError(
		ErrCodeMissingInput,
		fmt.Sprintf("input file not found: %s", path),
		"Run `pscale import d1 export`, export with wrangler/npx, or pass an existing dump with --input",
	)
}

func errMissingTool(name, remediation string) error {
	return newMigrationError(
		ErrCodeMissingTool,
		fmt.Sprintf("required tool not found: %s", name),
		remediation,
	)
}

func errExistingImportTables(tables []string) error {
	return newMigrationError(
		ErrCodeDestinationConflict,
		fmt.Sprintf("destination already has tables from this import: %s", strings.Join(tables, ", ")),
		"Use a new branch, drop the conflicting tables, or choose a database without overlapping table names before importing",
	)
}

// ErrLintBlocked returns a structured error when lint errors block import.
func ErrLintBlocked(reason string) error {
	return newMigrationError(ErrCodeLintBlocked, reason, lintBlockedRemediation)
}
