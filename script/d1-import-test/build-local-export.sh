#!/usr/bin/env bash
# Build a wrangler-style SQL export locally (schema + generated seed). No D1/wrangler.
#
# Usage:
#   ./script/d1-import-test/build-local-export.sh 1
#   SEED_DIR=/tmp/seed-5gb SEED_TARGET_GB=5 ./script/d1-import-test/build-local-export.sh 5
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SIZE_GB="${1:-${SEED_TARGET_GB:-1}}"
OUT="${D1_EXPORT:-/tmp/import-test-${SIZE_GB}gb-export.sql}"
SEED_DIR="${SEED_DIR:-$ROOT/seed/local-${SIZE_GB}gb}"
REGENERATE="${REGENERATE_SEED:-true}"

if [[ ! "$SIZE_GB" =~ ^[0-9]+([.][0-9]+)?$ ]]; then
  echo "ERROR: size must be a number (GB), got: $SIZE_GB" >&2
  exit 1
fi

seed_order=(
  organizations
  users
  teams
  team_members
  projects
  tags
  project_tags
  tasks
  task_dependencies
  comments
  attachments
  audit_log
  api_keys
  sessions
  notifications
  invoices
  line_items
  external_entities
  entity_links
)

echo "==> [export ${SIZE_GB}gb] Building local SQL export (~${SIZE_GB} GB target)"
echo "    output:   $OUT"
echo "    seed dir: $SEED_DIR"

mkdir -p "$SEED_DIR"

if [[ "$REGENERATE" == "true" ]]; then
  echo "==> [export ${SIZE_GB}gb] Generating seed"
  gen_start=$(date +%s)
  find "$SEED_DIR" -maxdepth 1 -type f -name '*.sql' -delete
  rm -f "$SEED_DIR/SUMMARY.json"
  SEED_DIR="$SEED_DIR" SEED_TARGET_GB="$SIZE_GB" python3 "$ROOT/generate_seed.py"
  gen_end=$(date +%s)
  echo "==> [export ${SIZE_GB}gb] Seed generation: $((gen_end - gen_start))s"
fi

if [[ ! -f "$SEED_DIR/SUMMARY.json" ]]; then
  echo "ERROR: seed generation did not produce $SEED_DIR/SUMMARY.json" >&2
  exit 1
fi

python3 - "$SEED_DIR/SUMMARY.json" "$SIZE_GB" <<'PY'
import json, sys
summary = json.load(open(sys.argv[1]))
target_gb = float(sys.argv[2])
target_bytes = int(target_gb * 1024**3)
actual = int(summary.get("estimated_blob_bytes", 0))
min_bytes = int(target_bytes * 0.99)
if actual < min_bytes:
    print(
        f"ERROR: seed blob storage {actual} bytes ({actual/1024**3:.3f} GB) "
        f"below target {target_bytes} bytes ({target_gb} GB)",
        file=sys.stderr,
    )
    sys.exit(1)
print(f"seed ok: {actual/1024**3:.3f} GB blob payload (target {target_gb} GB)")
PY

tmp="${OUT}.tmp.$$"
{
  echo "PRAGMA foreign_keys=OFF;"
  echo "-- Local D1-style export generated $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "-- Target size: ~${SIZE_GB} GB"
  echo
  grep -v '^PRAGMA foreign_keys' "$ROOT/schema.sql"
  echo
  for prefix in "${seed_order[@]}"; do
    mapfile -t files < <(find "$SEED_DIR" -maxdepth 1 -name "${prefix}_*.sql" | sort)
    for file in "${files[@]}"; do
      [[ -f "$file" ]] || continue
      cat "$file"
      echo
    done
  done
} > "$tmp"

mv "$tmp" "$OUT"
bytes=$(stat -f%z "$OUT" 2>/dev/null || stat -c%s "$OUT")
gb=$(python3 -c "print(round($bytes / (1024**3), 3))")
echo "==> [export ${SIZE_GB}gb] Wrote $OUT (${gb} GB)"
if [[ -n "${EXPORT_READY_FILE:-}" ]]; then
  touch "$EXPORT_READY_FILE"
fi
