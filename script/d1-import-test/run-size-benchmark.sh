#!/usr/bin/env bash
# Import benchmark: 1 GB and 5 GB from local SQL exports, 9 GB from existing export.
# Parallelizes export generation and DB provisioning; imports run when both are ready.
#
# Usage:
#   ./script/d1-import-test/run-size-benchmark.sh
#   BUILD_LOCAL=false ./script/d1-import-test/run-size-benchmark.sh
#   SIZES="1 9" PARALLEL_IMPORTS=false ./script/d1-import-test/run-size-benchmark.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI="$(cd "$ROOT/../.." && pwd)"
MONOREPO_ROOT="$(cd "$CLI/../.." && pwd)"
LOG="${D1_BENCHMARK_LOG:-/tmp/d1-size-benchmark.log}"
STATE_DIR="${BENCH_STATE_DIR:-/tmp/d1-bench-$(date +%Y%m%d-%H%M%S)}"
BUILD_LOCAL="${BUILD_LOCAL:-true}"
PARALLEL_IMPORTS="${PARALLEL_IMPORTS:-true}"
SIZES=(${SIZES:-1 5 9})

mkdir -p "$STATE_DIR"
exec > >(tee -a "$LOG") 2>&1

echo "=============================================="
echo "D1 import size benchmark $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "Log:        $LOG"
echo "State dir:  $STATE_DIR"
echo "Sizes:      ${SIZES[*]}"
echo "=============================================="

if [[ ! -x "$CLI/pscale-test" ]]; then
  echo "==> Building pscale-test"
  (cd "$CLI" && go build -o pscale-test ./cmd/pscale)
fi

export PSCALE_DISABLE_DEV_WARNING=true
export PSCALE_TEST_MODE=1
export PSCALE_ORG="${PSCALE_ORG:-bb}"

PSCALE="${CLI}/pscale-test"
API_URL="${PSCALE_API_URL:-http://api.pscaledev.com:3000/v1}"
if ! "$PSCALE" --api-url "$API_URL" auth check >/dev/null 2>&1; then
  echo "ERROR: pscale auth required" >&2
  exit 1
fi
if ! curl -sS --connect-timeout 2 -o /dev/null http://127.0.0.1:8080/ 2>/dev/null; then
  echo "ERROR: singularity not responding on :8080" >&2
  exit 1
fi

bench_ts="$(basename "$STATE_DIR" | sed 's/^d1-bench-//')"
declare -A EXPORT_FILE DB_NAME BASE_DB

for size in "${SIZES[@]}"; do
  BASE_DB["$size"]="cf-d1-import-${size}gb"
  DB_NAME["$size"]="${BASE_DB[$size]}-${bench_ts}"
  echo "${DB_NAME[$size]}" > "$STATE_DIR/db-${size}gb.name"

  if [[ "$size" == "9" ]]; then
    EXPORT_FILE["$size"]="${D1_EXPORT_9GB:-/tmp/import-test-9gb-export.sql}"
    if [[ ! -f "${EXPORT_FILE[$size]}" ]]; then
      echo "ERROR: 9 GB export not found: ${EXPORT_FILE[$size]}" >&2
      exit 1
    fi
    touch "$STATE_DIR/export-9gb.ready"
  else
    EXPORT_FILE["$size"]="/tmp/import-test-${size}gb-export.sql"
  fi
done

echo ""
echo "==> Phase 1: parallel export generation + DB provisioning"
pids=()

for size in "${SIZES[@]}"; do
  if [[ "$size" != "9" && "$BUILD_LOCAL" == "true" ]]; then
    if [[ -f "${EXPORT_FILE[$size]}" && "${REGENERATE_EXPORTS:-true}" != "true" ]]; then
      echo "==> [${size}gb] Reusing export ${EXPORT_FILE[$size]}"
      touch "$STATE_DIR/export-${size}gb.ready"
    else
      (
        export SEED_DIR="$ROOT/seed/local-${size}gb"
        export D1_EXPORT="${EXPORT_FILE[$size]}"
        export EXPORT_READY_FILE="$STATE_DIR/export-${size}gb.ready"
        "$ROOT/build-local-export.sh" "$size"
      ) > "$STATE_DIR/export-${size}gb.log" 2>&1 &
      pids+=("$!")
      echo "==> [${size}gb] Export build started (pid $!)"
    fi
  elif [[ "$size" != "9" && -f "${EXPORT_FILE[$size]}" ]]; then
    touch "$STATE_DIR/export-${size}gb.ready"
  fi

  (
    PROVISION_READY_FILE="$STATE_DIR/provision-${size}gb.ready" \
      "$ROOT/provision-database.sh" "${DB_NAME[$size]}"
  ) > "$STATE_DIR/provision-${size}gb.log" 2>&1 &
  pids+=("$!")
  echo "==> [${size}gb] DB provision started: ${DB_NAME[$size]} (pid $!)"
done

echo "==> Phase 1 jobs launched; starting import watcher (exports + branch readiness)"
export BENCH_STATE_DIR="$STATE_DIR"
export BENCH_WATCH_LOG="$STATE_DIR/watch.log"
export SIZES="${SIZES[*]}"
chmod +x "$ROOT/bench-watch-imports.sh"
"$ROOT/bench-watch-imports.sh"
exit $?
