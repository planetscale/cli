# D1 import — pre-deploy checklist

Track open items before shipping `pscale import d1` publicly.

## ORM metadata baselining (required UX)

**Problem:** ORM migration bookkeeping tables (`_prisma_migrations`, `schema_migrations`, `__drizzle_migrations`, Knex/Sequelize/Flyway/Liquibase/Django/Alembic/TypeORM/Goose tables) are **intentionally skipped** on import. Application schema + data land on Postgres; ORM metadata does not.

Users must **manually baseline** their ORM on Postgres after import (stamp migrations as applied, or create a fresh initial migration from the imported schema). We cannot safely auto-run their ORM CLI without knowing migration files on disk — importing SQLite metadata rows or replaying migrations would lie about schema state or collide with existing tables.

**Today:**
- `lint` emits info-level issues per detected ORM with remediation text (`orm_metadata.go`)
- `convert-schema`, pgloader data-only, and `verify` skip these tables
- `import start` / `verify` next steps do **not** surface ORM baseline guidance prominently

**Before deploy — address:**
- [ ] Post-import `next_steps` aggregating detected ORMs from lint (after successful `start` / `verify`)
- [ ] JSON field on import/verify response, e.g. `skipped_orm_tables` + `orm_baseline_hints[]`
- [ ] Optional read-only helper: `pscale import d1 baseline-plan --input …` (copy-paste commands per ORM)
- [ ] Agent skill or `.agents/docs` section: post-import workflow must include ORM baseline when `*_MIGRATIONS` lint codes present
- [ ] Public docs: explicit “ORM bookkeeping is not migrated; baseline on Postgres” callout

**Out of scope for v1 (unless product asks):**
- Fully automated ORM baseline (requires `--orm` + migrations dir; risky without user confirmation)
- Importing SQLite ORM metadata rows into Postgres

See `orm_metadata.go` for per-ORM remediation strings.
