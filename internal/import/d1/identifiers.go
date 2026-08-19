package d1

import (
	"fmt"
	"hash/fnv"
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
	if _, err := quotePgloaderIdentifier(name); err != nil {
		target := "table"
		if column != "" {
			target = "column"
		}
		issues = append(issues, Issue{
			Code:        "UNSAFE_IDENTIFIER",
			Severity:    SeverityError,
			Table:       table,
			Column:      column,
			Message:     fmt.Sprintf("%s name %q cannot be represented safely in a pgloader control file", target, name),
			Remediation: "Rename the " + target + " in SQLite before export so it does not contain quotes or control characters",
		})
	}
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

// fitPostgresIdentifier truncates name to PostgreSQL's 63-byte identifier limit. When
// truncation is required, a stable hash suffix is appended so distinct long names do not
// collide after truncation.
func fitPostgresIdentifier(name string) string {
	if len(name) <= postgresMaxIdentifierBytes {
		return name
	}
	sum := fnv.New32a()
	_, _ = sum.Write([]byte(name))
	suffix := fmt.Sprintf("_%08x", sum.Sum32())
	keep := postgresMaxIdentifierBytes - len(suffix)
	if keep < 1 {
		return suffix[1 : postgresMaxIdentifierBytes+1]
	}
	return name[:keep] + suffix
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
