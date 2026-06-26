package d1

import "testing"

func TestIsORMMetadataTable(t *testing.T) {
	tests := []struct {
		table string
		want  bool
		code  string
	}{
		{"__drizzle_migrations", true, "DRIZZLE_MIGRATIONS"},
		{"__drizzle_migrations_journal", true, "DRIZZLE_MIGRATIONS"},
		{"_prisma_migrations", true, "PRISMA_MIGRATIONS"},
		{"knex_migrations", true, "KNEX_MIGRATIONS"},
		{"knex_migrations_lock", true, "KNEX_MIGRATIONS"},
		{"SequelizeMeta", true, "SEQUELIZE_META"},
		{"schema_migrations", true, "RAILS_MIGRATIONS"},
		{"ar_internal_metadata", true, "RAILS_MIGRATIONS"},
		{"flyway_schema_history", true, "FLYWAY_MIGRATIONS"},
		{"databasechangelog", true, "LIQUIBASE_MIGRATIONS"},
		{"django_migrations", true, "DJANGO_MIGRATIONS"},
		{"alembic_version", true, "ALEMBIC_VERSION"},
		{"typeorm_metadata", true, "TYPEORM_METADATA"},
		{"goose_db_version", true, "GOOSE_MIGRATIONS"},
		{"users", false, ""},
		{"migrations", false, ""},
		{"organizations", false, ""},
	}

	for _, tc := range tests {
		got := IsORMMetadataTable(tc.table)
		if got != tc.want {
			t.Fatalf("IsORMMetadataTable(%q) = %v, want %v", tc.table, got, tc.want)
		}
		if tc.want {
			rule := ORMMetadataRule(tc.table)
			if rule == nil || rule.code != tc.code {
				t.Fatalf("ORMMetadataRule(%q) = %v, want code %q", tc.table, rule, tc.code)
			}
		}
	}
}

func TestLintORMMetadataTables(t *testing.T) {
	result, err := Lint(testFixture(t))
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}

	found := map[string]bool{}
	for _, issue := range result.Issues {
		if issue.Code == "DRIZZLE_MIGRATIONS" || issue.Code == "PRISMA_MIGRATIONS" {
			found[issue.Code] = true
		}
	}
	if !found["DRIZZLE_MIGRATIONS"] {
		t.Fatal("expected DRIZZLE_MIGRATIONS lint issue")
	}
	if !found["PRISMA_MIGRATIONS"] {
		t.Fatal("expected PRISMA_MIGRATIONS lint issue")
	}
}
