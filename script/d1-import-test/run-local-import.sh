#!/usr/bin/env bash
# Provision a fresh PlanetScale Postgres DB, then run the CLI import test.
#
# By default each run targets a new empty database (FRESH_DB=new).
# For CLI-only testing against an already-provisioned DB, use run-cli-import.sh.
#
# Usage:
#   IMPORT_PROFILE=9gb ./script/d1-import-test/run-local-import.sh
#   FRESH_DB=recreate PSCALE_DB=cf-d1-import-9gb ./script/d1-import-test/run-local-import.sh
#   FRESH_DB=reuse SKIP_DB_CREATE=true ./script/d1-import-test/run-cli-import.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI="$(cd "$ROOT/../.." && pwd)"
MONOREPO_ROOT="$(cd "$CLI/../.." && pwd)"

ORG="${PSCALE_ORG:-bb}"
BRANCH="${PSCALE_BRANCH:-main}"
REGION="${PSCALE_REGION:-dev-aws-us-east-1-1}"
CLUSTER_SIZE="${PSCALE_CLUSTER_SIZE:-PS_10}"
PG_MAJOR_VERSION="${PSCALE_PG_MAJOR_VERSION:-17}"
REPLICAS="${PSCALE_REPLICAS:-0}"
API_URL="${PSCALE_API_URL:-http://api.pscaledev.com:3000/v1}"
AUTH_URL="${PSCALE_AUTH_URL:-http://auth.pscaledev.com:3000}"
FRESH_DB="${FRESH_DB:-new}"
AUTO_EXPORT="${AUTO_EXPORT:-false}"

case "${IMPORT_PROFILE:-smoke}" in
  smoke)
    BASE_DB="${PSCALE_DB:-d1-import-test}"
    EXPORT="${D1_EXPORT:-/tmp/import-test-export.sql}"
    ;;
  1gb)
    BASE_DB="${PSCALE_DB:-cf-d1-import-1gb}"
    EXPORT="${D1_EXPORT:-/tmp/import-test-1gb-export.sql}"
    ;;
  5gb)
    BASE_DB="${PSCALE_DB:-cf-d1-import-5gb}"
    EXPORT="${D1_EXPORT:-/tmp/import-test-5gb-export.sql}"
    ;;
  9gb)
    BASE_DB="${PSCALE_DB:-cf-d1-import-9gb}"
    EXPORT="${D1_EXPORT:-/tmp/import-test-9gb-export.sql}"
    ;;
  *)
    echo "ERROR: unknown IMPORT_PROFILE=${IMPORT_PROFILE:-} (use smoke, 1gb, 5gb, or 9gb)" >&2
    exit 1
    ;;
esac

export PSCALE_DISABLE_DEV_WARNING=true
export PSCALE_TEST_MODE=1
export PSCALE_ORG="$ORG"
export PSCALE_BRANCH="$BRANCH"
export D1_EXPORT="$EXPORT"

PSCALE="${CLI}/pscale-test"
if [[ ! -x "$PSCALE" ]]; then
  (cd "$CLI" && go build -o pscale-test ./cmd/pscale)
fi

pscale_cmd() {
  "$PSCALE" --api-url "$API_URL" "$@" --org "$ORG"
}

db_cmd() {
  "$PSCALE" --api-url "$API_URL" database "$@" --org "$ORG"
}

db_exists() {
  db_cmd show "$1" --format json >/dev/null 2>&1
}

region_pskube_alias() {
  case "$1" in
    dev-aws-us-east-1-1) echo dev-aws-fatih-useast1 ;;
    dev-aws-us-east-1-2) echo dev-aws-noonan-useast1 ;;
    dev-aws-us-east-1-3) echo dev-aws-shared-useast1 ;;
    dev-aws-us-east-2-1) echo dev-aws-mdlayher-useast2 ;;
    dev-aws-us-east-2-2) echo dev-aws-orch1-useast2 ;;
    dev-aws-us-east-2-3) echo dev-aws-orch2-useast2 ;;
    dev-aws-us-west-2-1) echo dev-aws-fatih-uswest2 ;;
    dev-aws-us-west-2-4) echo dev-aws-shared-uswest2 ;;
    dev-aws-us-west-2-5) echo dev-aws-rcrowley-uswest2 ;;
    dev-aws-us-west-2-6) echo dev-aws-orch1-uswest2 ;;
    dev-aws-eu-central-1-1) echo dev-aws-fatih-eucentral1 ;;
    dev-aws-eu-central-1-2) echo dev-aws-amir-eucentral1 ;;
    dev-aws-eu-west-2-1) echo dev-aws-shared-euwest2 ;;
    dev-aws-eu-west-2-2) echo dev-aws-shared2-euwest2 ;;
    dev-gcp-us-east4-1) echo dev-gcp-mdlayher-useast4 ;;
    *) echo "$1" ;;
  esac
}

check_branch_pskube() {
  local branch_id="$1"
  local alias
  alias="$(region_pskube_alias "$REGION")"
  echo "==> pskube branch status (alias: $alias, branch: $branch_id)"
  if ! command -v pskube >/dev/null 2>&1; then
    echo "pskube not installed; skip k8s status"
    return 0
  fi
  pskube "$alias" get horizonclusters.hzdb.co -n hz-data "hzc-${branch_id}" -o wide 2>/dev/null || true
  pskube "$alias" get horizoninstances.hzdb.co -n hz-data -l "hzdb.co/branch=${branch_id}" -o wide 2>/dev/null || true
}

provision_database() {
  local name="$1"
  echo "==> Creating Postgres database $name (org: $ORG, region: $REGION, size: $CLUSTER_SIZE)"
  db_cmd create "$name" \
    --engine postgresql \
    --region "$REGION" \
    --cluster-size "$CLUSTER_SIZE" \
    --replicas "$REPLICAS" \
    --major-version "$PG_MAJOR_VERSION" \
    --wait

  local branch_id
  branch_id="$(pscale_cmd branch show "$name" "$BRANCH" --format json | python3 -c "import json,sys; print(json.load(sys.stdin)['id'])")"
  check_branch_pskube "$branch_id"
}

echo "==> Checking pscale auth"
if ! "$PSCALE" --api-url "$API_URL" auth check >/dev/null 2>&1; then
  echo "Run interactively first:"
  echo "  pscale auth login --api-url $AUTH_URL"
  exit 1
fi

echo "==> Checking singularity"
if ! curl -sS --connect-timeout 2 -o /dev/null http://127.0.0.1:8080/ 2>/dev/null; then
  echo "Singularity not responding on :8080. Restart with:"
  echo "  cd $MONOREPO_ROOT && nix develop -c process-compose process restart singularity -p 8181 --address localhost"
  exit 1
fi

if [[ ! -f "$EXPORT" ]]; then
  if [[ "$AUTO_EXPORT" == "true" ]]; then
    echo "==> Exporting D1 import-test -> $EXPORT"
    wrangler d1 export import-test --remote --output "$EXPORT"
  else
    echo "ERROR: export not found: $EXPORT" >&2
    exit 1
  fi
fi

DB_NAME="$BASE_DB"
if [[ -n "${PSCALE_DB:-}" ]]; then
  DB_NAME="$PSCALE_DB"
  echo "==> Using pre-provisioned database: $DB_NAME"
elif [[ "${SKIP_DB_CREATE:-false}" == "true" ]]; then
  echo "ERROR: SKIP_DB_CREATE=true but PSCALE_DB is not set" >&2
  exit 1
else
case "$FRESH_DB" in
  new)
    DB_NAME="${BASE_DB}-$(date +%Y%m%d-%H%M%S)"
    echo "==> Fresh database (FRESH_DB=new): $DB_NAME"
    provision_database "$DB_NAME"
    ;;
  recreate)
    DB_NAME="$BASE_DB"
    echo "==> Recreating database (FRESH_DB=recreate): $DB_NAME"
    if db_exists "$DB_NAME"; then
      echo "    deleting existing database"
      db_cmd delete "$DB_NAME" --force
    fi
    provision_database "$DB_NAME"
    ;;
  reuse)
    DB_NAME="$BASE_DB"
    echo "==> Reusing database (FRESH_DB=reuse): $DB_NAME"
    echo "    import will DROP SCHEMA public CASCADE before applying DDL"
    if ! db_exists "$DB_NAME"; then
      provision_database "$DB_NAME"
    fi
    ;;
  *)
    echo "ERROR: unknown FRESH_DB=$FRESH_DB (use new, recreate, or reuse)" >&2
    exit 1
    ;;
esac
fi

export PSCALE_DB="$DB_NAME"
exec "$ROOT/run-cli-import.sh"
