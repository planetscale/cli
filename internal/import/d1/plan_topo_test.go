package d1

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseReferencedTableName(t *testing.T) {
	cases := map[string]string{
		`REFERENCES parent_table(id)`:                   "parent_table",
		`REFERENCES Parent_Table(id)`:                   "Parent_Table",
		`REFERENCES "Parent_Table"(id)`:                 "Parent_Table",
		`REFERENCES [Parent_Table](id)`:                 "Parent_Table",
		`REFERENCES main.parent_table(id)`:              "parent_table",
		`REFERENCES "main"."parent_table"(id)`:          "parent_table",
		`REFERENCES [main].[parent_table](id)`:          "parent_table",
		`REFERENCES main."Parent"(id)`:                  "Parent",
		`REFERENCES main.[Parent](id)`:                  "Parent",
		`REFERENCES "my.table"(id)`:                     "my.table",
		`REFERENCES [my.table](id)`:                     "my.table",
		`REFERENCES Parent`:                             "Parent",
		`FOREIGN KEY (x) REFERENCES users(id)`:          "users",
		`FOREIGN KEY (x) REFERENCES [Parent_Table](id)`: "Parent_Table",
		`FOREIGN KEY (x) REFERENCES main.parent(id)`:    "parent",
	}
	for in, want := range cases {
		got := parseReferencedTableName(in)
		if got != want {
			t.Errorf("parseReferencedTableName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTopologicalLoadOrderCaseInsensitiveFK(t *testing.T) {
	tables := []TableSchema{
		{Name: "parent_table"},
		{
			Name: "child_table",
			Columns: []ColumnSchema{{
				Name:       "parent_id",
				Type:       "INTEGER",
				ForeignKey: `REFERENCES Parent_Table(id)`,
			}},
			RawDDL: `CREATE TABLE child_table (id INTEGER PRIMARY KEY, parent_id INTEGER, FOREIGN KEY (parent_id) REFERENCES Parent_Table(id))`,
		},
	}
	order := topologicalLoadOrder(tables)
	if len(order) != 2 {
		t.Fatalf("order=%v", order)
	}
	if order[0] != "parent_table" || order[1] != "child_table" {
		t.Fatalf("expected parent_table before child_table, got %v", order)
	}
}

func TestBuildImportTablesSQLOmitsForeignKeys(t *testing.T) {
	tables := []TableSchema{
		{
			Name: "parent_table",
			Columns: []ColumnSchema{{
				Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true,
			}},
		},
		{
			Name: "child_table",
			Columns: []ColumnSchema{
				{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
				{Name: "parent_id", Type: "INTEGER", ForeignKey: `REFERENCES Parent_Table(id)`},
			},
		},
	}

	sql, err := buildImportTablesSQL("", tables)
	if err != nil {
		t.Fatalf("buildImportTablesSQL: %v", err)
	}
	if strings.Contains(sql, "REFERENCES") || strings.Contains(sql, "ALTER TABLE") {
		t.Fatalf("import schema SQL must omit foreign keys (applied post-load):\n%s", sql)
	}
}

func TestDeferredForeignKeysSQL(t *testing.T) {
	tables := []TableSchema{
		{
			Name: "parent_table",
			Columns: []ColumnSchema{{
				Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true,
			}},
		},
		{
			Name: "child_table",
			Columns: []ColumnSchema{
				{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
				{Name: "parent_id", Type: "INTEGER", ForeignKey: `REFERENCES Parent_Table(id)`},
			},
		},
	}

	fkSQL := buildDeferredForeignKeysSQL(tables, nil)
	wantDrop := `ALTER TABLE "child_table" DROP CONSTRAINT IF EXISTS "d1_fk_child_table_parent_id";`
	wantAdd := `ALTER TABLE "child_table" ADD CONSTRAINT "d1_fk_child_table_parent_id" FOREIGN KEY ("parent_id") REFERENCES "parent_table" ("id");`
	if !strings.Contains(fkSQL, wantDrop) || !strings.Contains(fkSQL, wantAdd) {
		t.Fatalf("expected idempotent deferred FK alters:\n%s", fkSQL)
	}
}

func TestDeferredForeignKeysSQLCyclic(t *testing.T) {
	tables := []TableSchema{
		{
			Name: "table_a",
			Columns: []ColumnSchema{
				{Name: "id", Type: "INTEGER", PrimaryKey: true},
				{Name: "b_id", Type: "INTEGER", ForeignKey: `REFERENCES table_b(id)`},
			},
		},
		{
			Name: "table_b",
			Columns: []ColumnSchema{
				{Name: "id", Type: "INTEGER", PrimaryKey: true},
				{Name: "a_id", Type: "INTEGER", ForeignKey: `REFERENCES table_a(id)`},
			},
		},
	}

	createSQL, err := buildImportTablesSQL("", tables)
	if err != nil {
		t.Fatalf("buildImportTablesSQL: %v", err)
	}
	if strings.Contains(createSQL, "REFERENCES") {
		t.Fatalf("expected CREATE TABLE SQL without foreign keys:\n%s", createSQL)
	}
	fkSQL := buildDeferredForeignKeysSQL(tables, nil)
	if !strings.Contains(fkSQL, `ALTER TABLE "table_a"`) || !strings.Contains(fkSQL, `ALTER TABLE "table_b"`) {
		t.Fatalf("expected deferred foreign keys for both tables:\n%s", fkSQL)
	}
	assertValidPostgresDDL(t, createSQL+"\n"+fkSQL)
}

func TestLintUnresolvedForeignKey(t *testing.T) {
	table := TableSchema{
		Name: "child_table",
		Columns: []ColumnSchema{{
			Name:       "parent_id",
			Type:       "INTEGER",
			ForeignKey: `REFERENCES missing_parent(id)`,
		}},
	}
	issues := lintForeignKeyReferences(table, nil)
	if len(issues) != 1 || issues[0].Code != "UNRESOLVED_FOREIGN_KEY" {
		t.Fatalf("issues = %#v, want unresolved foreign key error", issues)
	}
}

func TestLintForeignKeyIgnoresCommentedRawDDL(t *testing.T) {
	parent := TableSchema{
		Name:    "users",
		Columns: []ColumnSchema{{Name: "id", Type: "INTEGER", PrimaryKey: true}},
	}
	child := TableSchema{
		Name: "posts",
		Columns: []ColumnSchema{
			{Name: "id", Type: "INTEGER", PrimaryKey: true},
			{Name: "user_id", Type: "INTEGER"},
		},
		RawDDL: `CREATE TABLE posts (
  id INTEGER PRIMARY KEY,
  user_id INTEGER
  -- FOREIGN KEY (user_id) REFERENCES missing_table(id)
);`,
	}
	issues := lintForeignKeyReferences(child, []TableSchema{parent, child})
	if len(issues) != 0 {
		t.Fatalf("commented RawDDL FK must not lint: %#v", issues)
	}
}

func TestColumnFKTargetCompositePositional(t *testing.T) {
	parent := TableSchema{
		Name: "parent",
		Columns: []ColumnSchema{
			{Name: "a", Type: "INTEGER"},
			{Name: "b", Type: "TEXT"},
		},
		Constraints: []string{`PRIMARY KEY (a, b)`},
	}
	child := TableSchema{
		Name: "child",
		Columns: []ColumnSchema{
			{Name: "pa", Type: "INTEGER"},
			{Name: "pb", Type: "TEXT"},
		},
		// No referenced column list: defaults to parent PK, positionally.
		Constraints: []string{`FOREIGN KEY (pa, pb) REFERENCES parent`},
	}
	all := []TableSchema{parent, child}

	if _, col := columnFKTarget(child.Columns[0], child, all); col != "a" {
		t.Fatalf("child.pa -> %q, want a", col)
	}
	if _, col := columnFKTarget(child.Columns[1], child, all); col != "b" {
		t.Fatalf("child.pb -> %q, want b", col)
	}

	// Explicit referenced column list must also map positionally.
	child.Constraints = []string{`FOREIGN KEY (pa, pb) REFERENCES parent(a, b)`}
	if _, col := columnFKTarget(child.Columns[1], child, all); col != "b" {
		t.Fatalf("explicit child.pb -> %q, want b", col)
	}
}

func TestParseReferencesTargetDefaultsToPrimaryKey(t *testing.T) {
	parent := TableSchema{
		Name:    "users",
		Columns: []ColumnSchema{{Name: "id", Type: "INTEGER", PrimaryKey: true}},
	}
	table, col := parseReferencesTarget(`REFERENCES users`, []TableSchema{parent})
	if table != "users" || col != "id" {
		t.Fatalf("got %s.%s, want users.id", table, col)
	}
	table, col = parseReferencesTarget(`REFERENCES users`, nil)
	if table != "users" || col != "" {
		t.Fatalf("without table set got %s.%s, want users.\"\"", table, col)
	}
}

func TestDeferredForeignKeysBracketAndSchemaQualified(t *testing.T) {
	tables := []TableSchema{
		{
			Name: "parent_table",
			Columns: []ColumnSchema{{
				Name: "id", Type: "INTEGER", PrimaryKey: true,
			}},
		},
		{
			Name: "child_bracket",
			Columns: []ColumnSchema{
				{Name: "id", Type: "INTEGER", PrimaryKey: true},
				{Name: "parent_id", Type: "INTEGER", ForeignKey: `REFERENCES [parent_table](id)`},
			},
		},
		{
			Name: "child_schema",
			Columns: []ColumnSchema{
				{Name: "id", Type: "INTEGER", PrimaryKey: true},
				{Name: "parent_id", Type: "INTEGER", ForeignKey: `REFERENCES main.parent_table(id)`},
			},
		},
		{
			Name: "child_mixed",
			Columns: []ColumnSchema{
				{Name: "id", Type: "INTEGER", PrimaryKey: true},
				{Name: "parent_id", Type: "INTEGER", ForeignKey: `REFERENCES main."parent_table"(id)`},
			},
		},
	}

	fkSQL := buildDeferredForeignKeysSQL(tables, nil)
	for _, want := range []string{
		`FOREIGN KEY ("parent_id") REFERENCES "parent_table" ("id");`,
	} {
		count := strings.Count(fkSQL, want)
		if count < 3 {
			t.Fatalf("expected at least 3 deferred FKs matching %q (got %d):\n%s", want, count, fkSQL)
		}
	}
	if strings.Contains(fkSQL, `"""parent_table"""`) || strings.Contains(fkSQL, `"[parent_table]"`) {
		t.Fatalf("quoted chars leaked into table identifier:\n%s", fkSQL)
	}
}

func TestConvertReferencesClauseNoColumnListUsesPrimaryKey(t *testing.T) {
	parent := TableSchema{
		Name: "Parent",
		Columns: []ColumnSchema{{
			Name: "Id", Type: "INTEGER", PrimaryKey: true,
		}},
	}
	got := convertReferencesClause(`REFERENCES Parent`, []TableSchema{parent})
	want := `REFERENCES "Parent" ("Id")`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCollectDeferredForeignKeysDedupesColumnAndTableFK(t *testing.T) {
	table := TableSchema{
		Name: "child_table",
		Columns: []ColumnSchema{
			{Name: "id", Type: "INTEGER", PrimaryKey: true},
			{Name: "parent_id", Type: "INTEGER", ForeignKey: `REFERENCES parent_table(id)`},
		},
		Constraints: []string{
			`FOREIGN KEY (parent_id) REFERENCES parent_table(id)`,
		},
	}
	parent := TableSchema{
		Name: "parent_table",
		Columns: []ColumnSchema{{
			Name: "id", Type: "INTEGER", PrimaryKey: true,
		}},
	}
	alters := collectDeferredForeignKeyAlters(table, []TableSchema{parent, table}, nil)
	if len(alters) != 1 {
		t.Fatalf("expected 1 deferred FK after dedupe, got %d:\n%v", len(alters), alters)
	}
	if !strings.Contains(alters[0], "DROP CONSTRAINT IF EXISTS") {
		t.Fatalf("expected replay-safe DROP before ADD:\n%s", alters[0])
	}
}

func TestFitPostgresIdentifierTruncatesConstraintNames(t *testing.T) {
	longTable := strings.Repeat("t", 40)
	longCol := strings.Repeat("c", 40)
	name := fitPostgresIdentifier(fmt.Sprintf("d1_fk_%s_%s", longTable, longCol))
	if len(name) > postgresMaxIdentifierBytes {
		t.Fatalf("fitPostgresIdentifier length %d exceeds %d: %q", len(name), postgresMaxIdentifierBytes, name)
	}
	if fitPostgresIdentifier(name) != name {
		t.Fatalf("fitPostgresIdentifier should be idempotent for fitted names")
	}
}

func TestDeferredForeignKeysDottedQuotedTableName(t *testing.T) {
	tables := []TableSchema{
		{
			Name: "my.table",
			Columns: []ColumnSchema{{
				Name: "id", Type: "INTEGER", PrimaryKey: true,
			}},
		},
		{
			Name: "child",
			Columns: []ColumnSchema{
				{Name: "id", Type: "INTEGER", PrimaryKey: true},
				{Name: "parent_id", Type: "INTEGER", ForeignKey: `REFERENCES "my.table"(id)`},
			},
		},
	}
	fkSQL := buildDeferredForeignKeysSQL(tables, nil)
	want := `FOREIGN KEY ("parent_id") REFERENCES "my.table" ("id");`
	if !strings.Contains(fkSQL, want) {
		t.Fatalf("expected dotted quoted table name preserved:\n%s", fkSQL)
	}
}

func TestConvertSchemaPartsDefersForeignKeys(t *testing.T) {
	sql := `CREATE TABLE parent_table (id INTEGER PRIMARY KEY);
CREATE TABLE child_table (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES [parent_table](id));
`
	parts, _, err := ConvertSchemaParts(writeDump(t, sql))
	if err != nil {
		t.Fatalf("ConvertSchemaParts: %v", err)
	}
	if strings.Contains(parts.Tables, "REFERENCES") || strings.Contains(parts.Tables, "ALTER TABLE") {
		t.Fatalf("expected tables section without foreign keys:\n%s", parts.Tables)
	}
	if !strings.Contains(parts.ForeignKeys, `FOREIGN KEY ("parent_id") REFERENCES "parent_table" ("id");`) {
		t.Fatalf("expected deferred foreign keys section:\n%s", parts.ForeignKeys)
	}
}
