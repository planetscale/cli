package d1

import (
	"fmt"
	"unicode"
)

const postgresMaxIdentifierBytes = 63

func lintIdentifiers(table TableSchema) []Issue {
	var issues []Issue
	issues = append(issues, lintIdentifier(table.Name, table.Name, "")...)
	for _, col := range table.Columns {
		issues = append(issues, lintIdentifier(table.Name, col.Name, col.Name)...)
	}
	return issues
}

func lintIdentifier(table, name, column string) []Issue {
	var issues []Issue
	if len(name) > postgresMaxIdentifierBytes {
		target := "table"
		if column != "" {
			target = "column"
		}
		issues = append(issues, Issue{
			Code:        "IDENTIFIER_TOO_LONG",
			Severity:    SeverityError,
			Table:       table,
			Column:      column,
			Message:     fmt.Sprintf("%s name %q exceeds PostgreSQL 63-byte identifier limit (%d bytes)", target, name, len(name)),
			Remediation: "Rename the " + target + " in SQLite before export, or use quoted identifiers that fit within 63 bytes in PostgreSQL",
		})
	}
	if hasMixedCaseIdentifier(name) {
		issues = append(issues, Issue{
			Code:        "MIXED_CASE_IDENTIFIER",
			Severity:    SeverityWarning,
			Table:       table,
			Column:      column,
			Message:     fmt.Sprintf("identifier %q contains uppercase letters", name),
			Remediation: "PostgreSQL folds unquoted identifiers to lowercase; prefer snake_case in D1 exports to avoid case mismatches during import",
		})
	}
	return issues
}

func hasMixedCaseIdentifier(name string) bool {
	hasUpper := false
	hasLower := false
	for _, r := range name {
		if unicode.IsUpper(r) {
			hasUpper = true
		}
		if unicode.IsLower(r) {
			hasLower = true
		}
	}
	return hasUpper && hasLower
}
