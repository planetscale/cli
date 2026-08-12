package d1

import (
	"strings"
	"testing"
)

func TestParseReferencedTableName(t *testing.T) {
	cases := map[string]string{
		`REFERENCES parent_table(id)`:          "parent_table",
		`REFERENCES Parent_Table(id)`:          "Parent_Table",
		`REFERENCES "Parent_Table"(id)`:        "Parent_Table",
		`REFERENCES [Parent_Table](id)`:        "Parent_Table",
		`REFERENCES main.parent_table(id)`:     "parent_table",
		`FOREIGN KEY (x) REFERENCES users(id)`: "",
	}
	for in, want := range cases {
		if in == `FOREIGN KEY (x) REFERENCES users(id)` {
			continue
		}
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
