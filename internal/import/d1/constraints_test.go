package d1

import (
	"strings"
	"testing"
)

func TestParseTableLevelForeignKey(t *testing.T) {
	cols, refs := parseTableLevelForeignKey(`FOREIGN KEY (entity_id) REFERENCES external_entities(id)`)
	if len(cols) != 1 || cols[0] != "entity_id" {
		t.Fatalf("unexpected columns: %#v", cols)
	}
	refTable, refCol := parseReferencesTarget(refs)
	if refTable != "external_entities" || refCol != "id" {
		t.Fatalf("unexpected ref target: %s.%s", refTable, refCol)
	}
}

func TestColumnFKTargetUsesTableConstraint(t *testing.T) {
	table := TableSchema{
		Name: "entity_links",
		Columns: []ColumnSchema{
			{Name: "entity_id", Type: "TEXT", NotNull: true},
			{Name: "post_id", Type: "INTEGER", NotNull: true},
		},
		Constraints: []string{
			`PRIMARY KEY (entity_id, post_id)`,
			`FOREIGN KEY (entity_id) REFERENCES external_entities(id)`,
			`FOREIGN KEY (post_id) REFERENCES posts(id)`,
		},
	}
	col := table.Columns[0]

	refTable, refCol := columnFKTarget(col, table)
	if refTable != "external_entities" || refCol != "id" {
		t.Fatalf("got %s.%s", refTable, refCol)
	}
}

func TestConvertCheckConstraintRequotesMixedCaseColumn(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE t ("MixedCase" INTEGER, CHECK (MixedCase > 0));`)
	if !strings.Contains(ddl, `CHECK ("MixedCase" > 0)`) {
		t.Fatalf("expected CHECK column reference re-quoted with declared case:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

// TestConvertCheckConstraintFunctionCallNotQuotedAsColumn guards against a bare
// identifier that happens to share its name with a table column being quoted as a
// column reference when it's actually a function call (e.g. a column named "length"
// alongside a CHECK using the length(...) scalar function). Quoting the function name
// produces invalid Postgres DDL like "length"(name) instead of length(name). (length is
// used here rather than count because Postgres disallows aggregate functions like count()
// inside CHECK constraints.)
func TestConvertCheckConstraintFunctionCallNotQuotedAsColumn(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE t (name TEXT, length INTEGER, CHECK (length(name) > 0));`)
	if strings.Contains(ddl, `"length"(`) {
		t.Fatalf("function call length(name) must not be quoted as a column reference:\n%s", ddl)
	}
	if !strings.Contains(ddl, `CHECK (length("name") > 0)`) {
		t.Fatalf(`expected length(...) function call preserved as-is (not quoted) in CHECK:%s`, "\n"+ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertCheckConstraintDoubleQuotedLiterals(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE t (status TEXT, CHECK (status IN ("active", "inactive")));`)
	if !strings.Contains(ddl, `CHECK ("status" IN ('active', 'inactive'))`) {
		t.Fatalf("expected double-quoted CHECK literals converted to string literals:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertReferencesClauseCanonicalizesCase(t *testing.T) {
	sql := `CREATE TABLE Users (id INTEGER PRIMARY KEY);
CREATE TABLE Posts (id INTEGER PRIMARY KEY, user_id INTEGER REFERENCES USERS(ID));
`
	ddl := convertTablesDDL(t, sql)
	if !strings.Contains(ddl, `REFERENCES "Users" ("id")`) {
		t.Fatalf("expected REFERENCES canonicalized to declared table/column case:\n%s", ddl)
	}
	// Verify the canonicalized REFERENCES clause against real Postgres with tables created
	// in dependency order; table load ordering for case-mismatched FK references is a
	// separate, pre-existing concern not covered by this test.
	verifyDDL := `CREATE TABLE "Users" ("id" BIGINT PRIMARY KEY); CREATE TABLE "Posts" ("id" BIGINT PRIMARY KEY, "user_id" BIGINT REFERENCES "Users" ("id"));`
	assertValidPostgresDDL(t, verifyDDL)
}

func TestConvertReferencesClauseKeepsSafeActionTail(t *testing.T) {
	sql := `CREATE TABLE users (id INTEGER PRIMARY KEY);
CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER REFERENCES users(id) ON DELETE CASCADE ON UPDATE SET NULL);
`
	ddl := convertTablesDDL(t, sql)
	if !strings.Contains(ddl, `REFERENCES "users" ("id") ON DELETE CASCADE ON UPDATE SET NULL`) {
		t.Fatalf("expected safe FK action tail preserved:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertReferencesClauseDropsInjectionTail(t *testing.T) {
	got := convertReferencesClause(`REFERENCES users(id)); DROP TABLE users; --`, nil)
	if strings.Contains(strings.ToUpper(got), "DROP TABLE") {
		t.Fatalf("injected DROP must not appear in REFERENCES clause: %q", got)
	}
	if strings.Contains(got, ");") {
		t.Fatalf("escaping parentheses must not appear in REFERENCES clause: %q", got)
	}
	if got != `REFERENCES "users" ("id")` {
		t.Fatalf("got %q, want REFERENCES with action tail stripped", got)
	}

	got = convertReferencesClause(`REFERENCES users(id) ON DELETE CASCADE); DROP TABLE users; --`, nil)
	if strings.Contains(strings.ToUpper(got), "DROP TABLE") || strings.Contains(got, "CASCADE") {
		// Hostile tail must be dropped entirely (not partially trusted).
		t.Fatalf("hostile mixed tail must be dropped: %q", got)
	}
	if got != `REFERENCES "users" ("id")` {
		t.Fatalf("got %q, want REFERENCES without hostile tail", got)
	}

	if got := convertReferencesClause(`NOT A REFERENCES CLAUSE); DROP TABLE users; --`, nil); got != "" {
		t.Fatalf("unparseable REFERENCES must be omitted, got %q", got)
	}
}

func TestConvertTableForeignKeyDropsUnparseableReferences(t *testing.T) {
	for _, clause := range []string{
		`FOREIGN KEY (user_id) REFERENCES invalid`,
		`CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES invalid`,
	} {
		if got := convertTableConstraint(clause, TableSchema{}, nil, nil); got != "" {
			t.Errorf("unparseable table REFERENCES must be omitted, got %q", got)
		}
	}
}

// End-to-end: attacker dumps smuggle "); DROP ...; CREATE TABLE ..." after a real
// REFERENCES ... ON DELETE CASCADE close. Parse must stop at the balanced CREATE TABLE
// close, and conversion must never emit the injected statements into executed DDL.
func TestConvertReferencesTailInjectionEndToEnd(t *testing.T) {
	sql := `CREATE TABLE a (id INTEGER PRIMARY KEY);
CREATE TABLE b ( x INTEGER REFERENCES a(id) ON DELETE CASCADE); DROP TABLE users; CREATE TABLE dummy (z int );
`
	ddl := convertTablesDDL(t, sql)
	upper := strings.ToUpper(ddl)
	if strings.Contains(upper, "DROP TABLE") {
		t.Fatalf("injected DROP must not appear in converted DDL:\n%s", ddl)
	}
	if strings.Contains(upper, `"DUMMY"`) || strings.Contains(ddl, `"dummy"`) {
		t.Fatalf("smuggled CREATE TABLE must not appear in converted DDL:\n%s", ddl)
	}
	if !strings.Contains(ddl, `REFERENCES "a" ("id")`) {
		t.Fatalf("expected safe REFERENCES clause:\n%s", ddl)
	}
	// Legitimate action should survive once the trailing junk is excluded from the body.
	if !strings.Contains(ddl, `ON DELETE CASCADE`) {
		t.Fatalf("expected ON DELETE CASCADE preserved after stripping smuggled SQL:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestQuoteColumnListStripsIndexedColumnModifiers(t *testing.T) {
	got := quoteColumnList("a, b DESC")
	want := `"a", "b"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	ddl := convertTablesDDL(t, `CREATE TABLE t (a INTEGER, b INTEGER, PRIMARY KEY (a, b DESC));`)
	if !strings.Contains(ddl, `PRIMARY KEY ("a", "b")`) {
		t.Fatalf("expected PRIMARY KEY column list without indexed-column modifiers:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertUniqueConstraintDropsOnConflictClause(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE t (a INTEGER, b INTEGER, UNIQUE (a, b) ON CONFLICT REPLACE);`)
	if strings.Contains(strings.ToUpper(ddl), "ON CONFLICT") {
		t.Fatalf("ON CONFLICT clause must not appear in Postgres DDL:\n%s", ddl)
	}
	if !strings.Contains(ddl, `UNIQUE ("a", "b")`) {
		t.Fatalf("expected UNIQUE constraint converted:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertPrimaryKeyConstraintDropsOnConflictClause(t *testing.T) {
	got := convertPrimaryKeyConstraint("PRIMARY KEY (a, b) ON CONFLICT ROLLBACK", TableSchema{})
	want := `PRIMARY KEY ("a", "b")`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// SQLite resolves column names in table constraints case-insensitively, so the local
// column list of PRIMARY KEY / UNIQUE / FOREIGN KEY constraints must be canonicalized to
// the declared case, exactly like the referenced side of REFERENCES already is.
func TestConvertTableConstraintCanonicalizesLocalColumnCase(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE t (id INTEGER, PRIMARY KEY (ID));`)
	if !strings.Contains(ddl, `PRIMARY KEY ("id")`) {
		t.Fatalf("expected PRIMARY KEY column canonicalized to declared case:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)

	ddl = convertTablesDDL(t, `CREATE TABLE t (a INTEGER, b INTEGER, UNIQUE (A, B));`)
	if !strings.Contains(ddl, `UNIQUE ("a", "b")`) {
		t.Fatalf("expected UNIQUE columns canonicalized to declared case:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)

	sql := `CREATE TABLE users (id INTEGER PRIMARY KEY);
CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER, FOREIGN KEY (USER_ID) REFERENCES users(id));
`
	ddl = convertTablesDDL(t, sql)
	if !strings.Contains(ddl, `FOREIGN KEY ("user_id") REFERENCES "users" ("id")`) {
		t.Fatalf("expected FOREIGN KEY local columns canonicalized to declared case:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

// Named constraints (CONSTRAINT <name> <body>) must get the same conversion as unnamed
// ones instead of being passed through verbatim.
func TestConvertNamedConstraint(t *testing.T) {
	ddl := convertTablesDDL(t, `CREATE TABLE t ("MixedCase" INTEGER, CONSTRAINT chk_mixed CHECK (MixedCase > 0));`)
	if !strings.Contains(ddl, `CONSTRAINT "chk_mixed" CHECK ("MixedCase" > 0)`) {
		t.Fatalf("expected named CHECK constraint converted:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)

	sql := `CREATE TABLE Users (id INTEGER PRIMARY KEY);
CREATE TABLE Posts (id INTEGER PRIMARY KEY, user_id INTEGER, CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES USERS(ID));
`
	ddl = convertTablesDDL(t, sql)
	if !strings.Contains(ddl, `ALTER TABLE "Posts" ADD CONSTRAINT "fk_user" FOREIGN KEY ("user_id") REFERENCES "Users" ("id")`) {
		t.Fatalf("expected named FOREIGN KEY constraint deferred and converted:\n%s", ddl)
	}

	ddl = convertTablesDDL(t, `CREATE TABLE t (a INTEGER, b INTEGER, CONSTRAINT uq_ab UNIQUE (a, b) ON CONFLICT REPLACE);`)
	if !strings.Contains(ddl, `CONSTRAINT "uq_ab" UNIQUE ("a", "b")`) {
		t.Fatalf("expected named UNIQUE constraint converted with ON CONFLICT dropped:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

// SQL keywords inside a CHECK expression must never be quoted as column references, even
// when the table has a column with the same name.
func TestConvertCheckExprKeywordsNotQuotedAsColumns(t *testing.T) {
	table := TableSchema{Name: "t", Columns: []ColumnSchema{
		{Name: "a", Type: "INTEGER"},
		{Name: "b", Type: "INTEGER"},
		{Name: "and", Type: "INTEGER"},
		{Name: "end", Type: "INTEGER"},
	}}
	got := convertCheckExpr("a > 0 AND b < 5", table, nil)
	want := `"a" > 0 AND "b" < 5`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	// A keyword-named column can still be referenced when quoted in the source DDL.
	got = convertCheckExpr(`"end" > 0`, table, nil)
	want = `"end" > 0`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// Bracket- and backtick-quoted identifiers are valid SQLite quoting that Postgres
// rejects; both must be converted to double-quoted identifiers with declared case.
func TestConvertCheckExprBracketAndBacktickIdentifiers(t *testing.T) {
	table := TableSchema{Name: "t", Columns: []ColumnSchema{{Name: "Age", Type: "INTEGER"}}}
	cases := map[string]string{
		"[Age] > 0":     `"Age" > 0`,
		"[age] > 0":     `"Age" > 0`,
		"`Age` > 0":     `"Age" > 0`,
		"`age` > 0":     `"Age" > 0`,
		"[unknown] > 0": `"unknown" > 0`,
	}
	for expr, want := range cases {
		if got := convertCheckExpr(expr, table, nil); got != want {
			t.Fatalf("convertCheckExpr(%q) = %q, want %q", expr, got, want)
		}
	}
	ddl := convertTablesDDL(t, "CREATE TABLE t (\"Age\" INTEGER, CHECK ([Age] >= 0));")
	if !strings.Contains(ddl, `CHECK ("Age" >= 0)`) {
		t.Fatalf("expected bracket identifier converted in CHECK:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func booleanColumnCtx(columns ...string) *TypeCoercionContext {
	cols := make(map[string][]string, len(columns))
	for _, c := range columns {
		cols[c] = []string{"0", "1"}
	}
	return &TypeCoercionContext{Samples: ColumnSamples{"t": cols}}
}

func TestConvertCheckExprBooleanLiteralRewrite(t *testing.T) {
	table := TableSchema{Name: "t", Columns: []ColumnSchema{
		{Name: "is_active", Type: "INTEGER"},
		{Name: "other", Type: "INTEGER"},
	}}
	ctx := booleanColumnCtx("is_active")

	cases := map[string]string{
		"is_active = 1":                `"is_active" = true`,
		"is_active = 0":                `"is_active" = false`,
		"is_active <> 0":               `"is_active" <> false`,
		"is_active != 1":               `"is_active" != true`,
		"is_active <= 1":               `"is_active" <= true`,
		"is_active IS 1":               `"is_active" IS true`,
		"is_active IS NOT 0":           `"is_active" IS NOT false`,
		"is_active IS DISTINCT FROM 1": `"is_active" IS DISTINCT FROM true`,
		"is_active == 1":               `"is_active" = true`,
		"0 = is_active":                `false = "is_active"`,
		"1 <> is_active":               `true <> "is_active"`,
		"1 IS is_active":               `"is_active" IS true`,
		"0 IS NOT is_active":           `"is_active" IS NOT false`,
		"1 == is_active":               `true = "is_active"`,
		"is_active IN (0, 1)":          `"is_active" IN (false, true)`,
		"is_active NOT IN (0, 1)":      `"is_active" NOT IN (false, true)`,
		"is_active BETWEEN 0 AND 1":    `"is_active" BETWEEN false AND true`,
		"is_active IN (1,0)":           `"is_active" IN (true,false)`,
		"NOT is_active IN (0,1)":       `NOT "is_active" IN (false,true)`,
		"other = 1":                    `"other" = 1`,
		"other = 0":                    `"other" = 0`,
		"is_active = 2":                `"is_active" = 2`,
		"is_active = 1 AND other = 1":  `"is_active" = true AND "other" = 1`,
		"other = 'is_active = 1'":      `"other" = 'is_active = 1'`,
	}
	for expr, want := range cases {
		if got := convertCheckExpr(expr, table, ctx); got != want {
			t.Errorf("convertCheckExpr(%q) = %q, want %q", expr, got, want)
		}
	}

	quotedTable := TableSchema{Name: "t", Columns: []ColumnSchema{{Name: "is'active", Type: "INTEGER"}}}
	if got := convertCheckExpr("[is'active] = 1", quotedTable, booleanColumnCtx("is'active")); got != `"is'active" = true` {
		t.Fatalf("quoted column: got %q", got)
	}
}

func TestConvertCheckExprBooleanLiteralRewritePreservesDecimals(t *testing.T) {
	table := TableSchema{Name: "t", Columns: []ColumnSchema{
		{Name: "is_active", Type: "INTEGER"},
	}}
	ctx := booleanColumnCtx("is_active")

	cases := map[string]string{
		"is_active = 0.0":             `"is_active" = 0.0`,
		"is_active = 1.0":             `"is_active" = 1.0`,
		"0.0 = is_active":             `0.0 = "is_active"`,
		"is_active BETWEEN 0 AND 1.0": `"is_active" BETWEEN 0 AND 1.0`,
		"is_active BETWEEN 0.0 AND 1": `"is_active" BETWEEN 0.0 AND 1`,
		"is_active IN (0, 1.5)":       `"is_active" IN (0, 1.5)`,
		"is_active = 10":              `"is_active" = 10`,
		"is_active = 1e0":             `"is_active" = 1e0`,
		"is_active = 1.":              `"is_active" = 1.`,
		"is_active = 1.e0":            `"is_active" = 1.e0`,
		"is_active = 0x1":             `"is_active" = 0x1`,
		"is_active = 1_0":             `"is_active" = 1_0`,
		"is_active = 1 + 0":           `"is_active" = 1 + 0`,
		"- 1 = is_active":             `- 1 = "is_active"`,
	}
	for expr, want := range cases {
		if got := convertCheckExpr(expr, table, ctx); got != want {
			t.Errorf("convertCheckExpr(%q) = %q, want %q", expr, got, want)
		}
	}
}

func TestConvertCheckExprBooleanRewriteRequiresContext(t *testing.T) {
	table := TableSchema{Name: "t", Columns: []ColumnSchema{{Name: "is_active", Type: "INTEGER"}}}
	got := convertCheckExpr("is_active IN (0, 1)", table, nil)
	want := `"is_active" IN (0, 1)`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestConvertCheckConstraintBooleanLiteralRewrite(t *testing.T) {
	sql := `CREATE TABLE t (
  id INTEGER PRIMARY KEY,
  is_active INTEGER NOT NULL DEFAULT 0,
  CHECK (is_active IN (0,1))
);
INSERT INTO t (id, is_active) VALUES (1, 0), (2, 1);
`
	ddl := convertTablesDDL(t, sql)
	if !strings.Contains(ddl, `"is_active" BOOLEAN`) {
		t.Fatalf("expected is_active to be coerced to BOOLEAN:\n%s", ddl)
	}
	if !strings.Contains(ddl, `CHECK ("is_active" IN (false,true))`) {
		t.Fatalf("expected CHECK literals rewritten to boolean literals:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)

	// Column-level CHECK, and a reversed-operand comparison against the boolean default.
	sql = `CREATE TABLE t2 (
  id INTEGER PRIMARY KEY,
  is_active INTEGER NOT NULL DEFAULT 0 CHECK (0 = is_active OR is_active = 1)
);
INSERT INTO t2 (id, is_active) VALUES (1, 0), (2, 1);
`
	ddl = convertTablesDDL(t, sql)
	if !strings.Contains(ddl, `"is_active" BOOLEAN`) {
		t.Fatalf("expected is_active to be coerced to BOOLEAN:\n%s", ddl)
	}
	if !strings.Contains(ddl, `CHECK (false = "is_active" OR "is_active" = true)`) {
		t.Fatalf("expected column-level CHECK literals rewritten to boolean literals:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)

	sql = `CREATE TABLE t3 (
  id INTEGER PRIMARY KEY,
  is_active INTEGER NOT NULL DEFAULT 0 CHECK (is_active IS 0 OR is_active NOT IN (0,1))
);
INSERT INTO t3 (id, is_active) VALUES (1, 0), (2, 1);
`
	ddl = convertTablesDDL(t, sql)
	if !strings.Contains(ddl, `CHECK ("is_active" IS false OR "is_active" NOT IN (false,true))`) {
		t.Fatalf("expected IS and NOT IN literals rewritten to boolean literals:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestConvertCheckConstraintBooleanLiteralRewriteIgnoresNonBooleanColumn(t *testing.T) {
	sql := `CREATE TABLE t (
  id INTEGER PRIMARY KEY,
  status INTEGER NOT NULL DEFAULT 0,
  CHECK (status IN (0, 1, 2))
);
INSERT INTO t (id, status) VALUES (1, 0), (2, 1), (3, 2);
`
	ddl := convertTablesDDL(t, sql)
	if strings.Contains(ddl, `"status" BOOLEAN`) {
		t.Fatalf("status has a non-0/1 sampled value and must not be coerced to BOOLEAN:\n%s", ddl)
	}
	if !strings.Contains(ddl, `CHECK ("status" IN (0, 1, 2))`) {
		t.Fatalf("expected CHECK literals left untouched for a non-boolean column:\n%s", ddl)
	}
	assertValidPostgresDDL(t, ddl)
}

func TestIsPartialIndexDDLNoWhitespaceBeforeWhere(t *testing.T) {
	if !isPartialIndexDDL(`CREATE UNIQUE INDEX idx ON t (a)WHERE deleted_at IS NULL;`) {
		t.Fatal("expected partial index to be detected without whitespace before WHERE")
	}
}

func TestConvertIndexDDLSkipsPartialIndexNoWhitespace(t *testing.T) {
	got := convertIndexDDL(`CREATE UNIQUE INDEX idx ON t (a)WHERE deleted_at IS NULL;`)
	if got != "" {
		t.Fatalf("convertIndexDDL = %q, want empty (partial index should be skipped, not silently made non-partial)", got)
	}
}
