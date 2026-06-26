# D1 `import-test` stress database

Synthetic schema + seed data for exercising `pscale import d1`.

## CLI import test (recommended)

Use **`run-cli-import.sh`** to exercise the full `pscale import d1` pipeline against an export file that is already on disk. This is the script to use for import timing and verification.

**Smoke (~2 MB export):**

```bash
wrangler d1 export import-test --remote --output /tmp/import-test-export.sql
IMPORT_PROFILE=smoke SKIP_DB_CREATE=true ./script/d1-import-test/run-cli-import.sh
```

**9 GB export (fresh database each run — default):**

```bash
IMPORT_PROFILE=9gb ./script/d1-import-test/run-local-import.sh
```

**Reuse same DB name but wipe it first:**

```bash
FRESH_DB=recreate PSCALE_DB=cf-d1-import-9gb ./script/d1-import-test/run-local-import.sh
```

**Provision DB + CLI import:**

```bash
IMPORT_PROFILE=smoke ./script/d1-import-test/run-local-import.sh
```

| Variable | Default | Purpose |
|----------|---------|---------|
| `IMPORT_PROFILE` | `smoke` | `smoke` or `9gb` (sets export path + DB name) |
| `D1_EXPORT` | profile default | Path to wrangler SQL export |
| `PSCALE_DB` | profile default | Target PlanetScale Postgres database |
| `PSCALE_ORG` | `bb` | Organization |
| `IMPORT_METHOD` | `pgloader` | `pgloader` or `psql` |
| `IMPORT_RUN_DIR` | `/tmp/d1-cli-import-…` | JSON artifacts (preview/start/verify) |
| `FRESH_DB` | `new` | `new` (timestamped DB), `recreate` (delete+create same name), or `reuse` (reset public schema in place) |
| `SKIP_DB_CREATE` | — | Deprecated; use `FRESH_DB` instead |

The start JSON includes `data.timings` (total, schema, pgloader per-table) when built from current `pscale-test`.

## D1 dataset prep (separate from CLI import)

Load remote D1 and/or time wrangler export — **not** the PlanetScale import:

```bash
./script/d1-import-test/load-bulk.sh
./script/d1-import-test/time-export.sh

# Optional: D1 prep then CLI import in one go
RUN_CLI_IMPORT=true SKIP_DB_CREATE=true ./script/d1-import-test/run-9gb-benchmark.sh
```

## Load (remote)

**Smoke / small seeds** — one `wrangler d1 execute` per batch file:

```bash
./script/d1-import-test/load.sh
```

**Multi‑MB / ~9 GB seeds** — merge batches into ~50 MB chunks first (Option B):

```bash
./script/d1-import-test/load-bulk.sh
# or fresh 9 GB target:
D1_FRESH=true SEED_TARGET_GB=9 ./script/d1-import-test/load-bulk.sh
```

`load-bulk.sh` runs `generate_seed.py`, then `merge_seed_chunks.py` (concatenates complete SQL statements into `seed/chunks/chunk_*.sql`), then one `wrangler d1 execute --file` per chunk. D1 is single-threaded per database, so chunks upload sequentially, but ~180 wrangler calls for 9 GB beats ~188k per-row files.

| Variable | Default | Purpose |
|----------|---------|---------|
| `CHUNK_TARGET_MB` | `50` | Target size per merged chunk file |
| `SEED_MAX_STATEMENT_BYTES` | `100000` | D1 max per SQL statement (do not exceed) |

Set `D1_DATABASE=import-test` and `D1_REMOTE=true` (default).

### Volume

By default the seed generator targets **~9 GB** of data (under D1's 10 GB cap), mostly via 1 MiB attachment blobs.

Quick local smoke test (~2 MB):

```bash
SEED_TARGET_GB=0.002 ./script/d1-import-test/load.sh
```

Moderate import e2e (~30 MB, D1-safe blob batches):

```bash
SEED_TARGET_GB=0.03 ./script/d1-import-test/load.sh
```

Blob INSERT batches are capped at ~100 KB per statement (`SEED_MAX_STATEMENT_BYTES`) to avoid D1 `SQLITE_TOOBIG`.

Tune with:

| Variable | Default | Purpose |
|----------|---------|---------|
| `SEED_TARGET_GB` | `9` | Approximate total DB size |
| `SEED_PAYLOAD_BYTES` | `1048576` | Attachment blob size (bytes) |
| `SEED_RESERVED_BYTES` | `536870912` | Headroom for non-blob rows |
| `SEED_TASKS_PER_PROJECT` | `16` | Tasks per project |
| `SEED_PROJECTS_PER_ORG` | `30` | Projects per org |

A full ~9 GB remote load: use `load-bulk.sh` (~200 chunk uploads). Avoid `load.sh` at that scale (one wrangler call per batch file).

## Reload from scratch

```bash
wrangler d1 execute import-test --remote --file=script/d1-import-test/reset.sql
./script/d1-import-test/load.sh
```

## Export + lint + preview

```bash
wrangler d1 export import-test --remote --output /tmp/import-test-export.sql
pscale import d1 lint --input /tmp/import-test-export.sql --format json
pscale import d1 start --input /tmp/import-test-export.sql --org ... --database ... --branch ... --dry-run --force --format json
```

The dry-run returns a `migration_id` and full import plan without loading Postgres. Run `start` again without `--dry-run` to import.

## Schema coverage (31 tables)

| Feature | Tables / columns |
|---------|------------------|
| Autoincrement PKs | most application tables |
| TEXT/UUID primary keys | `external_entities.id` |
| UUID foreign keys | `entity_links.entity_id` (table-level FK) |
| 0/1 booleans | `is_active`, `is_admin`, `is_public`, `is_read`, … |
| TEXT timestamps | `created_at`, `updated_at`, `due_at`, … |
| JSON in TEXT | `settings`, `metadata`, `profile_json`, `payload`, … |
| Multi-level FKs | org → user → project → task → comment → attachment |
| Junction tables | `team_members`, `project_tags`, `task_dependencies`, `entity_links` |
| Table-level FKs | composite PK tables, `entity_links`, … |
| Self-referential FKs | `tasks.parent_task_id`, `comments.parent_comment_id` |
| REAL columns | `invoices.subtotal`, `invoice_line_items.unit_price` |
| BLOB columns | `attachments.payload` (primary bulk data for large targets) |
| CHECK constraints | `tasks.status`, `tasks.priority` |
| UNIQUE constraints | column + composite (`projects`, `tags`, `external_entities`) |
| Secondary indexes | 12 indexes including composite + unique |
| ORM migration metadata | `__drizzle_*`, `_prisma_migrations`, Knex, Sequelize, Rails, Flyway, Liquibase, Django, Alembic, TypeORM, Goose (**skipped on import**) |

Default relational seed (before blob budget): 25 users/org, 40 projects/org, 20 tasks/project, 2 comments/task.

FTS5 virtual tables are **not** included — they cause `ParseDump` to fail on export.

## Default seed volume (~9 GB)

~8.5k attachments × 1 MiB blobs + relational rows. See `seed/SUMMARY.json` after generation.

Edit `SEED_TARGET_GB` or constants in `generate_seed.py` to scale (respect D1 10 GB cap).
