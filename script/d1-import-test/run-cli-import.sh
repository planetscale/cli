#!/usr/bin/env bash
# End-to-end pscale import d1 CLI test against an existing wrangler SQL export.
#
# Prerequisites: pscale auth, local dev stack (singularity), Postgres database,
# and export file on disk. Does not load D1 or run wrangler export.
#
# Usage:
#   ./script/d1-import-test/run-cli-import.sh
#   IMPORT_PROFILE=9gb PSCALE_DB=cf-d1-import-9gb ./script/d1-import-test/run-cli-import.sh
#   D1_EXPORT=/tmp/my-export.sql PSCALE_DB=my-db ./script/d1-import-test/run-cli-import.sh
set -euo pipefail

CLI="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ORG="${PSCALE_ORG:-bb}"
BRANCH="${PSCALE_BRANCH:-main}"
API_URL="${PSCALE_API_URL:-http://api.pscaledev.com:3000/v1}"

case "${IMPORT_PROFILE:-smoke}" in
  smoke)
    DB_NAME="${PSCALE_DB:-d1-import-test}"
    EXPORT="${D1_EXPORT:-/tmp/import-test-export.sql}"
    ;;
  1gb)
    DB_NAME="${PSCALE_DB:-cf-d1-import-1gb}"
    EXPORT="${D1_EXPORT:-/tmp/import-test-1gb-export.sql}"
    ;;
  5gb)
    DB_NAME="${PSCALE_DB:-cf-d1-import-5gb}"
    EXPORT="${D1_EXPORT:-/tmp/import-test-5gb-export.sql}"
    ;;
  9gb)
    DB_NAME="${PSCALE_DB:-cf-d1-import-9gb}"
    EXPORT="${D1_EXPORT:-/tmp/import-test-9gb-export.sql}"
    ;;
  *)
    echo "ERROR: unknown IMPORT_PROFILE=${IMPORT_PROFILE:-} (use smoke, 1gb, 5gb, or 9gb)" >&2
    exit 1
    ;;
esac

METHOD="${IMPORT_METHOD:-pgloader}"
RUN_DIR="${IMPORT_RUN_DIR:-/tmp/d1-cli-import-$(date +%Y%m%d-%H%M%S)}"
mkdir -p "$RUN_DIR"

export PSCALE_DISABLE_DEV_WARNING=true
export PSCALE_TEST_MODE=1

PSCALE="${CLI}/pscale-test"
if [[ ! -x "$PSCALE" ]]; then
  echo "==> Building pscale-test"
  (cd "$CLI" && go build -o pscale-test ./cmd/pscale)
fi

pscale_cmd() {
  "$PSCALE" --api-url "$API_URL" "$@"
}

pscale_org_cmd() {
  "$PSCALE" --api-url "$API_URL" "$@" --org "$ORG"
}

json_field() {
  local file="$1" path="$2"
  python3 - "$file" "$path" <<'PY'
import json, sys
doc = json.load(open(sys.argv[1]))
path = sys.argv[2].split(".")
cur = doc
for part in path:
    if not isinstance(cur, dict) or part not in cur:
        cur = None
        break
    cur = cur[part]
if cur is None and path[0] in doc:
    cur = doc[path[0]]
if cur is None:
    sys.exit(1)
if isinstance(cur, bool):
    print("true" if cur else "false")
else:
    print(cur)
PY
}

require_json_ok() {
  local file="$1" label="$2"
  local status
  status="$(json_field "$file" status)"
  if [[ "$status" != "ok" && "$status" != "dry_run" ]]; then
    echo "ERROR: $label failed (status=$status). See $file" >&2
    exit 1
  fi
}

require_json_ok_or_warning() {
  local file="$1" label="$2"
  local status
  status="$(json_field "$file" status)"
  if [[ "$status" != "ok" && "$status" != "dry_run" && "$status" != "warning" ]]; then
    echo "ERROR: $label failed (status=$status). See $file" >&2
    exit 1
  fi
}

require_verify_matched() {
  local file="$1"
  local matched
  matched="$(json_field "$file" data.matched)"
  if [[ "$matched" != "true" ]]; then
    echo "ERROR: verify did not match. See $file" >&2
    exit 1
  fi
}

print_timings() {
  local file="$1"
  python3 - "$file" <<'PY'
import json, sys
doc = json.load(open(sys.argv[1]))
data = doc.get("data") or {}
timings = data.get("timings")
if not timings:
    print("  (no timings in CLI response — rebuild pscale-test)")
    sys.exit(0)
total = timings.get("total_ms", 0) / 1000
print(f"  total:          {total:.1f}s")
for key, label in [
    ("sqlite_staging_ms", "sqlite staging"),
    ("schema_ms", "schema apply"),
    ("pgloader_ms", "pgloader"),
    ("index_build_ms", "index build"),
    ("sequence_reset_ms", "sequence reset"),
]:
    ms = timings.get(key)
    if ms:
        print(f"  {label + ':':16} {ms/1000:.1f}s")
loads = timings.get("table_loads") or []
if loads:
    slow = sorted(loads, key=lambda x: x.get("ms", 0), reverse=True)[:5]
    print("  slowest tables:")
    for row in slow:
        print(f"    {row['table']}: {row['ms']/1000:.1f}s")
PY
}

if [[ ! -f "$EXPORT" ]]; then
  echo "ERROR: export not found: $EXPORT" >&2
  echo "Export first, e.g. wrangler d1 export import-test --remote --output $EXPORT" >&2
  exit 1
fi

if ! "$PSCALE" --api-url "$API_URL" auth check >/dev/null 2>&1; then
  echo "ERROR: pscale auth required. Run: pscale auth login --api-url ${PSCALE_AUTH_URL:-http://auth.pscaledev.com:3000}" >&2
  exit 1
fi

echo "==> CLI import test"
echo "    org:      $ORG"
echo "    database: $DB_NAME"
echo "    branch:   $BRANCH"
echo "    export:   $EXPORT ($(python3 -c "import os; print(round(os.path.getsize('$EXPORT')/(1024**3),2))") GB)"
echo "    method:   $METHOD"
echo "    run dir:  $RUN_DIR"

IMPORT_START=$(date +%s)

echo "==> import d1 doctor"
pscale_cmd import d1 doctor --format json | tee "$RUN_DIR/doctor.json"
require_json_ok_or_warning "$RUN_DIR/doctor.json" "doctor"

echo "==> import d1 lint"
pscale_cmd import d1 lint --input "$EXPORT" --format json | tee "$RUN_DIR/lint.json"
require_json_ok_or_warning "$RUN_DIR/lint.json" "lint"

echo "==> import d1 start --dry-run (preview)"
pscale_org_cmd import d1 start \
  --input "$EXPORT" \
  --database "$DB_NAME" \
  --branch "$BRANCH" \
  --dry-run \
  --force \
  --format json | tee "$RUN_DIR/preview.json"
require_json_ok "$RUN_DIR/preview.json" "preview"

MIGRATION_ID="$(python3 -c "
import json
d = json.load(open('$RUN_DIR/preview.json'))
print(d.get('migration_id') or (d.get('data') or {}).get('migration_id', ''))
")"
if [[ -z "$MIGRATION_ID" ]]; then
  echo "ERROR: could not read migration_id from $RUN_DIR/preview.json" >&2
  exit 1
fi
echo "    migration_id: $MIGRATION_ID"

echo "==> import d1 start"
START_WALL=$(date +%s)
pscale_org_cmd import d1 start \
  --database "$DB_NAME" \
  --branch "$BRANCH" \
  --input "$EXPORT" \
  --migration-id "$MIGRATION_ID" \
  --method "$METHOD" \
  --force \
  --format json | tee "$RUN_DIR/start.json"
START_WALL_END=$(date +%s)
require_json_ok "$RUN_DIR/start.json" "start"

echo "==> import d1 verify"
pscale_org_cmd import d1 verify \
  --database "$DB_NAME" \
  --branch "$BRANCH" \
  --migration-id "$MIGRATION_ID" \
  --input "$EXPORT" \
  --format json | tee "$RUN_DIR/verify.json"
require_json_ok "$RUN_DIR/verify.json" "verify"
require_verify_matched "$RUN_DIR/verify.json"

IMPORT_END=$(date +%s)
WALL_SEC=$((IMPORT_END - IMPORT_START))
START_SEC=$((START_WALL_END - START_WALL))

echo ""
echo "=============================================="
echo "CLI import passed"
echo "  migration_id:  $MIGRATION_ID"
echo "  database:      $ORG/$DB_NAME/$BRANCH"
echo "  wall clock:    ${WALL_SEC}s ($(python3 -c "print(round($WALL_SEC/60,1))") min)"
echo "  start phase:   ${START_SEC}s ($(python3 -c "print(round($START_SEC/60,1))") min)"
echo "  artifacts:     $RUN_DIR/"
echo "  CLI timings:"
print_timings "$RUN_DIR/start.json"
echo "=============================================="
