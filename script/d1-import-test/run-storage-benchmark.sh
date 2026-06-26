#!/usr/bin/env bash
# Full 1/5/9 GB storage benchmark: build exports, provision DBs, import, collect results.
# SEED_TARGET_GB = Postgres logical blob storage (sum of attachment payload bytes).
#
# Usage:
#   ./script/d1-import-test/run-storage-benchmark.sh
#   BENCH_STATE_DIR=/tmp/d1-bench-manual ./script/d1-import-test/run-storage-benchmark.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI="$(cd "$ROOT/../.." && pwd)"
STATE_DIR="${BENCH_STATE_DIR:-/tmp/d1-bench-$(date +%Y%m%d-%H%M%S)}"
LOG="${D1_BENCHMARK_LOG:-/tmp/d1-storage-benchmark.log}"
SIZES=(${SIZES:-1 5 9})
bench_ts="$(basename "$STATE_DIR" | sed 's/^d1-bench-//')"

mkdir -p "$STATE_DIR"
exec >> "$LOG" 2>&1

echo "=============================================="
echo "Storage benchmark start $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "State: $STATE_DIR"
echo "Sizes: ${SIZES[*]}"
echo "=============================================="

export PSCALE_DISABLE_DEV_WARNING=true
export PSCALE_TEST_MODE=1
export PSCALE_ORG="${PSCALE_ORG:-bb}"

if [[ ! -x "$CLI/pscale-test" ]]; then
  (cd "$CLI" && go build -o pscale-test ./cmd/pscale)
fi

PSCALE="${CLI}/pscale-test"
API_URL="${PSCALE_API_URL:-http://api.pscaledev.com:3000/v1}"
if ! "$PSCALE" --api-url "$API_URL" auth check >/dev/null 2>&1; then
  echo "ERROR: pscale auth required" >&2
  exit 1
fi

declare -A EXPORT_FILE DB_NAME
for size in "${SIZES[@]}"; do
  DB_NAME["$size"]="cf-d1-import-${size}gb-${bench_ts}"
  echo "${DB_NAME[$size]}" > "$STATE_DIR/db-${size}gb.name"
  if [[ "$size" == "9" ]]; then
    EXPORT_FILE["$size"]="${D1_EXPORT_9GB:-/tmp/import-test-9gb-export.sql}"
    if [[ ! -f "${EXPORT_FILE[$size]}" ]]; then
      echo "ERROR: 9 GB export missing: ${EXPORT_FILE[$size]}" >&2
      exit 1
    fi
    touch "$STATE_DIR/export-9gb.ready"
  else
    EXPORT_FILE["$size"]="/tmp/import-test-${size}gb-export.sql"
  fi
done

echo ""
echo "==> Phase 1: build 1GB and 5GB exports (Postgres storage target)"
pids=()
for size in "${SIZES[@]}"; do
  [[ "$size" == "9" ]] && continue
  (
    export SEED_DIR="$ROOT/seed/local-${size}gb"
    export D1_EXPORT="${EXPORT_FILE[$size]}"
    export EXPORT_READY_FILE="$STATE_DIR/export-${size}gb.ready"
    export REGENERATE_SEED=true
    "$ROOT/build-local-export.sh" "$size"
  ) > "$STATE_DIR/export-${size}gb.log" 2>&1 &
  pids+=("$!")
  echo "export ${size}gb pid $!"
done

fail=0
for pid in "${pids[@]}"; do
  wait "$pid" || fail=1
done
if [[ "$fail" -ne 0 ]]; then
  echo "ERROR: export build failed; see $STATE_DIR/export-*.log" >&2
  exit 1
fi

echo ""
echo "==> Phase 2: provision Postgres databases"
pids=()
for size in "${SIZES[@]}"; do
  (
    PROVISION_READY_FILE="$STATE_DIR/provision-${size}gb.ready" \
      "$ROOT/provision-database.sh" "${DB_NAME[$size]}"
  ) > "$STATE_DIR/provision-${size}gb.log" 2>&1 &
  pids+=("$!")
  echo "provision ${size}gb: ${DB_NAME[$size]} pid $!"
done
fail=0
for pid in "${pids[@]}"; do
  wait "$pid" || fail=1
done
if [[ "$fail" -ne 0 ]]; then
  echo "ERROR: provision failed; see $STATE_DIR/provision-*.log" >&2
  exit 1
fi

echo ""
echo "==> Phase 3: imports"
export BENCH_STATE_DIR="$STATE_DIR"
export BENCH_WATCH_LOG="$STATE_DIR/watch.log"
export SIZES="${SIZES[*]}"
export POLL_SEC="${POLL_SEC:-15}"
"$ROOT/bench-watch-imports.sh"

echo ""
echo "==> Phase 4: collect results"
"$ROOT/collect-benchmark-results.sh" "$STATE_DIR"

echo ""
echo "=============================================="
echo "Storage benchmark complete $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "Report: $STATE_DIR/report.txt"
echo "=============================================="
