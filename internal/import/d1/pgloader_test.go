package d1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pgloaderOrganizationsOK = `
2026-06-29T17:07:37.780572-04:00 LOG report summary reset
             table name     errors       rows      bytes      total time
-----------------------  ---------  ---------  ---------  --------------
                  fetch          0          0                     0.000s
        fetch meta data          0          1                     0.021s
               Truncate          0          1                     0.053s
          organizations          0         28     2.7 kB          0.775s
      Total import time          ✓         28     2.7 kB          1.770s
`

const pgloaderTeamMembersZero = `
2026-06-29T17:10:50.659297-04:00 LOG report summary reset
             table name     errors       rows      bytes      total time
-----------------------  ---------  ---------  ---------  --------------
                  fetch          0          0                     0.000s
        fetch meta data          0          0                     0.012s
      Total import time          ✓          0                     0.966s
`

func mustBuildPgloaderScript(t *testing.T, sqlitePath, destURI string, cfg pgloaderScriptConfig, castTables, allTables []TableSchema, coerceCtx *TypeCoercionContext) string {
	t.Helper()
	script, err := buildPgloaderScript(sqlitePath, destURI, cfg, castTables, allTables, coerceCtx)
	if err != nil {
		t.Fatalf("buildPgloaderScript: %v", err)
	}
	return script
}

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
	script := mustBuildPgloaderScript(t, "/tmp/test.sqlite", "postgresql://u:p@host/db", pgloaderScriptConfig{
		dataOnly:       true,
		tableName:      "organizations",
		resetSequences: false,
		profile:        pgloaderProfileForTable(0),
	}, []TableSchema{table}, []TableSchema{table}, nil)

	checks := []string{
		"WITH data only, create no tables, create no indexes, truncate, disable triggers,",
		"reset no sequences,",
		"workers = 8, concurrency = 2,",
		"batch rows = 25000,",
		"batch size = 20 MB,",
		"prefetch rows = 25000",
		"INCLUDING ONLY TABLE NAMES LIKE 'organizations'",
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
		"column organizations.is_active to boolean",
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
	script := mustBuildPgloaderScript(t, "/tmp/test.sqlite", "postgresql://u:p@host/db", pgloaderScriptConfig{
		dataOnly:       true,
		tableName:      "attachments",
		resetSequences: true,
		profile:        pgloaderProfileForTable(pgloaderLargeTableRowThreshold),
	}, nil, nil, nil)

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

func TestBuildPgloaderScriptUUIDCast(t *testing.T) {
	tables, err := ParseDump(testFixture(t))
	if err != nil {
		t.Fatalf("ParseDump: %v", err)
	}
	ctx, err := BuildTypeCoercionContext(testFixture(t), tables)
	if err != nil {
		t.Fatalf("BuildTypeCoercionContext: %v", err)
	}
	var entityLinks *TableSchema
	for i := range tables {
		if tables[i].Name == "entity_links" {
			entityLinks = &tables[i]
			break
		}
	}
	if entityLinks == nil {
		t.Fatal("expected entity_links table")
	}

	script := mustBuildPgloaderScript(t, "/tmp/test.sqlite", "postgresql://u:p@host/db", pgloaderScriptConfig{
		dataOnly:  true,
		tableName: "entity_links",
		profile:   pgloaderProfileForTable(0),
	}, []TableSchema{*entityLinks}, tables, ctx)

	for _, want := range []string{
		`column "entity_links"."entity_id" to uuid using sqlite-text-to-uuid`,
		`column "entity_links"."linked_at" to timestamptz using sqlite-timestamp-to-timestamp`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script missing %q\n%s", want, script)
		}
	}

	var externalEntities *TableSchema
	for i := range tables {
		if tables[i].Name == "external_entities" {
			externalEntities = &tables[i]
			break
		}
	}
	if externalEntities == nil {
		t.Fatal("expected external_entities table")
	}
	script = mustBuildPgloaderScript(t, "/tmp/test.sqlite", "postgresql://u:p@host/db", pgloaderScriptConfig{
		dataOnly:  true,
		tableName: "external_entities",
		profile:   pgloaderProfileForTable(0),
	}, []TableSchema{*externalEntities}, tables, ctx)
	if !strings.Contains(script, `column "external_entities"."id" to uuid using sqlite-text-to-uuid`) {
		t.Fatalf("script missing external_entities UUID cast\n%s", script)
	}
}

func TestBuildPgloaderScriptQuotesCastIdentifiers(t *testing.T) {
	table := TableSchema{
		Name: "users; BEFORE LOAD DO $$evil$$",
		Columns: []ColumnSchema{{
			Name: "is_admin, type text",
			Type: "BOOLEAN",
		}},
	}

	script := mustBuildPgloaderScript(t, "/tmp/test.sqlite", "postgresql://u:p@host/db", pgloaderScriptConfig{
		dataOnly:  true,
		tableName: table.Name,
	}, []TableSchema{table}, []TableSchema{table}, nil)

	want := `column "users; BEFORE LOAD DO $$evil$$"."is_admin, type text" to boolean using sqlite-int-to-boolean`
	if !strings.Contains(script, want) {
		t.Fatalf("cast identifiers were not safely quoted:\n%s", script)
	}
}

// A name containing a double quote cannot use pgloader's double-quoted form, which has no
// escape syntax. pgloader's CAST column refs accept either quoting form and strip both to
// the same inner text, so such names fall back to the single-quoted form.
func TestBuildPgloaderScriptQuotesCastIdentifierContainingDoubleQuote(t *testing.T) {
	table := TableSchema{
		Name:    `we"ird`,
		Columns: []ColumnSchema{{Name: `is_"admin"`, Type: "BOOLEAN"}},
	}

	script := mustBuildPgloaderScript(t, "/tmp/test.sqlite", "postgresql://u:p@host/db", pgloaderScriptConfig{
		dataOnly:  true,
		tableName: table.Name,
	}, []TableSchema{table}, []TableSchema{table}, nil)

	want := `column 'we"ird'.'is_"admin"' to boolean using sqlite-int-to-boolean`
	if !strings.Contains(script, want) {
		t.Fatalf("expected single-quoted cast identifiers:\n%s", script)
	}
	if strings.Contains(script, `"we"ird"`) {
		t.Fatalf("double-quoted form cannot hold an embedded double quote:\n%s", script)
	}
}

func TestBuildPgloaderScriptRejectsUnquotableIdentifier(t *testing.T) {
	table := TableSchema{
		Name:    "O'Brien",
		Columns: []ColumnSchema{{Name: "enabled", Type: "INTEGER"}},
	}

	_, err := buildPgloaderScript("/tmp/test.sqlite", "postgresql://u:p@host/db", pgloaderScriptConfig{
		dataOnly:  true,
		tableName: table.Name,
	}, []TableSchema{table}, []TableSchema{table}, nil)
	if err == nil {
		t.Fatal("expected identifier containing a single quote to be rejected")
	}

	table = TableSchema{
		Name:    "users",
		Columns: []ColumnSchema{{Name: "O'Brien", Type: "INTEGER"}},
	}
	_, err = buildPgloaderScript("/tmp/test.sqlite", "postgresql://u:p@host/db", pgloaderScriptConfig{
		dataOnly:  true,
		tableName: table.Name,
	}, []TableSchema{table}, []TableSchema{table}, nil)
	if err == nil {
		t.Fatal("expected column name containing a single quote to be rejected")
	}
}

func TestPgloaderTableLoadFileNameCannotTraverse(t *testing.T) {
	name := pgloaderTableLoadFileName("x/../../ESCAPED")
	if filepath.Base(name) != name {
		t.Fatalf("load filename escaped work directory: %q", name)
	}
	if strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		t.Fatalf("load filename contains path syntax: %q", name)
	}
}

func TestBuildPgloaderScriptFullLoadResetsSequences(t *testing.T) {
	script := mustBuildPgloaderScript(t, "/tmp/test.sqlite", "postgresql://u:p@host/db", pgloaderScriptConfig{
		dataOnly:       true,
		resetSequences: true,
		profile:        pgloaderProfileForTable(0),
	}, nil, nil, nil)
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
	got := mustPgloaderTableNameFilter(t, "entity_links", nil)
	want := ` LIKE 'entity_links'`
	if got != want {
		t.Fatalf("pgloaderTableNameFilter() = %q, want %q", got, want)
	}
	got = mustPgloaderTableNameFilter(t, "100%done", nil)
	if got != ` LIKE '100%done'` {
		t.Fatalf("pgloaderTableNameFilter() = %q", got)
	}
	all := []string{"tbl_a", "tbl1a", "users"}
	got = mustPgloaderTableNameFilter(t, "tbl_a", all)
	if !strings.Contains(got, ` LIKE 'tbl_a'`) {
		t.Fatalf("pgloaderTableNameFilter() = %q", got)
	}
	if !strings.Contains(got, `EXCLUDING TABLE NAMES LIKE 'tbl1a'`) {
		t.Fatalf("expected false-positive exclusion, got %q", got)
	}
	if strings.Contains(got, `EXCLUDING TABLE NAMES LIKE 'users'`) {
		t.Fatalf("did not expect users excluded, got %q", got)
	}
	if _, err := pgloaderTableNameFilter("O'Brien", nil); err == nil {
		t.Fatal("expected single-quoted table names to be rejected")
	}
}

func mustPgloaderTableNameFilter(t *testing.T, name string, all []string) string {
	t.Helper()
	got, err := pgloaderTableNameFilter(name, all)
	if err != nil {
		t.Fatalf("pgloaderTableNameFilter(%q): %v", name, err)
	}
	return got
}

func TestSQLLikeMatch(t *testing.T) {
	if !sqlLikeMatch("tbl_a", "tbl1a") {
		t.Fatal("expected tbl_a pattern to match tbl1a")
	}
	if sqlLikeMatch("tbl_a", "users") {
		t.Fatal("expected tbl_a pattern not to match users")
	}
	if sqlLikeMatch("user_data", "users_data") {
		t.Fatal("expected user_data pattern not to match users_data")
	}
}

func TestPgloaderHadErrors(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name: "clean summary",
			output: `
| errors | rows | bytes | total time
|      0 |  100 |  1 kB | 1.000 s
`,
			want: false,
		},
		{
			name: "summary with errors",
			output: `
| errors | rows | bytes | total time
|      3 |   97 |  1 kB | 1.000 s
`,
			want: true,
		},
		{
			name:   "database error",
			output: "Database error 42501: must be owner of table users",
			want:   true,
		},
		{
			name:   "insufficient privilege",
			output: "INSUFFICIENT-PRIVILEGE disable triggers",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pgloaderHadErrors(tt.output); got != tt.want {
				t.Fatalf("pgloaderHadErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPgloaderFetchMetaDataTableCount(t *testing.T) {
	if got := pgloaderFetchMetaDataTableCount(pgloaderOrganizationsOK); got != 1 {
		t.Fatalf("organizations meta = %d, want 1", got)
	}
	if got := pgloaderFetchMetaDataTableCount(pgloaderTeamMembersZero); got != 0 {
		t.Fatalf("team_members meta = %d, want 0", got)
	}
	if got := pgloaderFetchMetaDataTableCount("no summary"); got != -1 {
		t.Fatalf("missing meta = %d, want -1", got)
	}
}

func TestPgloaderRowsCopied(t *testing.T) {
	rows, ok := pgloaderRowsCopied(pgloaderOrganizationsOK, "organizations")
	if !ok || rows != 28 {
		t.Fatalf("organizations rows = (%d, %v), want (28, true)", rows, ok)
	}
	rows, ok = pgloaderRowsCopied(pgloaderTeamMembersZero, "team_members")
	if ok || rows != 0 {
		t.Fatalf("team_members rows = (%d, %v), want (0, false)", rows, ok)
	}
}

func TestValidatePgloaderTableLoad(t *testing.T) {
	if err := validatePgloaderTableLoad(pgloaderOrganizationsOK, "organizations", 28); err != nil {
		t.Fatalf("expected ok load: %v", err)
	}
	if err := validatePgloaderTableLoad(pgloaderOrganizationsOK, "organizations", 0); err == nil {
		t.Fatal("expected error when staged SQLite row count does not match pgloader output")
	}
	if err := validatePgloaderTableLoad(pgloaderOrganizationsOK, "organizations", 30); err == nil {
		t.Fatal("expected error for row count mismatch")
	}
	if err := validatePgloaderTableLoad(pgloaderTeamMembersZero, "team_members", 700); err == nil {
		t.Fatal("expected error for 0-row load")
	} else if me, ok := err.(*MigrationError); !ok || me.Info.Code != ErrCodeImportFailed {
		t.Fatalf("error = %#v", err)
	}
	if err := validatePgloaderTableLoad(pgloaderTeamMembersZero, "team_members", 0); err == nil {
		t.Fatal("expected error when pgloader matched 0 source tables")
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
