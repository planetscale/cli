package d1

import "testing"

func TestLintIdentifiers(t *testing.T) {
	longName := stringsRepeat("a", 64)
	table := TableSchema{
		Name: longName,
		Columns: []ColumnSchema{
			{Name: "ok_col", Type: "TEXT"},
			{Name: stringsRepeat("b", 64), Type: "TEXT"},
			{Name: "UserId", Type: "INTEGER"},
		},
	}

	issues := lintIdentifiers(table)
	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d: %#v", len(issues), issues)
	}
	if issues[0].Code != "IDENTIFIER_TOO_LONG" || issues[0].Severity != SeverityError {
		t.Fatalf("table issue = %#v", issues[0])
	}
	if issues[1].Code != "IDENTIFIER_TOO_LONG" || issues[1].Column == "" {
		t.Fatalf("column issue = %#v", issues[1])
	}
	if issues[2].Code != "MIXED_CASE_IDENTIFIER" || issues[2].Column != "UserId" {
		t.Fatalf("mixed case issue = %#v", issues[2])
	}
}

func TestHasMixedCaseIdentifier(t *testing.T) {
	if hasMixedCaseIdentifier("user_id") {
		t.Fatal("snake_case should not flag")
	}
	if hasMixedCaseIdentifier("USER_ID") {
		t.Fatal("all caps should not flag")
	}
	if !hasMixedCaseIdentifier("UserId") {
		t.Fatal("mixed case should flag")
	}
}

func stringsRepeat(s string, n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = s[0]
	}
	return string(out)
}
