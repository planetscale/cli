#!/usr/bin/env python3
"""Generate batched seed SQL for the import-test D1 database."""

from __future__ import annotations

import json
import os
import textwrap
import uuid
from datetime import datetime, timedelta, timezone
from pathlib import Path

OUT_DIR = Path(os.environ.get("SEED_DIR", str(Path(__file__).resolve().parent / "seed")))
DB_NAME = "import-test"

# Target Postgres logical storage (bytes in bytea payloads + relational overhead).
# SEED_TARGET_GB=1 means ~1 GiB of attachment payload bytes land in Postgres.
TARGET_GB = float(os.environ.get("SEED_TARGET_GB", "9"))
TARGET_BYTES = int(TARGET_GB * 1024**3)
RESERVED_BYTES = int(os.environ.get("SEED_RESERVED_BYTES", str(64 * 1024**2)))  # relational row headroom
# D1 rejects very large single statements (SQLITE_TOOBIG); hex blobs are ~2x raw size.
D1_MAX_STATEMENT_BYTES = int(os.environ.get("SEED_MAX_STATEMENT_BYTES", "100000"))


def payload_size() -> int:
    if "SEED_PAYLOAD_BYTES" in os.environ:
        requested = int(os.environ["SEED_PAYLOAD_BYTES"])
    elif TARGET_BYTES < 32 * 1024 * 1024:
        requested = 16 * 1024
    else:
        requested = 1024 * 1024
    # Hex literals are ~2x raw bytes; keep statements under D1_MAX_STATEMENT_BYTES.
    max_raw = max((D1_MAX_STATEMENT_BYTES - 4096) // 2, 4096)
    return min(requested, max_raw)


PAYLOAD_BYTES = payload_size()

BATCH_SIZE = 50
MAX_STATEMENT_BYTES = 90_000
LARGE_ROW_BYTES = 32_000

USERS_PER_ORG = int(os.environ.get("SEED_USERS_PER_ORG", "25"))
TEAMS_PER_ORG = int(os.environ.get("SEED_TEAMS_PER_ORG", "5"))
TASKS_PER_PROJECT = int(os.environ.get("SEED_TASKS_PER_PROJECT", "20"))
COMMENTS_PER_TASK = int(os.environ.get("SEED_COMMENTS_PER_TASK", "2"))
PROJECTS_PER_ORG = int(os.environ.get("SEED_PROJECTS_PER_ORG", "40"))
INVOICES_PER_ORG = int(os.environ.get("SEED_INVOICES_PER_ORG", "5"))
NOTIFICATIONS_PER_USER = int(os.environ.get("SEED_NOTIFICATIONS_PER_USER", "3"))


def esc(value: str | None) -> str:
    if value is None:
        return "NULL"
    return "'" + value.replace("'", "''") + "'"


def ts(base: datetime, minutes: int) -> str:
    return (base + timedelta(minutes=minutes)).replace(tzinfo=timezone.utc).isoformat().replace("+00:00", "Z")


def payload_bytes(attachment_id: int) -> bytes:
    prefix = f"attachment-{attachment_id}-".encode()
    fill = max(PAYLOAD_BYTES - len(prefix), 0)
    return prefix + (b"x" * fill)


def blob_literal(data: bytes) -> str:
    return "X'" + data.hex() + "'"


def write_batches(name: str, header: str, rows: list[str], *, max_bytes: int = MAX_STATEMENT_BYTES) -> None:
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    batch: list[str] = []
    batch_bytes = len(header.encode())
    file_idx = 0

    def flush() -> None:
        nonlocal batch, batch_bytes, file_idx
        if not batch:
            return
        path = OUT_DIR / f"{name}_{file_idx:03d}.sql"
        path.write_text(header + ",\n".join(batch) + ";\n", encoding="utf-8")
        file_idx += 1
        batch = []
        batch_bytes = len(header.encode())

    for row in rows:
        row_bytes = len(row.encode()) + 2
        limit = max_bytes
        if row_bytes > LARGE_ROW_BYTES:
            limit = max(row_bytes + 2, limit)
        if batch and batch_bytes + row_bytes > limit:
            flush()
        batch.append(row)
        batch_bytes += row_bytes
    flush()


def derive_volume() -> dict[str, int]:
    # Full target goes to blob payloads — this is what shows up as Postgres storage.
    blob_budget = TARGET_BYTES
    attachment_target = max(blob_budget // max(PAYLOAD_BYTES, 1), 1)

    return {
        "users_per_org": USERS_PER_ORG,
        "teams_per_org": TEAMS_PER_ORG,
        "projects_per_org": PROJECTS_PER_ORG,
        "tasks_per_project": TASKS_PER_PROJECT,
        "comments_per_task": COMMENTS_PER_TASK,
        "attachment_target": attachment_target,
        "target_bytes": TARGET_BYTES,
        "payload_bytes": PAYLOAD_BYTES,
        "reserved_bytes": RESERVED_BYTES,
    }


def main() -> None:
    if OUT_DIR.exists():
        for child in OUT_DIR.glob("*.sql"):
            child.unlink()
    else:
        OUT_DIR.mkdir(parents=True)

    volume = derive_volume()
    attachment_target = volume["attachment_target"]

    base = datetime(2024, 6, 1, 12, 0, 0, tzinfo=timezone.utc)

    bootstrap = OUT_DIR / "000_bootstrap.sql"
    bootstrap.write_text(
        textwrap.dedent(
            f"""
            INSERT INTO __drizzle_migrations (id, hash, created_at) VALUES
              (1, '001_initial', 1710000000),
              (2, '002_projects', 1710500000),
              (3, '003_external_entities', 1711000000);

            INSERT INTO _prisma_migrations (id, checksum, finished_at, migration_name, logs, rolled_back_at, started_at, applied_steps_count) VALUES
              ('seed-001', 'abc123', '{ts(base, 1)}', '20240101000000_init', NULL, NULL, '{ts(base, 0)}', 1);

            INSERT INTO knex_migrations (id, name, batch, migration_time) VALUES
              (1, '001_initial.js', 1, 1710000000000);

            INSERT INTO knex_migrations_lock ("index", is_locked) VALUES
              (1, 0);

            INSERT INTO sequelizemeta (name) VALUES
              ('20240101000000-init.js');

            INSERT INTO schema_migrations (version) VALUES
              ('20240101000000');

            INSERT INTO ar_internal_metadata (key, value, created_at, updated_at) VALUES
              ('environment', 'production', '{ts(base, 0)}', '{ts(base, 0)}');

            INSERT INTO flyway_schema_history (installed_rank, version, description, type, script, checksum, installed_by, installed_on, execution_time, success) VALUES
              (1, '1', 'initial', 'SQL', 'V1__initial.sql', 12345, 'seed', '{ts(base, 0)}', 42, 1);

            INSERT INTO databasechangelog (id, author, filename, dateexecuted, orderexecuted, exectype, md5sum, description, comments, tag, liquibase, contexts, labels, deployment_id) VALUES
              ('seed-001', 'seed', 'db/changelog/001.xml', '{ts(base, 0)}', 1, 'EXECUTED', 'abc', 'initial', NULL, NULL, '4.0', NULL, NULL, 'seed');

            INSERT INTO databasechangeloglock (id, locked, lockgranted, lockedby) VALUES
              (1, 0, NULL, NULL);

            INSERT INTO django_migrations (id, app, name, applied) VALUES
              (1, 'core', '0001_initial', '{ts(base, 0)}');

            INSERT INTO alembic_version (version_num) VALUES
              ('001_initial');

            INSERT INTO typeorm_metadata (type, "database", "schema", "table", name, value) VALUES
              ('seed', 'import-test', 'main', 'organizations', 'version', '1');

            INSERT INTO goose_db_version (id, version_id, is_applied, tstamp) VALUES
              (1, 1, 1, '{ts(base, 0)}');

            INSERT INTO organizations (id, slug, name, plan, is_active, settings, created_at, updated_at, deleted_at) VALUES
              (1, 'acme', 'Acme Corp', 'enterprise', 1, '{json.dumps({"timezone": "UTC", "features": ["audit", "sso"]})}', '{ts(base, 0)}', '{ts(base, 5)}', NULL),
              (2, 'globex', 'Globex Industries', 'pro', 1, '{json.dumps({"timezone": "America/New_York"})}', '{ts(base, 10)}', '{ts(base, 15)}', NULL);

            INSERT INTO users (id, org_id, email, display_name, role, is_active, is_admin, profile_json, last_login_at, created_at, updated_at) VALUES
              (1, 1, 'alice@acme.test', 'Alice Admin', 'admin', 1, 1, '{json.dumps({"title": "Platform Lead"})}', '{ts(base, 60)}', '{ts(base, 20)}', '{ts(base, 60)}'),
              (2, 1, 'bob@acme.test', 'Bob Builder', 'member', 1, 0, '{json.dumps({"title": "Engineer"})}', '{ts(base, 120)}', '{ts(base, 25)}', '{ts(base, 120)}'),
              (3, 2, 'carol@globex.test', 'Carol CFO', 'admin', 1, 1, NULL, '{ts(base, 180)}', '{ts(base, 30)}', '{ts(base, 180)}');

            INSERT INTO external_entities (id, org_id, name, kind, metadata, created_at) VALUES
              ('550e8400-e29b-41d4-a716-446655440000', 1, 'Bootstrap Webhook', 'webhook', '{json.dumps({"seed": True})}', '{ts(base, 40)}');
            """
        ).strip()
        + "\n",
        encoding="utf-8",
    )

    org_rows: list[str] = []
    user_rows: list[str] = []
    team_rows: list[str] = []
    team_member_rows: list[str] = []
    project_rows: list[str] = []
    tag_rows: list[str] = []
    project_tag_rows: list[str] = []
    task_rows: list[str] = []
    task_dep_rows: list[str] = []
    comment_rows: list[str] = []
    attachment_rows: list[str] = []
    audit_rows: list[str] = []
    api_key_rows: list[str] = []
    session_rows: list[str] = []
    notification_rows: list[str] = []
    invoice_rows: list[str] = []
    line_item_rows: list[str] = []
    external_entity_rows: list[str] = []
    entity_link_rows: list[str] = []

    org_id = 3
    user_id = 4
    team_id = 1
    project_id = 1
    tag_id = 1
    task_id = 1
    comment_id = 1
    attachment_id = 1
    audit_id = 1
    api_key_id = 1
    session_id = 1
    notification_id = 1
    invoice_id = 1
    line_item_id = 1

    while attachment_id <= attachment_target:
        slug = f"org-{org_id}"
        org_rows.append(
            f"({org_id}, {esc(slug)}, {esc(slug.replace('-', ' ').title())}, 'pro', 1, "
            f"{esc(json.dumps({'seed': True, 'index': org_id}))}, {esc(ts(base, org_id))}, {esc(ts(base, org_id + 1))}, NULL)"
        )

        entity_uuid = str(uuid.uuid4())
        external_entity_rows.append(
            f"({esc(entity_uuid)}, {org_id}, {esc(f'Entity {org_id}')}, 'integration', "
            f"{esc(json.dumps({'org': org_id, 'seed': True}))}, {esc(ts(base, org_id + 2))})"
        )

        org_user_ids: list[int] = []
        for u in range(USERS_PER_ORG):
            email = f"user{user_id}@{slug}.test"
            org_user_ids.append(user_id)
            user_rows.append(
                f"({user_id}, {org_id}, {esc(email)}, {esc(f'User {user_id}')}, 'member', 1, {1 if u == 0 else 0}, "
                f"{esc(json.dumps({'team': u % TEAMS_PER_ORG}))}, {esc(ts(base, user_id))}, {esc(ts(base, user_id))}, {esc(ts(base, user_id + 1))})"
            )
            for n in range(NOTIFICATIONS_PER_USER):
                notification_rows.append(
                    f"({notification_id}, {user_id}, 'mention', {esc(f'Mention {notification_id}')}, "
                    f"{esc('You were mentioned in a comment')}, {esc(json.dumps({'task_id': task_id}))}, "
                    f"{n % 2}, NULL, {esc(ts(base, notification_id))})"
                )
                notification_id += 1
            api_key_rows.append(
                f"({api_key_id}, {user_id}, {esc(f'key-{api_key_id}')}, {esc(f'pk_{api_key_id:04d}')}, 1, "
                f"{esc(json.dumps(['read', 'write']))}, {esc(ts(base, api_key_id + 720))}, {esc(ts(base, api_key_id))})"
            )
            api_key_id += 1
            session_rows.append(
                f"({session_id}, {user_id}, {esc(f'tok_{session_id:08x}')}, '127.0.0.1', 'seed-script', "
                f"{esc(ts(base, session_id))}, {esc(ts(base, session_id + 1440))}, {esc(ts(base, session_id))})"
            )
            session_id += 1
            user_id += 1

        org_team_ids: list[int] = []
        for t in range(TEAMS_PER_ORG):
            org_team_ids.append(team_id)
            team_rows.append(
                f"({team_id}, {org_id}, {esc(f'Team {team_id}')}, 0, {esc(ts(base, team_id))})"
            )
            for member_idx, member_user in enumerate(org_user_ids[:5]):
                team_member_rows.append(
                    f"({team_id}, {member_user}, {esc('lead' if member_idx == 0 else 'member')}, {esc(ts(base, team_id + member_idx))})"
                )
            team_id += 1

        org_tag_ids: list[int] = []
        for label in ("backend", "frontend", "infra", "design"):
            org_tag_ids.append(tag_id)
            tag_rows.append(f"({tag_id}, {org_id}, {esc(label)}, {esc('#' + format(tag_id * 111111 % 0xFFFFFF, '06x'))})")
            tag_id += 1

        org_project_ids: list[int] = []
        for p in range(PROJECTS_PER_ORG):
            if attachment_id > attachment_target:
                break
            owner = org_user_ids[p % len(org_user_ids)]
            team = org_team_ids[p % len(org_team_ids)]
            org_project_ids.append(project_id)
            project_rows.append(
                f"({project_id}, {org_id}, {owner}, {team}, {esc(f'project-{project_id}')}, {esc(f'Project {project_id}')}, "
                f"{esc(f'Description for project {project_id}')}, {p % 3 == 0}, 0, "
                f"{esc(json.dumps({'priority': p % 5, 'labels': ['seed']}))}, {esc(ts(base, project_id))}, {esc(ts(base, project_id + 2))})"
            )
            for tg in org_tag_ids[:2]:
                project_tag_rows.append(f"({project_id}, {tg})")

            prev_task_in_project: int | None = None
            for tk in range(TASKS_PER_PROJECT):
                if attachment_id > attachment_target:
                    break
                assignee = org_user_ids[(p + tk) % len(org_user_ids)]
                parent = prev_task_in_project if tk > 0 and tk % 4 == 0 else None
                status = ("open", "in_progress", "done", "cancelled")[tk % 4]
                task_rows.append(
                    f"({task_id}, {project_id}, {assignee}, {parent if parent else 'NULL'}, "
                    f"{esc(f'Task {task_id}')}, {esc(f'Body for task {task_id}')}, {esc(status)}, {(tk % 5) + 1}, "
                    f"{round(1.5 + (task_id % 7), 2)}, {1 if tk % 6 == 0 else 0}, "
                    f"{esc(json.dumps({'tags': ['seed', status]}))}, {esc(ts(base, task_id + 1000))}, "
                    f"{esc(ts(base, task_id + 2000)) if status == 'done' else 'NULL'}, "
                    f"{esc(ts(base, task_id))}, {esc(ts(base, task_id + 1))})"
                )
                if prev_task_in_project is not None and tk % 3 == 0:
                    task_dep_rows.append(f"({task_id}, {prev_task_in_project})")
                prev_task_in_project = task_id

                if tk == 0:
                    entity_link_rows.append(
                        f"({esc(entity_uuid)}, {task_id}, {esc(ts(base, task_id + 3))})"
                    )

                for c in range(COMMENTS_PER_TASK):
                    if attachment_id > attachment_target:
                        break
                    author = org_user_ids[(c + tk) % len(org_user_ids)]
                    parent_comment = comment_id - 1 if c > 0 else None
                    comment_rows.append(
                        f"({comment_id}, {task_id}, {author}, {parent_comment if parent_comment else 'NULL'}, "
                        f"{esc(f'Comment {comment_id} on task {task_id}')}, {1 if c > 0 else 0}, "
                        f"{esc(ts(base, comment_id))}, {esc(ts(base, comment_id + 1))})"
                    )
                    blob = payload_bytes(attachment_id)
                    attachment_rows.append(
                        f"({attachment_id}, {comment_id}, {esc(f'file-{attachment_id}.bin')}, 'application/octet-stream', "
                        f"{len(blob)}, {esc(f'sha256:{attachment_id:08x}')}, {blob_literal(blob)}, {esc(ts(base, attachment_id))})"
                    )
                    attachment_id += 1
                    comment_id += 1

                task_id += 1
            project_id += 1

        for inv in range(INVOICES_PER_ORG):
            subtotal = round(1000 + inv * 250.5, 2)
            tax = round(subtotal * 0.08, 2)
            total = round(subtotal + tax, 2)
            invoice_rows.append(
                f"({invoice_id}, {org_id}, {esc(f'INV-{invoice_id:05d}')}, {subtotal}, {tax}, {total}, 'sent', "
                f"{esc(ts(base, invoice_id + 5000))}, {esc(ts(base, invoice_id + 6000))}, NULL, {esc(ts(base, invoice_id))})"
            )
            for line in range(3):
                qty = line + 1
                unit = round(50.25 + line, 2)
                amount = round(qty * unit, 2)
                related_task = max(task_id - 1 - line, 1)
                line_item_rows.append(
                    f"({line_item_id}, {invoice_id}, {related_task}, {esc(f'Line {line_item_id}')}, "
                    f"{qty}, {unit}, {amount})"
                )
                line_item_id += 1
            invoice_id += 1

        for a in range(10):
            if not org_project_ids:
                break
            actor = org_user_ids[a % len(org_user_ids)]
            audit_rows.append(
                f"({audit_id}, {org_id}, {actor}, 'update', 'project', {org_project_ids[a % len(org_project_ids)]}, "
                f"{esc(json.dumps({'field': 'name', 'seed': True}))}, {esc(ts(base, audit_id + 8000))})"
            )
            audit_id += 1

        org_id += 1

    actual_attachments = attachment_id - 1
    estimated_blob_bytes = actual_attachments * PAYLOAD_BYTES
    if estimated_blob_bytes < int(TARGET_BYTES * 0.99):
        raise SystemExit(
            f"seed under storage target: generated {estimated_blob_bytes} bytes, "
            f"need >= {int(TARGET_BYTES * 0.99)} ({TARGET_GB} GiB Postgres payload target)"
        )

    write_batches("organizations", "INSERT INTO organizations (id, slug, name, plan, is_active, settings, created_at, updated_at, deleted_at) VALUES\n", org_rows)
    write_batches("users", "INSERT INTO users (id, org_id, email, display_name, role, is_active, is_admin, profile_json, last_login_at, created_at, updated_at) VALUES\n", user_rows)
    write_batches("teams", "INSERT INTO teams (id, org_id, name, is_archived, created_at) VALUES\n", team_rows)
    write_batches("team_members", "INSERT INTO team_members (team_id, user_id, role, joined_at) VALUES\n", team_member_rows)
    write_batches("projects", "INSERT INTO projects (id, org_id, owner_user_id, team_id, slug, name, description, is_public, is_archived, metadata, created_at, updated_at) VALUES\n", project_rows)
    write_batches("tags", "INSERT INTO tags (id, org_id, label, color) VALUES\n", tag_rows)
    write_batches("project_tags", "INSERT INTO project_tags (project_id, tag_id) VALUES\n", project_tag_rows)
    write_batches("tasks", "INSERT INTO tasks (id, project_id, assignee_user_id, parent_task_id, title, body, status, priority, estimate_hours, is_blocked, labels_json, due_at, completed_at, created_at, updated_at) VALUES\n", task_rows)
    write_batches("task_dependencies", "INSERT INTO task_dependencies (task_id, depends_on_task_id) VALUES\n", task_dep_rows)
    write_batches("comments", "INSERT INTO comments (id, task_id, author_user_id, parent_comment_id, body, is_edited, created_at, updated_at) VALUES\n", comment_rows)
    write_batches(
        "attachments",
        "INSERT INTO attachments (id, comment_id, filename, content_type, byte_size, checksum, payload, uploaded_at) VALUES\n",
        attachment_rows,
        max_bytes=min(D1_MAX_STATEMENT_BYTES, max(PAYLOAD_BYTES * 2 + 4096, MAX_STATEMENT_BYTES)),
    )
    write_batches("audit_log", "INSERT INTO audit_log (id, org_id, actor_user_id, action, entity_type, entity_id, payload, created_at) VALUES\n", audit_rows)
    write_batches("api_keys", "INSERT INTO api_keys (id, user_id, name, key_prefix, is_active, scopes_json, expires_at, created_at) VALUES\n", api_key_rows)
    write_batches("sessions", "INSERT INTO sessions (id, user_id, token_hash, ip_address, user_agent, last_seen_at, expires_at, created_at) VALUES\n", session_rows)
    write_batches("notifications", "INSERT INTO notifications (id, user_id, kind, title, body, payload, is_read, read_at, created_at) VALUES\n", notification_rows)
    write_batches("invoices", "INSERT INTO invoices (id, org_id, invoice_number, subtotal, tax, total, status, issued_at, due_at, paid_at, created_at) VALUES\n", invoice_rows)
    write_batches("line_items", "INSERT INTO invoice_line_items (id, invoice_id, task_id, description, quantity, unit_price, amount) VALUES\n", line_item_rows)
    write_batches(
        "external_entities",
        "INSERT INTO external_entities (id, org_id, name, kind, metadata, created_at) VALUES\n",
        external_entity_rows,
    )
    write_batches("entity_links", "INSERT INTO entity_links (entity_id, task_id, linked_at) VALUES\n", entity_link_rows)

    estimated_blob_bytes = (attachment_id - 1) * PAYLOAD_BYTES
    if attachment_id - 1 < attachment_target:
        raise SystemExit(
            f"ERROR: generated {attachment_id - 1} attachments, expected {attachment_target} "
            f"({estimated_blob_bytes} bytes, target {TARGET_BYTES})"
        )
    min_bytes = int(TARGET_BYTES * 0.99)
    if estimated_blob_bytes < min_bytes:
        raise SystemExit(
            f"ERROR: blob bytes {estimated_blob_bytes} below target minimum {min_bytes}"
        )
    summary = {
        **volume,
        "organizations": len(org_rows) + 2,
        "users": len(user_rows) + 3,
        "teams": len(team_rows),
        "projects": len(project_rows),
        "tasks": len(task_rows),
        "comments": len(comment_rows),
        "attachments": attachment_id - 1,
        "external_entities": len(external_entity_rows) + 1,
        "entity_links": len(entity_link_rows),
        "estimated_blob_bytes": estimated_blob_bytes,
        "estimated_total_bytes": estimated_blob_bytes + RESERVED_BYTES,
        "seed_files": len(list(OUT_DIR.glob("*.sql"))),
    }
    (OUT_DIR / "SUMMARY.json").write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(summary, indent=2))


if __name__ == "__main__":
    main()
