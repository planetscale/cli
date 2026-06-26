-- Drop all import-test tables for a clean reload (reverse dependency order).
PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS entity_links;
DROP TABLE IF EXISTS external_entities;
DROP TABLE IF EXISTS invoice_line_items;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS task_dependencies;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS project_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS organizations;

DROP TABLE IF EXISTS goose_db_version;
DROP TABLE IF EXISTS typeorm_metadata;
DROP TABLE IF EXISTS alembic_version;
DROP TABLE IF EXISTS django_migrations;
DROP TABLE IF EXISTS databasechangeloglock;
DROP TABLE IF EXISTS databasechangelog;
DROP TABLE IF EXISTS flyway_schema_history;
DROP TABLE IF EXISTS ar_internal_metadata;
DROP TABLE IF EXISTS schema_migrations;
DROP TABLE IF EXISTS sequelizemeta;
DROP TABLE IF EXISTS knex_migrations_lock;
DROP TABLE IF EXISTS knex_migrations;
DROP TABLE IF EXISTS _prisma_migrations;
DROP TABLE IF EXISTS __drizzle_migrations;

PRAGMA foreign_keys = ON;
