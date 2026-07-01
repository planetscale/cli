-- Sample D1 export fixture for import d1 tests
PRAGMA foreign_keys=OFF;

CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  email TEXT NOT NULL UNIQUE,
  active INTEGER DEFAULT 1,
  created_at TEXT NOT NULL
);

CREATE TABLE posts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  body TEXT,
  published INTEGER DEFAULT 0,
  metadata TEXT,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE external_entities (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE entity_links (
  entity_id TEXT NOT NULL,
  post_id INTEGER NOT NULL,
  linked_at TEXT NOT NULL,
  PRIMARY KEY (entity_id, post_id),
  FOREIGN KEY (entity_id) REFERENCES external_entities(id),
  FOREIGN KEY (post_id) REFERENCES posts(id)
);

CREATE TABLE __drizzle_migrations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  hash TEXT NOT NULL,
  created_at INTEGER
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

INSERT INTO users (id, email, active, created_at) VALUES
  (1, 'alice@example.com', 1, '2024-01-01T00:00:00Z'),
  (2, 'bob@example.com', 0, '2024-01-02T00:00:00Z');

INSERT INTO posts (id, user_id, title, body, published, metadata) VALUES
  (1, 1, 'Hello', 'World', 1, '{"tags":["intro"]}'),
  (2, 1, 'Draft', 'Work in progress', 0, NULL);

INSERT INTO external_entities (id, name, created_at) VALUES
  ('550e8400-e29b-41d4-a716-446655440000', 'Webhook A', '2024-01-01T00:00:00Z');

INSERT INTO entity_links (entity_id, post_id, linked_at) VALUES
  ('550e8400-e29b-41d4-a716-446655440000', 1, '2024-01-02T00:00:00Z');

INSERT INTO __drizzle_migrations (id, hash, created_at) VALUES
  (1, 'abc123', 1700000000);

CREATE INDEX idx_users_email ON users(email);
CREATE UNIQUE INDEX idx_entity_links_post ON entity_links(post_id);
