package d1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildPgloaderScriptDataOnlyPerTable(t *testing.T) {
	table := TableSchema{
		Name: "organizations",
		Columns: []ColumnSchema{
			{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
			{Name: "slug", Type: "TEXT", NotNull: true},
			{Name: "is_active", Type: "INTEGER", NotNull: true},
			{Name: "created_at", Type: "TEXT", NotNull: true},
		},
	}
	script := buildPgloaderScript("/tmp/test.sqlite", "postgresql://u:p@host/db", pgloaderScriptConfig{
		dataOnly:       true,
		tableName:      "organizations",
		resetSequences: false,
		profile:        pgloaderProfileForTable(0),
	}, []TableSchema{table}, []TableSchema{table})

	checks := []string{
		"WITH data only, create no tables, create no indexes, truncate, disable triggers,",
		"reset no sequences,",
		"workers = 8, concurrency = 2,",
		"batch rows = 25000,",
		"batch size = 20 MB,",
		"prefetch rows = 25000",
		"INCLUDING ONLY TABLE NAMES LIKE 'organizations' ESCAPE '\\'",
		"column organizations.is_active to boolean using sqlite-int-to-boolean",
		"column organizations.created_at to timestamptz using sqlite-timestamp-to-timestamp",
		"SET work_mem to '256MB'",
		"synchronous_commit to 'off'",
	}
	for _, want := range checks {
		if !strings.Contains(script, want) {
			t.Fatalf("script missing %q\n%s", want, script)
		}
	}
	for _, bad := range []string{
		"column organizations.id to boolean",
		"column organizations.slug to timestamptz",
		"type integer to boolean",
		"type text to timestamptz",
	} {
		if strings.Contains(script, bad) {
			t.Fatalf("script should not contain %q\n%s", bad, script)
		}
	}
}

func TestBuildPgloaderScriptLargeTableProfile(t *testing.T) {
	script := buildPgloaderScript("/tmp/test.sqlite", "postgresql://u:p@host/db", pgloaderScriptConfig{
		dataOnly:       true,
		tableName:      "attachments",
		resetSequences: true,
		profile:        pgloaderProfileForTable(pgloaderLargeTableRowThreshold),
	}, nil, nil)

	for _, want := range []string{
		"workers = 2, concurrency = 1,",
		"batch rows = 10000,",
		"prefetch rows = 5000",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script missing %q\n%s", want, script)
		}
	}
}

func TestBuildPgloaderScriptFullLoadResetsSequences(t *testing.T) {
	script := buildPgloaderScript("/tmp/test.sqlite", "postgresql://u:p@host/db", pgloaderScriptConfig{
		dataOnly:       true,
		resetSequences: true,
		profile:        pgloaderProfileForTable(0),
	}, nil, nil)
	if !strings.Contains(script, "reset sequences,") {
		t.Fatalf("expected reset sequences in final table script:\n%s", script)
	}
	if strings.Contains(script, "INCLUDING ONLY") {
		t.Fatalf("did not expect table filter for full load:\n%s", script)
	}
}

func TestPgloaderLoadTablesSkipsORMMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dump.sql")
	if err := os.WriteFile(path, []byte(`
CREATE TABLE organizations (id INTEGER PRIMARY KEY);
CREATE TABLE __drizzle_migrations (id INTEGER PRIMARY KEY);
CREATE TABLE users (id INTEGER PRIMARY KEY, org_id INTEGER);
`), 0o600); err != nil {
		t.Fatal(err)
	}

	tables, err := PgloaderLoadTables(path)
	if err != nil {
		t.Fatalf("PgloaderLoadTables: %v", err)
	}
	if len(tables) != 2 {
		t.Fatalf("tables = %v, want [organizations users]", tables)
	}
	if tables[0] != "organizations" || tables[1] != "users" {
		t.Fatalf("load order = %v", tables)
	}
}

func TestPgloaderTableNameFilterExactMatch(t *testing.T) {
	got := pgloaderTableNameFilter("entity_links")
	want := ` LIKE 'entity\_links' ESCAPE '\'`
	if got != want {
		t.Fatalf("pgloaderTableNameFilter() = %q, want %q", got, want)
	}
	got = pgloaderTableNameFilter("100%done")
	if got != ` LIKE '100\%done' ESCAPE '\'` {
		t.Fatalf("pgloaderTableNameFilter() = %q", got)
	}
	got = pgloaderTableNameFilter("tbl_a")
	if got != ` LIKE 'tbl\_a' ESCAPE '\'` {
		t.Fatalf("pgloaderTableNameFilter() = %q", got)
	}
	got = pgloaderTableNameFilter("O'Brien")
	if got != ` LIKE 'O''Brien' ESCAPE '\'` {
		t.Fatalf("pgloaderTableNameFilter() = %q", got)
	}
}

func TestConvertSchemaPartsSplitsIndexes(t *testing.T) {
	parts, count, err := ConvertSchemaParts(testFixture(t))
	if err != nil {
		t.Fatalf("ConvertSchemaParts: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected 4 tables, got %d", count)
	}
	if !strings.Contains(parts.Tables, `CREATE TABLE IF NOT EXISTS "users"`) {
		t.Fatalf("expected users table DDL")
	}
	if strings.Contains(parts.Tables, "CREATE INDEX") {
		t.Fatalf("tables section should not contain indexes")
	}
	if !strings.Contains(parts.Indexes, `CREATE INDEX IF NOT EXISTS "idx_users_email"`) {
		t.Fatalf("expected index DDL in indexes section:\n%s", parts.Indexes)
	}
}
