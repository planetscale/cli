package d1

import (
	"strings"
	"testing"
)

func TestConflictingImportTables(t *testing.T) {
	existing := map[string]struct{}{
		"organizations": {},
		"posts":         {},
		"other_app":     {},
	}
	conflicts := conflictingImportTables([]string{"organizations", "users", "posts"}, existing)
	if len(conflicts) != 2 || conflicts[0] != "organizations" || conflicts[1] != "posts" {
		t.Fatalf("conflicts = %v", conflicts)
	}
}

func TestErrExistingImportTables(t *testing.T) {
	err := errExistingImportTables([]string{"users", "posts"})
	requireMigrationErr(t, err, ErrCodeDestinationConflict)
	me, _ := migrationErr(err)
	if !strings.Contains(me.Info.Message, "users, posts") {
		t.Fatalf("message = %q", me.Info.Message)
	}
}

func TestBuildImportTablesSQLCreatesAllImportTables(t *testing.T) {
	tables := []TableSchema{
		{
			Name: "organizations",
			Columns: []ColumnSchema{
				{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
			},
		},
		{
			Name: "users",
			Columns: []ColumnSchema{
				{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
			},
		},
	}

	sql, err := buildImportTablesSQL("", tables)
	if err != nil {
		t.Fatalf("buildImportTablesSQL: %v", err)
	}
	if !strings.Contains(sql, `CREATE TABLE IF NOT EXISTS "organizations"`) {
		t.Fatalf("expected organizations table DDL:\n%s", sql)
	}
	if !strings.Contains(sql, `CREATE TABLE IF NOT EXISTS "users"`) {
		t.Fatalf("expected users table DDL:\n%s", sql)
	}
}

func TestImportTableNamesSkipsORMMetadata(t *testing.T) {
	names := importTableNames([]TableSchema{
		{Name: "users"},
		{Name: "__drizzle_migrations"},
		{Name: "posts"},
	})
	if len(names) != 2 || names[0] != "users" || names[1] != "posts" {
		t.Fatalf("names = %v", names)
	}
}
