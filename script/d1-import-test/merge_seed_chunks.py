#!/usr/bin/env python3
"""Merge per-batch seed SQL files into large chunks for wrangler d1 execute --file.

D1 limits (see Cloudflare docs):
  - Max SQL statement length: 100 KB (each statement in a file)
  - Max import file size via wrangler: 5 GB

Each small seed file is kept intact (never split). Chunks are concatenations of
complete statements, target ~50 MB per chunk by default.
"""

from __future__ import annotations

import json
import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
SEED_DIR = ROOT / "seed"
CHUNKS_DIR = SEED_DIR / "chunks"

# Keep in sync with load.sh / load-bulk.sh seed order.
SEED_ORDER: list[str] = [
    "000_bootstrap.sql",
    "organizations",
    "users",
    "teams",
    "team_members",
    "projects",
    "tags",
    "project_tags",
    "tasks",
    "task_dependencies",
    "comments",
    "attachments",
    "audit_log",
    "api_keys",
    "sessions",
    "notifications",
    "invoices",
    "line_items",
    "external_entities",
    "entity_links",
]

D1_MAX_STATEMENT_BYTES = int(os.environ.get("SEED_MAX_STATEMENT_BYTES", "100000"))
CHUNK_TARGET_BYTES = int(float(os.environ.get("CHUNK_TARGET_MB", "50")) * 1024 * 1024)


def ordered_seed_files() -> list[Path]:
    files: list[Path] = []
    for prefix in SEED_ORDER:
        if prefix.endswith(".sql"):
            path = SEED_DIR / prefix
            if path.is_file():
                files.append(path)
            continue
        for path in sorted(SEED_DIR.glob(f"{prefix}_*.sql")):
            files.append(path)
    return files


def validate_statements(files: list[Path]) -> None:
    oversize: list[tuple[str, int]] = []
    for path in files:
        size = path.stat().st_size
        if size > D1_MAX_STATEMENT_BYTES:
            oversize.append((path.name, size))
    if oversize:
        sample = oversize[:5]
        details = ", ".join(f"{name} ({size} bytes)" for name, size in sample)
        extra = f" (+{len(oversize) - 5} more)" if len(oversize) > 5 else ""
        raise SystemExit(
            f"ERROR: {len(oversize)} seed file(s) exceed D1 max statement size "
            f"({D1_MAX_STATEMENT_BYTES} bytes): {details}{extra}\n"
            "Regenerate seed with a smaller SEED_PAYLOAD_BYTES or lower SEED_MAX_STATEMENT_BYTES."
        )


def merge_chunks(files: list[Path], *, dry_run: bool = False) -> dict:
    if not files:
        raise SystemExit(f"no seed SQL files found under {SEED_DIR}")

    validate_statements(files)

    if not dry_run:
        if CHUNKS_DIR.exists():
            for child in CHUNKS_DIR.glob("chunk_*.sql"):
                child.unlink()
        else:
            CHUNKS_DIR.mkdir(parents=True)

    chunk_idx = 0
    current_files: list[Path] = []
    current_bytes = 0
    chunk_manifest: list[dict] = []

    def flush() -> None:
        nonlocal chunk_idx, current_files, current_bytes
        if not current_files:
            return
        chunk_idx += 1
        out_path = CHUNKS_DIR / f"chunk_{chunk_idx:04d}.sql"
        file_count = len(current_files)
        total_bytes = sum(f.stat().st_size for f in current_files)
        entry = {
            "chunk": chunk_idx,
            "file": out_path.name,
            "source_files": file_count,
            "bytes": total_bytes,
        }
        chunk_manifest.append(entry)
        if not dry_run:
            with out_path.open("wb") as out:
                for src in current_files:
                    data = src.read_bytes()
                    out.write(data)
                    if data and not data.endswith(b"\n"):
                        out.write(b"\n")
        current_files = []
        current_bytes = 0

    for path in files:
        size = path.stat().st_size
        if size > CHUNK_TARGET_BYTES:
            flush()
            chunk_idx += 1
            out_path = CHUNKS_DIR / f"chunk_{chunk_idx:04d}.sql"
            chunk_manifest.append(
                {
                    "chunk": chunk_idx,
                    "file": out_path.name,
                    "source_files": 1,
                    "bytes": size,
                    "note": "oversized single file kept alone",
                }
            )
            if not dry_run:
                out_path.write_bytes(path.read_bytes())
            continue
        if current_files and current_bytes + size > CHUNK_TARGET_BYTES:
            flush()
        current_files.append(path)
        current_bytes += size
    flush()

    total_bytes = sum(e["bytes"] for e in chunk_manifest)
    return {
        "chunk_target_bytes": CHUNK_TARGET_BYTES,
        "d1_max_statement_bytes": D1_MAX_STATEMENT_BYTES,
        "source_files": len(files),
        "chunks": len(chunk_manifest),
        "total_bytes": total_bytes,
        "manifest": chunk_manifest,
    }


def main() -> None:
    dry_run = "--dry-run" in sys.argv
    files = ordered_seed_files()
    summary = merge_chunks(files, dry_run=dry_run)
    summary_path = CHUNKS_DIR / "MANIFEST.json"
    if not dry_run:
        CHUNKS_DIR.mkdir(parents=True, exist_ok=True)
        summary_path.write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(summary, indent=2))
    if dry_run:
        print(f"(dry run — no files written under {CHUNKS_DIR})", file=sys.stderr)


if __name__ == "__main__":
    main()
