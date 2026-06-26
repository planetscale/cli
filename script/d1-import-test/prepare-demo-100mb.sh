#!/usr/bin/env bash
# Prepare ~100 MiB D1 export + SQLite + fresh Postgres DB for demos.
#
# Usage:
#   ./script/d1-import-test/prepare-demo-100mb.sh
#   PSCALE_DB=cf-d1-import-demo ./script/d1-import-test/prepare-demo-100mb.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI="$(cd "$ROOT/../.." && pwd)"
DEMO_DIR="${DEMO_DIR:-/tmp/d1-demo-100mb}"
DB="${PSCALE_DB:-cf-d1-import-demo-$(date +%Y%m%d)}"
TARGET_GB="$(python3 -c "print(100*1024*1024/(1024**3))")"

mkdir -p "$DEMO_DIR"

export PSCALE_DISABLE_DEV_WARNING=true
export PSCALE_TEST_MODE=1

echo "==> Building ~100 MiB export (target ${TARGET_GB} GB payload)"
SEED_DIR="$ROOT/seed/demo-100mb" \
  D1_EXPORT="$DEMO_DIR/import-test-100mb-export.sql" \
  SEED_TARGET_GB="$TARGET_GB" \
  REGENERATE_SEED=true \
  "$ROOT/build-local-export.sh" "$TARGET_GB"

echo "==> Building SQLite file"
SQLITE="$DEMO_DIR/import-test-100mb.sqlite"
rm -f "$SQLITE"
grep -v '^PRAGMA foreign_keys' "$DEMO_DIR/import-test-100mb-export.sql" | sqlite3 "$SQLITE"

echo "==> Provisioning database $DB"
PROVISION_READY_FILE="$DEMO_DIR/provision.ready" \
  "$ROOT/provision-database.sh" "$DB" > "$DEMO_DIR/provision.log" 2>&1

echo "$DB" > "$DEMO_DIR/database.name"
cat > "$DEMO_DIR/demo.env" <<EOF
export PSCALE_ORG=${PSCALE_ORG:-bb}
export PSCALE_DB=$DB
export PSCALE_BRANCH=${PSCALE_BRANCH:-main}
export PSCALE_API_URL=${PSCALE_API_URL:-http://api.pscaledev.com:3000/v1}
export D1_EXPORT=$DEMO_DIR/import-test-100mb-export.sql
export D1_SQLITE=$DEMO_DIR/import-test-100mb.sqlite
EOF

ls -lh "$DEMO_DIR/import-test-100mb-export.sql" "$SQLITE"
python3 - "$SQLITE" <<'PY'
import os, sqlite3, sys
db = sys.argv[1]
c = sqlite3.connect(db)
n = c.execute("select count(*) from attachments").fetchone()[0]
b = c.execute("select sum(length(payload)) from attachments").fetchone()[0]
print(f"  attachments: {n}")
print(f"  payload:     {b/(1024*1024):.1f} MiB")
print(f"  sqlite file: {os.path.getsize(db)/(1024*1024):.1f} MiB")
PY

echo ""
echo "Demo ready."
echo "  database:  $DB"
echo "  export:    $DEMO_DIR/import-test-100mb-export.sql"
echo "  sqlite:    $SQLITE"
echo "  env:       source $DEMO_DIR/demo.env"
echo ""
echo "Run import:"
echo "  source $DEMO_DIR/demo.env && cd $CLI && ./script/d1-import-test/run-cli-import.sh"
