package d1

import (
	"strings"
	"testing"
)

func TestParseColumnDefaultBeforeNotNull(t *testing.T) {
	col := parseColumn("active INTEGER DEFAULT 1 NOT NULL")
	if col.DefaultValue != "1" {
		t.Fatalf("default = %q, want 1", col.DefaultValue)
	}
	if !col.NotNull {
		t.Fatal("expected NOT NULL")
	}
}

func TestParseColumnDefaultStringNotNullNotConstraint(t *testing.T) {
	col := parseColumn("status TEXT DEFAULT 'value NOT NULL'")
	if col.DefaultValue != "'value NOT NULL'" {
		t.Fatalf("default = %q, want quoted literal", col.DefaultValue)
	}
	if col.NotNull {
		t.Fatal("NOT NULL inside default string must not set column constraint")
	}
}

func TestParseColumnCheckNotNullNotConstraint(t *testing.T) {
	col := parseColumn("status TEXT CHECK (status IS NOT NULL)")
	if col.NotNull {
		t.Fatal("NOT NULL inside CHECK must not set column constraint")
	}
}

func TestTrimDefaultClause(t *testing.T) {
	cases := map[string]string{
		"1 NOT NULL":              "1",
		"'draft' NOT NULL UNIQUE": "'draft'",
		"CURRENT_TIMESTAMP":       "CURRENT_TIMESTAMP",
	}
	for in, want := range cases {
		if got := trimDefaultClause(in); got != want {
			t.Fatalf("trimDefaultClause(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseColumnUniqueConstraint(t *testing.T) {
	col := parseColumn("email TEXT NOT NULL UNIQUE")
	if !col.Unique {
		t.Fatal("expected column-level UNIQUE constraint")
	}

	col = parseColumn("unique_token TEXT NOT NULL")
	if col.Unique {
		t.Fatalf("identifier unique_token should not be treated as UNIQUE constraint")
	}

	col = parseColumn("unique_id INTEGER PRIMARY KEY")
	if col.Unique {
		t.Fatalf("identifier unique_id should not be treated as UNIQUE constraint")
	}
}

func TestParseColumnCapturesCheckExpression(t *testing.T) {
	col := parseColumn("age INTEGER CHECK (age >= 0)")
	if len(col.CheckExprs) != 1 || col.CheckExprs[0] != "age >= 0" {
		t.Fatalf("CheckExprs = %#v, want [\"age >= 0\"]", col.CheckExprs)
	}
	if col.NotNull {
		t.Fatal("CHECK clause must not set NOT NULL")
	}
}

func TestParseColumnCheckThenNotNull(t *testing.T) {
	col := parseColumn("age INTEGER CHECK (age >= 0) NOT NULL")
	if len(col.CheckExprs) != 1 || col.CheckExprs[0] != "age >= 0" {
		t.Fatalf("CheckExprs = %#v", col.CheckExprs)
	}
	if !col.NotNull {
		t.Fatal("expected NOT NULL after CHECK clause to still be detected")
	}
}

func TestParseColumnGeneratedStored(t *testing.T) {
	col := parseColumn("total REAL GENERATED ALWAYS AS (price * qty) STORED")
	if col.Type != "REAL" {
		t.Fatalf("Type = %q, want REAL", col.Type)
	}
	if col.GeneratedExpr != "price * qty" {
		t.Fatalf("GeneratedExpr = %q, want %q", col.GeneratedExpr, "price * qty")
	}
	if col.GeneratedMode != "STORED" {
		t.Fatalf("GeneratedMode = %q, want STORED", col.GeneratedMode)
	}
	if col.DefaultValue != "" {
		t.Fatalf("generated column must not also parse a DEFAULT, got %q", col.DefaultValue)
	}
}

func TestParseColumnGeneratedVirtualDefaultMode(t *testing.T) {
	col := parseColumn("total REAL AS (price * qty)")
	if col.GeneratedExpr != "price * qty" {
		t.Fatalf("GeneratedExpr = %q, want %q", col.GeneratedExpr, "price * qty")
	}
	if col.GeneratedMode != "VIRTUAL" {
		t.Fatalf("GeneratedMode = %q, want VIRTUAL (SQLite's default when omitted)", col.GeneratedMode)
	}
}

func TestParseColumnGeneratedThenNotNull(t *testing.T) {
	col := parseColumn("total REAL GENERATED ALWAYS AS (price * qty) STORED NOT NULL")
	if col.GeneratedExpr != "price * qty" {
		t.Fatalf("GeneratedExpr = %q", col.GeneratedExpr)
	}
	if !col.NotNull {
		t.Fatal("expected NOT NULL after generated clause to still be detected")
	}
}

// Precision and scale must survive column parsing (DECIMAL(10,2) stays intact) so the
// Postgres type mapping can carry them over as NUMERIC(10,2).
func TestParseColumnPreservesDecimalPrecision(t *testing.T) {
	col := parseColumn("amount DECIMAL(10,2) NOT NULL")
	if col.Type != "DECIMAL(10,2)" {
		t.Fatalf("Type = %q, want DECIMAL(10,2)", col.Type)
	}
	if !col.NotNull {
		t.Fatal("expected NOT NULL detected after DECIMAL(10,2)")
	}
}

func TestExtractGeneratedClauseNoMatch(t *testing.T) {
	cleaned, expr, mode := extractGeneratedClause("NOT NULL UNIQUE")
	if expr != "" || mode != "" || cleaned != "NOT NULL UNIQUE" {
		t.Fatalf("got cleaned=%q expr=%q mode=%q", cleaned, expr, mode)
	}
}

func TestExtractCheckClausesMultiple(t *testing.T) {
	cleaned, checks := extractCheckClauses("CHECK (a > 0) NOT NULL")
	if len(checks) != 1 || checks[0] != "a > 0" {
		t.Fatalf("checks = %#v", checks)
	}
	if strings.Contains(cleaned, "CHECK") {
		t.Fatalf("cleaned still contains CHECK: %q", cleaned)
	}
}

func TestParseTableBodyIgnoresSmuggledSQLAfterClose(t *testing.T) {
	ddl := `CREATE TABLE b ( x INTEGER REFERENCES a(id) ON DELETE CASCADE); DROP TABLE users; CREATE TABLE dummy (z int );`
	cols, constraints := parseTableBody(ddl)
	if len(constraints) != 0 {
		t.Fatalf("constraints = %#v", constraints)
	}
	if len(cols) != 1 {
		t.Fatalf("cols = %#v", cols)
	}
	if cols[0].ForeignKey != `REFERENCES a(id) ON DELETE CASCADE` {
		t.Fatalf("ForeignKey = %q, want REFERENCES without smuggled SQL", cols[0].ForeignKey)
	}
	if strings.Contains(strings.ToUpper(cols[0].ForeignKey), "DROP") {
		t.Fatalf("ForeignKey must not include injected DROP: %q", cols[0].ForeignKey)
	}
}

// TestMatchingParenEndClosesStringAtBackslash is a regression test for
// planetscale/surfaces#4141. PostgreSQL (standard_conforming_strings=on, the default since
// 9.1) and SQLite do not treat backslash as a string-literal escape character: '\' is a
// complete, valid string literal containing one backslash. matchingParenEnd used to treat
// backslash as an escape (`s[i-1] != '\\'`), so it kept scanning past this literal's real end
// looking for a closing quote, absorbing the ')' that a real SQL parser closes CHECK(...) and
// CREATE TABLE(...) with. That let a later ';' in the dump start a new, attacker-controlled
// statement that psql would execute. The fix must close the string at the first quote that
// isn't doubled, exactly like PostgreSQL/SQLite do, regardless of a preceding backslash.
func TestMatchingParenEndClosesStringAtBackslash(t *testing.T) {
	s := `(name <> '\'))`
	end, ok := matchingParenEnd(s, 0)
	if !ok {
		t.Fatalf("matchingParenEnd(%q) = not found, want a match", s)
	}
	want := strings.Index(s, ")")
	if end != want {
		t.Fatalf("matchingParenEnd(%q) = %d, want %d (the first ')' right after the closed string literal)", s, end, want)
	}
}

// TestParseTableBodyBackslashQuoteCheckDoesNotSmuggleSQL is the end-to-end regression test
// for planetscale/surfaces#4141: a CHECK expression containing the literal '\' used to fool
// matchingParenEnd into treating the attacker's "); DROP TABLE ...; --" tail as still being
// inside the string literal, pulling it into the parsed column list instead of stopping at
// the real close of CREATE TABLE(...).
func TestParseTableBodyBackslashQuoteCheckDoesNotSmuggleSQL(t *testing.T) {
	ddl := `CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT CHECK (name <> '\')); DROP TABLE users; --'));`
	cols, constraints := parseTableBody(ddl)
	if len(constraints) != 0 {
		t.Fatalf("constraints = %#v, want none", constraints)
	}
	if len(cols) != 2 {
		t.Fatalf("cols = %#v, want 2 columns (id, name)", cols)
	}
	if cols[1].Name != "name" {
		t.Fatalf("cols[1].Name = %q, want %q", cols[1].Name, "name")
	}
	if len(cols[1].CheckExprs) != 1 || cols[1].CheckExprs[0] != `name <> '\'` {
		t.Fatalf("CheckExprs = %#v, want [%q]", cols[1].CheckExprs, `name <> '\'`)
	}
	for _, col := range cols {
		for _, check := range col.CheckExprs {
			if strings.Contains(strings.ToUpper(check), "DROP") {
				t.Fatalf("CheckExprs must not include smuggled DROP: %q", check)
			}
		}
	}
}

// TestMatchingParenEndRealBackslashNotNearQuoteBoundary confirms the fix doesn't regress
// legitimate CHECK expressions that contain real backslashes elsewhere in a string literal
// (i.e. not immediately before the closing quote, the pattern above).
func TestMatchingParenEndRealBackslashNotNearQuoteBoundary(t *testing.T) {
	s := `(path <> 'C:\Users\foo')`
	end, ok := matchingParenEnd(s, 0)
	if !ok {
		t.Fatalf("matchingParenEnd(%q) = not found, want a match", s)
	}
	if want := len(s) - 1; end != want {
		t.Fatalf("matchingParenEnd(%q) = %d, want %d (the final ')')", s, end, want)
	}
}
