package d1

import (
	"strings"
)

type ormMetadataRule struct {
	code        string
	orm         string
	remediation string
	match       func(table string) bool
}

var ormMetadataRules = []ormMetadataRule{
	{
		code: "DRIZZLE_MIGRATIONS",
		orm:  "Drizzle",
		remediation: "After import, baseline Drizzle on Postgres (e.g. drizzle-kit push or a fresh migrations folder); " +
			"do not rely on SQLite __drizzle_migrations history",
		match: func(table string) bool {
			return strings.HasPrefix(strings.ToLower(table), "__drizzle")
		},
	},
	{
		code: "PRISMA_MIGRATIONS",
		orm:  "Prisma",
		remediation: "After import, baseline Prisma on Postgres (e.g. prisma db pull then prisma migrate resolve / new initial migration); " +
			"do not import _prisma_migrations from SQLite",
		match: matchTableName("_prisma_migrations"),
	},
	{
		code:        "KNEX_MIGRATIONS",
		orm:         "Knex",
		remediation: "After import, re-baseline Knex migration history on Postgres; knex_migrations from SQLite is not valid on Postgres",
		match:       matchAnyTableName("knex_migrations", "knex_migrations_lock"),
	},
	{
		code:        "SEQUELIZE_META",
		orm:         "Sequelize",
		remediation: "After import, re-baseline Sequelize migration history on Postgres; SequelizeMeta from SQLite is not valid on Postgres",
		match:       matchTableName("sequelizemeta"),
	},
	{
		code:        "RAILS_MIGRATIONS",
		orm:         "Rails ActiveRecord",
		remediation: "After import, re-baseline Rails schema_migrations on Postgres; SQLite migration versions do not transfer cleanly",
		match:       matchAnyTableName("schema_migrations", "ar_internal_metadata"),
	},
	{
		code:        "FLYWAY_MIGRATIONS",
		orm:         "Flyway",
		remediation: "After import, baseline Flyway on Postgres; flyway_schema_history from SQLite must not be reused",
		match:       matchTableName("flyway_schema_history"),
	},
	{
		code:        "LIQUIBASE_MIGRATIONS",
		orm:         "Liquibase",
		remediation: "After import, baseline Liquibase on Postgres; databasechangelog tables from SQLite must not be reused",
		match:       matchAnyTableName("databasechangelog", "databasechangeloglock"),
	},
	{
		code:        "DJANGO_MIGRATIONS",
		orm:         "Django",
		remediation: "After import, run django migrate --fake-initial or otherwise baseline django_migrations on Postgres",
		match:       matchTableName("django_migrations"),
	},
	{
		code:        "ALEMBIC_VERSION",
		orm:         "Alembic",
		remediation: "After import, stamp Alembic to the correct Postgres revision; alembic_version from SQLite is not portable",
		match:       matchTableName("alembic_version"),
	},
	{
		code:        "TYPEORM_METADATA",
		orm:         "TypeORM",
		remediation: "After import, baseline TypeORM migrations on Postgres; typeorm_metadata from SQLite is not valid on Postgres",
		match:       matchTableName("typeorm_metadata"),
	},
	{
		code:        "GOOSE_MIGRATIONS",
		orm:         "Goose",
		remediation: "After import, re-baseline Goose version table on Postgres; goose_db_version from SQLite is not portable",
		match:       matchTableName("goose_db_version"),
	},
}

func matchTableName(name string) func(string) bool {
	lower := strings.ToLower(name)
	return func(table string) bool {
		return strings.ToLower(table) == lower
	}
}

func matchAnyTableName(names ...string) func(string) bool {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[strings.ToLower(name)] = struct{}{}
	}
	return func(table string) bool {
		_, ok := set[strings.ToLower(table)]
		return ok
	}
}

// IsORMMetadataTable reports whether a table holds ORM/framework migration bookkeeping
// that should not be imported into Postgres.
func IsORMMetadataTable(name string) bool {
	return ORMMetadataRule(name) != nil
}

// ORMMetadataRule returns the matching ORM metadata rule, if any.
func ORMMetadataRule(name string) *ormMetadataRule {
	for i := range ormMetadataRules {
		if ormMetadataRules[i].match(name) {
			return &ormMetadataRules[i]
		}
	}
	return nil
}

func lintORMMetadata(table TableSchema) []Issue {
	rule := ORMMetadataRule(table.Name)
	if rule == nil {
		return nil
	}
	issues := []Issue{{
		Code:        rule.code,
		Severity:    SeverityInfo,
		Table:       table.Name,
		Message:     rule.orm + " migration metadata table detected",
		Remediation: rule.remediation,
	}}
	if strings.EqualFold(table.Name, "schema_migrations") && !looksLikeRailsSchemaMigrations(table) {
		issues = append(issues, Issue{
			Code:        "SCHEMA_MIGRATIONS_NAME_COLLISION",
			Severity:    SeverityWarning,
			Table:       table.Name,
			Message:     "table name matches Rails schema_migrations but column layout does not",
			Remediation: "If this is application data, rename the table before import; ORM metadata skip will exclude it from Postgres",
		})
	}
	return issues
}

func looksLikeRailsSchemaMigrations(table TableSchema) bool {
	if len(table.Columns) != 1 {
		return false
	}
	col := table.Columns[0]
	name := strings.ToLower(col.Name)
	if name != "version" {
		return false
	}
	t := strings.ToUpper(col.Type)
	return strings.Contains(t, "CHAR") || strings.Contains(t, "TEXT") || t == ""
}
