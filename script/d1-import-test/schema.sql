-- Stress schema for pscale import d1 testing (import-test D1 database).
-- Exercises: autoincrement PKs, 0/1 booleans, TEXT timestamps, JSON-in-TEXT,
-- multi-level FKs, junction tables, table-level FKs, self-referential FKs,
-- REAL columns, BLOB columns, CHECK constraints, indexes, __drizzle_migrations.
-- Note: FTS5 virtual tables are omitted — they break ParseDump on export.

PRAGMA foreign_keys = ON;

CREATE TABLE __drizzle_migrations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  hash TEXT NOT NULL,
  created_at INTEGER NOT NULL
);

CREATE TABLE _prisma_migrations (
  id TEXT PRIMARY KEY,
  checksum TEXT NOT NULL,
  finished_at TEXT,
  migration_name TEXT NOT NULL,
  logs TEXT,
  rolled_back_at TEXT,
  started_at TEXT NOT NULL,
  applied_steps_count INTEGER NOT NULL
);

CREATE TABLE knex_migrations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT,
  batch INTEGER,
  migration_time INTEGER
);

CREATE TABLE knex_migrations_lock (
  "index" INTEGER PRIMARY KEY,
  is_locked INTEGER
);

CREATE TABLE sequelizemeta (
  name TEXT PRIMARY KEY
);

CREATE TABLE schema_migrations (
  version TEXT PRIMARY KEY
);

CREATE TABLE ar_internal_metadata (
  key TEXT PRIMARY KEY,
  value TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE flyway_schema_history (
  installed_rank INTEGER PRIMARY KEY,
  version TEXT,
  description TEXT NOT NULL,
  type TEXT NOT NULL,
  script TEXT NOT NULL,
  checksum INTEGER,
  installed_by TEXT NOT NULL,
  installed_on TEXT NOT NULL DEFAULT (datetime('now')),
  execution_time INTEGER NOT NULL,
  success INTEGER NOT NULL
);

CREATE TABLE databasechangelog (
  id TEXT PRIMARY KEY,
  author TEXT NOT NULL,
  filename TEXT NOT NULL,
  dateexecuted TEXT NOT NULL,
  orderexecuted INTEGER NOT NULL,
  exectype TEXT NOT NULL,
  md5sum TEXT,
  description TEXT,
  comments TEXT,
  tag TEXT,
  liquibase TEXT,
  contexts TEXT,
  labels TEXT,
  deployment_id TEXT
);

CREATE TABLE databasechangeloglock (
  id INTEGER PRIMARY KEY,
  locked INTEGER NOT NULL,
  lockgranted TEXT,
  lockedby TEXT
);

CREATE TABLE django_migrations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  app TEXT NOT NULL,
  name TEXT NOT NULL,
  applied TEXT NOT NULL
);

CREATE TABLE alembic_version (
  version_num TEXT PRIMARY KEY
);

CREATE TABLE typeorm_metadata (
  type TEXT NOT NULL,
  "database" TEXT,
  "schema" TEXT,
  "table" TEXT,
  name TEXT,
  value TEXT
);

CREATE TABLE goose_db_version (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  version_id INTEGER NOT NULL,
  is_applied INTEGER NOT NULL,
  tstamp TEXT DEFAULT (datetime('now'))
);

CREATE TABLE organizations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  plan TEXT NOT NULL DEFAULT 'free',
  is_active INTEGER NOT NULL DEFAULT 1,
  settings TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT
);

CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER NOT NULL,
  email TEXT NOT NULL,
  display_name TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'member',
  is_active INTEGER NOT NULL DEFAULT 1,
  is_admin INTEGER NOT NULL DEFAULT 0,
  profile_json TEXT,
  last_login_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (org_id) REFERENCES organizations(id)
);

CREATE TABLE teams (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  is_archived INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  FOREIGN KEY (org_id) REFERENCES organizations(id)
);

CREATE TABLE team_members (
  team_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  role TEXT NOT NULL DEFAULT 'member',
  joined_at TEXT NOT NULL,
  PRIMARY KEY (team_id, user_id),
  FOREIGN KEY (team_id) REFERENCES teams(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE projects (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER NOT NULL,
  owner_user_id INTEGER NOT NULL,
  team_id INTEGER,
  slug TEXT NOT NULL,
  name TEXT NOT NULL,
  description TEXT,
  is_public INTEGER NOT NULL DEFAULT 0,
  is_archived INTEGER NOT NULL DEFAULT 0,
  metadata TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (org_id) REFERENCES organizations(id),
  FOREIGN KEY (owner_user_id) REFERENCES users(id),
  FOREIGN KEY (team_id) REFERENCES teams(id),
  UNIQUE (org_id, slug)
);

CREATE TABLE tags (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER NOT NULL,
  label TEXT NOT NULL,
  color TEXT NOT NULL DEFAULT '#6366f1',
  FOREIGN KEY (org_id) REFERENCES organizations(id),
  UNIQUE (org_id, label)
);

CREATE TABLE project_tags (
  project_id INTEGER NOT NULL,
  tag_id INTEGER NOT NULL,
  PRIMARY KEY (project_id, tag_id),
  FOREIGN KEY (project_id) REFERENCES projects(id),
  FOREIGN KEY (tag_id) REFERENCES tags(id)
);

CREATE TABLE tasks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id INTEGER NOT NULL,
  assignee_user_id INTEGER,
  parent_task_id INTEGER,
  title TEXT NOT NULL,
  body TEXT,
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'done', 'cancelled')),
  priority INTEGER NOT NULL DEFAULT 2 CHECK (priority BETWEEN 1 AND 5),
  estimate_hours REAL,
  is_blocked INTEGER NOT NULL DEFAULT 0,
  labels_json TEXT,
  due_at TEXT,
  completed_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (project_id) REFERENCES projects(id),
  FOREIGN KEY (assignee_user_id) REFERENCES users(id),
  FOREIGN KEY (parent_task_id) REFERENCES tasks(id)
);

CREATE TABLE task_dependencies (
  task_id INTEGER NOT NULL,
  depends_on_task_id INTEGER NOT NULL,
  PRIMARY KEY (task_id, depends_on_task_id),
  FOREIGN KEY (task_id) REFERENCES tasks(id),
  FOREIGN KEY (depends_on_task_id) REFERENCES tasks(id)
);

CREATE TABLE comments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id INTEGER NOT NULL,
  author_user_id INTEGER NOT NULL,
  parent_comment_id INTEGER,
  body TEXT NOT NULL,
  is_edited INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (task_id) REFERENCES tasks(id),
  FOREIGN KEY (author_user_id) REFERENCES users(id),
  FOREIGN KEY (parent_comment_id) REFERENCES comments(id)
);

CREATE TABLE attachments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  comment_id INTEGER NOT NULL,
  filename TEXT NOT NULL,
  content_type TEXT NOT NULL,
  byte_size INTEGER NOT NULL,
  checksum TEXT NOT NULL,
  payload BLOB,
  uploaded_at TEXT NOT NULL,
  FOREIGN KEY (comment_id) REFERENCES comments(id)
);

CREATE TABLE audit_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER NOT NULL,
  actor_user_id INTEGER,
  action TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  entity_id INTEGER NOT NULL,
  payload TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (org_id) REFERENCES organizations(id),
  FOREIGN KEY (actor_user_id) REFERENCES users(id)
);

CREATE TABLE api_keys (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  key_prefix TEXT NOT NULL,
  is_active INTEGER NOT NULL DEFAULT 1,
  scopes_json TEXT NOT NULL,
  expires_at TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  ip_address TEXT,
  user_agent TEXT,
  last_seen_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE notifications (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  kind TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT,
  payload TEXT,
  is_read INTEGER NOT NULL DEFAULT 0,
  read_at TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE invoices (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  org_id INTEGER NOT NULL,
  invoice_number TEXT NOT NULL UNIQUE,
  subtotal REAL NOT NULL,
  tax REAL NOT NULL DEFAULT 0,
  total REAL NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  issued_at TEXT,
  due_at TEXT,
  paid_at TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (org_id) REFERENCES organizations(id)
);

CREATE TABLE invoice_line_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  invoice_id INTEGER NOT NULL,
  task_id INTEGER,
  description TEXT NOT NULL,
  quantity REAL NOT NULL DEFAULT 1,
  unit_price REAL NOT NULL,
  amount REAL NOT NULL,
  FOREIGN KEY (invoice_id) REFERENCES invoices(id),
  FOREIGN KEY (task_id) REFERENCES tasks(id)
);

CREATE TABLE external_entities (
  id TEXT PRIMARY KEY,
  org_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT 'webhook',
  metadata TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (org_id) REFERENCES organizations(id)
);

CREATE TABLE entity_links (
  entity_id TEXT NOT NULL,
  task_id INTEGER NOT NULL,
  linked_at TEXT NOT NULL,
  PRIMARY KEY (entity_id, task_id),
  FOREIGN KEY (entity_id) REFERENCES external_entities(id),
  FOREIGN KEY (task_id) REFERENCES tasks(id)
);

CREATE INDEX idx_users_org_id ON users(org_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_teams_org_id ON teams(org_id);
CREATE INDEX idx_projects_org_id ON projects(org_id);
CREATE INDEX idx_projects_owner ON projects(owner_user_id);
CREATE INDEX idx_tasks_project_id ON tasks(project_id);
CREATE INDEX idx_tasks_assignee ON tasks(assignee_user_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_comments_task_id ON comments(task_id);
CREATE INDEX idx_audit_log_org_created ON audit_log(org_id, created_at);
CREATE INDEX idx_notifications_user_unread ON notifications(user_id, is_read);
CREATE INDEX idx_external_entities_org_id ON external_entities(org_id);
CREATE UNIQUE INDEX idx_external_entities_org_name ON external_entities(org_id, name);
