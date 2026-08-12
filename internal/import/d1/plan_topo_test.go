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
		`REFERENCES "my.table"(id)`:                     "my.table",
		`REFERENCES [my.table](id)`:                     "my.table",
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

func TestBuildImportTablesSQLDefersForeignKeys(t *testing.T) {
	tables := []TableSchema{
		{
			Name: "parent_table",
			Columns: []ColumnSchema{{
				Name:          "id",
				Type:          "INTEGER",
				PrimaryKey:    true,
				AutoIncrement: true,
			}},
		},
		{
			Name: "child_table",
			Columns: []ColumnSchema{
				{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
				{Name: "parent_id", Type: "INTEGER", ForeignKey: `REFERENCES Parent_Table(id)`},
			},
			RawDDL: `CREATE TABLE child_table (id INTEGER PRIMARY KEY AUTOINCREMENT, parent_id INTEGER, FOREIGN KEY (parent_id) REFERENCES Parent_Table(id))`,
		},
	}

	sql, err := buildImportTablesSQL("", tables)
	if err != nil {
		t.Fatalf("buildImportTablesSQL: %v", err)
	}
	if strings.Contains(sql, `CREATE TABLE IF NOT EXISTS "child_table"`) &&
		strings.Contains(strings.Split(sql, "ALTER TABLE")[0], `REFERENCES "parent_table"`) {
		t.Fatalf("expected inline CREATE TABLE without foreign keys:\n%s", sql)
	}
	if !strings.Contains(sql, `ALTER TABLE "child_table" ADD CONSTRAINT "d1_fk_child_table_parent_id" FOREIGN KEY ("parent_id") REFERENCES "parent_table" ("id");`) {
		t.Fatalf("expected deferred foreign key alter:\n%s", sql)
	}
}

func TestBuildImportTablesSQLCyclicForeignKeys(t *testing.T) {
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

	sql, err := buildImportTablesSQL("", tables)
	if err != nil {
		t.Fatalf("buildImportTablesSQL: %v", err)
	}
	createSection := strings.Split(sql, "-- Foreign keys")[0]
	if strings.Contains(createSection, "REFERENCES") {
		t.Fatalf("expected CREATE TABLE section without inline foreign keys:\n%s", createSection)
	}
	if !strings.Contains(sql, `ALTER TABLE "table_a"`) || !strings.Contains(sql, `ALTER TABLE "table_b"`) {
		t.Fatalf("expected deferred foreign keys for both tables:\n%s", sql)
	}
	assertValidPostgresDDL(t, sql)
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

func TestBuildImportTablesSQLBracketAndSchemaQualifiedFK(t *testing.T) {
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
	}

	sql, err := buildImportTablesSQL("", tables)
	if err != nil {
		t.Fatalf("buildImportTablesSQL: %v", err)
	}
	for _, want := range []string{
		`ALTER TABLE "child_bracket" ADD CONSTRAINT "d1_fk_child_bracket_parent_id" FOREIGN KEY ("parent_id") REFERENCES "parent_table" ("id");`,
		`ALTER TABLE "child_schema" ADD CONSTRAINT "d1_fk_child_schema_parent_id" FOREIGN KEY ("parent_id") REFERENCES "parent_table" ("id");`,
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("missing deferred FK %q in:\n%s", want, sql)
		}
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

func TestBuildImportTablesSQLDottedQuotedTableName(t *testing.T) {
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
	sql, err := buildImportTablesSQL("", tables)
	if err != nil {
		t.Fatalf("buildImportTablesSQL: %v", err)
	}
	want := `ALTER TABLE "child" ADD CONSTRAINT "d1_fk_child_parent_id" FOREIGN KEY ("parent_id") REFERENCES "my.table" ("id");`
	if !strings.Contains(sql, want) {
		t.Fatalf("expected dotted quoted table name preserved:\n%s", sql)
	}
}

func TestConvertSchemaPartsDefersForeignKeys(t *testing.T) {
	sql := `CREATE TABLE parent_table (id INTEGER PRIMARY KEY);
CREATE TABLE child_table (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES [parent_table](id));
`
	ddl := convertTablesDDL(t, sql)
	createSection := strings.Split(ddl, "-- Foreign keys")[0]
	if strings.Contains(createSection, "REFERENCES") {
		t.Fatalf("expected CREATE TABLE section without inline foreign keys:\n%s", createSection)
	}
	if !strings.Contains(ddl, `ALTER TABLE "child_table" ADD CONSTRAINT "d1_fk_child_table_parent_id" FOREIGN KEY ("parent_id") REFERENCES "parent_table" ("id");`) {
		t.Fatalf("expected deferred foreign key alter:\n%s", ddl)
	}
}
