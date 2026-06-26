#!/usr/bin/env bash
# Create a PlanetScale Postgres database and wait until ready.
#
# Usage:
#   ./script/d1-import-test/provision-database.sh cf-d1-import-5gb-20260625-120000
#   PSCALE_DB_FILE=/tmp/my-db.name ./script/d1-import-test/provision-database.sh cf-d1-import-5gb
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI="$(cd "$ROOT/../.." && pwd)"

ORG="${PSCALE_ORG:-bb}"
BRANCH="${PSCALE_BRANCH:-main}"
REGION="${PSCALE_REGION:-dev-aws-us-east-1-1}"
CLUSTER_SIZE="${PSCALE_CLUSTER_SIZE:-PS_10}"
PG_MAJOR_VERSION="${PSCALE_PG_MAJOR_VERSION:-17}"
REPLICAS="${PSCALE_REPLICAS:-0}"
API_URL="${PSCALE_API_URL:-http://api.pscaledev.com:3000/v1}"

NAME="${1:-}"
if [[ -z "$NAME" ]]; then
  echo "ERROR: database name required" >&2
  exit 1
fi

PSCALE="${CLI}/pscale-test"
if [[ ! -x "$PSCALE" ]]; then
  (cd "$CLI" && go build -o pscale-test ./cmd/pscale)
fi

db_cmd() {
  "$PSCALE" --api-url "$API_URL" database "$@" --org "$ORG"
}

echo "==> [provision] Creating Postgres database $NAME"
db_cmd create "$NAME" \
  --engine postgresql \
  --region "$REGION" \
  --cluster-size "$CLUSTER_SIZE" \
  --replicas "$REPLICAS" \
  --major-version "$PG_MAJOR_VERSION"

echo "==> [provision] Waiting for branch $BRANCH to be ready"
PSCALE_CMD=("$PSCALE" --api-url "$API_URL")
deadline=$((SECONDS + ${PROVISION_TIMEOUT_SEC:-900}))
while (( SECONDS < deadline )); do
  ready="$("${PSCALE_CMD[@]}" branch show "$NAME" "$BRANCH" --format json --org "$ORG" 2>/dev/null | python3 -c "import json,sys; print(json.load(sys.stdin).get('ready', False))" 2>/dev/null || echo False)"
  if [[ "$ready" == "True" ]]; then
    break
  fi
  sleep 5
done

if [[ "$ready" != "True" ]]; then
  echo "ERROR: branch not ready after ${PROVISION_TIMEOUT_SEC:-900}s" >&2
  exit 1
fi

echo "==> [provision] Database ready: $NAME (branch ready)"
if [[ -n "${PROVISION_READY_FILE:-}" ]]; then
  touch "$PROVISION_READY_FILE"
fi
if [[ -n "${PSCALE_DB_FILE:-}" ]]; then
  echo "$NAME" > "$PSCALE_DB_FILE"
fi
echo "$NAME"
