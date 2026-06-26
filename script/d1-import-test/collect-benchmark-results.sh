#!/usr/bin/env bash
# Collect import timings and Postgres storage sizes into a report.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI="$(cd "$ROOT/../.." && pwd)"
STATE_DIR="${1:-}"
REPORT="${2:-${STATE_DIR}/report.txt}"

if [[ -z "$STATE_DIR" || ! -d "$STATE_DIR" ]]; then
  echo "ERROR: state dir required" >&2
  exit 1
fi

export PSCALE_DISABLE_DEV_WARNING=true
export PSCALE_TEST_MODE=1
export PSCALE_ALLOW_NONINTERACTIVE_SHELL=1
PSCALE="${CLI}/pscale-test"
API_URL="${PSCALE_API_URL:-http://api.pscaledev.com:3000/v1}"
ORG="${PSCALE_ORG:-bb}"

{
  echo "D1 import storage benchmark report"
  echo "Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "State dir: $STATE_DIR"
  echo ""

  for size in 1 5 9; do
    db_file="$STATE_DIR/db-${size}gb.name"
    [[ -f "$db_file" ]] || continue
    db="$(cat "$db_file")"
    echo "=== ${size} GB ==="
    echo "database: $db"

    if [[ -f "$STATE_DIR/import-${size}gb.done" ]]; then
      echo "import: SUCCESS"
    elif [[ -f "$STATE_DIR/import-${size}gb.failed" ]]; then
      echo "import: FAILED"
    else
      echo "import: incomplete"
    fi

    grep "^${size}gb:" "$STATE_DIR/results.txt" 2>/dev/null || true

    run_dir="$(grep "^${size}gb:" "$STATE_DIR/results.txt" 2>/dev/null | cut -d: -f3- || true)"
    if [[ -n "$run_dir" && -f "$run_dir/start.json" ]]; then
      python3 - "$run_dir/start.json" <<'PY'
import json, sys
raw = open(sys.argv[1]).read()
idx = raw.rfind('{"status"')
if idx < 0:
    sys.exit(0)
d = json.loads(raw[idx:])
t = (d.get("data") or {}).get("timings") or {}
for k in ["total_ms", "schema_ms", "pgloader_ms", "index_build_ms", "sequence_reset_ms"]:
    v = t.get(k)
    if v is not None:
        print(f"  {k}: {v/1000:.1f}s")
loads = t.get("table_loads") or []
if loads:
    att = next((x for x in loads if x.get("table") == "attachments"), None)
    if att:
        print(f"  attachments_pgloader: {att['ms']/1000:.1f}s")
PY
    fi

    if echo "SELECT 1" | "$PSCALE" --api-url "$API_URL" shell "$db" main --org "$ORG" >/dev/null 2>&1; then
      echo "  postgres storage:"
      echo "SELECT pg_size_pretty(sum(octet_length(payload))) AS attachments_payload,
        pg_size_pretty(sum(pg_total_relation_size(format('%I.%I', schemaname, tablename)::regclass))) AS public_tables_on_disk,
        pg_size_pretty(pg_database_size(current_database())) AS pg_database_size,
        (SELECT count(*) FROM attachments) AS attachment_rows;" | \
        "$PSCALE" --api-url "$API_URL" shell "$db" main --org "$ORG" 2>/dev/null | grep -v "^$" | tail -4 | sed 's/^/    /'
    fi
    echo ""
  done
} > "$REPORT"

cat "$REPORT"
