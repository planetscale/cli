#!/usr/bin/env bash
# Watch a benchmark state dir and start imports as soon as export + DB are ready.
#
# Usage:
#   BENCH_STATE_DIR=/tmp/d1-bench-20260625-204823 ./script/d1-import-test/bench-watch-imports.sh
#   SIZES="5 9" BENCH_STATE_DIR=... ./script/d1-import-test/bench-watch-imports.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI="$(cd "$ROOT/../.." && pwd)"
STATE_DIR="${BENCH_STATE_DIR:-}"
POLL_SEC="${POLL_SEC:-15}"
SIZES=(${SIZES:-1 5 9})
LOG="${BENCH_WATCH_LOG:-${STATE_DIR}/watch.log}"

if [[ -z "$STATE_DIR" || ! -d "$STATE_DIR" ]]; then
  echo "ERROR: BENCH_STATE_DIR must point to an existing state directory" >&2
  exit 1
fi

export PSCALE_DISABLE_DEV_WARNING=true
export PSCALE_TEST_MODE=1
ORG="${PSCALE_ORG:-bb}"
export PSCALE_ORG="$ORG"

PSCALE="${CLI}/pscale-test"
API_URL="${PSCALE_API_URL:-http://api.pscaledev.com:3000/v1}"
BRANCH="${PSCALE_BRANCH:-main}"

mkdir -p "$STATE_DIR"
exec >> "$LOG" 2>&1

echo "=============================================="
echo "Import watcher started $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "State: $STATE_DIR"
echo "Sizes: ${SIZES[*]}"
echo "Poll:  ${POLL_SEC}s"
echo "=============================================="

export_file_for() {
  local size="$1"
  case "$size" in
    9) echo "${D1_EXPORT_9GB:-/tmp/import-test-9gb-export.sql}" ;;
    *) echo "/tmp/import-test-${size}gb-export.sql" ;;
  esac
}

db_name_for() {
  local size="$1"
  cat "$STATE_DIR/db-${size}gb.name"
}

branch_ready() {
  local db="$1"
  local ready
  ready="$("$PSCALE" --api-url "$API_URL" branch show "$db" "$BRANCH" --format json --org "$ORG" 2>/dev/null | python3 -c "import json,sys; print(json.load(sys.stdin).get('ready', False))" 2>/dev/null || echo False)"
  [[ "$ready" == "True" ]]
}

export_ready() {
  local size="$1"
  [[ -f "$STATE_DIR/export-${size}gb.ready" ]]
}

import_done() {
  local size="$1"
  [[ -f "$STATE_DIR/import-${size}gb.done" ]]
}

import_failed() {
  local size="$1"
  [[ -f "$STATE_DIR/import-${size}gb.failed" ]]
}

import_running() {
  local size="$1"
  local pid_file="$STATE_DIR/import-${size}gb.pid"
  if [[ ! -f "$pid_file" ]]; then
    return 1
  fi
  local pid
  pid="$(cat "$pid_file")"
  kill -0 "$pid" 2>/dev/null
}

start_import() {
  local size="$1"
  local db export run_dir profile log pid

  db="$(db_name_for "$size")"
  export="$(export_file_for "$size")"
  profile="${size}gb"
  run_dir="$STATE_DIR/import-${size}gb-$(date +%Y%m%d-%H%M%S)"
  log="$STATE_DIR/import-${size}gb.log"

  if [[ ! -f "$export" ]]; then
    echo "==> [watch ${size}gb] export file missing: $export"
    return 1
  fi

  echo "==> [watch ${size}gb] Starting import db=$db export=$export"
  touch "$STATE_DIR/import-${size}gb.started"

  (
    set -euo pipefail
    start=$(date +%s)
    if ! IMPORT_PROFILE="$profile" \
      D1_EXPORT="$export" \
      PSCALE_DB="$db" \
      IMPORT_RUN_DIR="$run_dir" \
      "$ROOT/run-cli-import.sh"; then
      touch "$STATE_DIR/import-${size}gb.failed"
      echo "==> [watch ${size}gb] Import FAILED"
      exit 1
    fi
    end=$(date +%s)
    wall=$((end - start))
    echo "${size}gb:${wall}s:${run_dir}" >> "$STATE_DIR/results.txt"
    touch "$STATE_DIR/import-${size}gb.done"
    echo "==> [watch ${size}gb] Import complete: ${wall}s"
  ) > "$log" 2>&1 &
  pid=$!
  echo "$pid" > "$STATE_DIR/import-${size}gb.pid"
}

remaining=0
for size in "${SIZES[@]}"; do
  if ! import_done "$size" && ! import_failed "$size"; then
    remaining=$((remaining + 1))
  fi
done

while (( remaining > 0 )); do
  for size in "${SIZES[@]}"; do
    if import_done "$size" || import_failed "$size" || import_running "$size"; then
      continue
    fi

    db="$(db_name_for "$size")"
    if export_ready "$size" && branch_ready "$db"; then
      if start_import "$size"; then
        :
      else
        touch "$STATE_DIR/import-${size}gb.failed"
        remaining=$((remaining - 1))
      fi
    else
      exp="no"; br="no"
      export_ready "$size" && exp="yes"
      branch_ready "$db" && br="yes"
      echo "==> [watch ${size}gb] waiting (export=$exp branch=$br db=$db)"
    fi
  done

  remaining=0
  for size in "${SIZES[@]}"; do
    if import_done "$size"; then
      continue
    fi
    if import_failed "$size"; then
      continue
    fi
    if import_running "$size"; then
      remaining=$((remaining + 1))
      continue
    fi
    remaining=$((remaining + 1))
  done

  if (( remaining > 0 )); then
    sleep "$POLL_SEC"
  fi
done

echo ""
echo "==> Waiting for running imports to finish"
wait_fail=0
for size in "${SIZES[@]}"; do
  pid_file="$STATE_DIR/import-${size}gb.pid"
  [[ -f "$pid_file" ]] || continue
  pid="$(cat "$pid_file")"
  if kill -0 "$pid" 2>/dev/null; then
    if ! wait "$pid"; then
      touch "$STATE_DIR/import-${size}gb.failed"
      wait_fail=1
    fi
  fi
done

echo ""
echo "=============================================="
echo "Import watcher finished $(date -u +%Y-%m-%dT%H:%M:%SZ)"
if [[ -f "$STATE_DIR/results.txt" ]]; then
  cat "$STATE_DIR/results.txt"
fi
if [[ "$wait_fail" -ne 0 ]]; then
  echo "ERROR: one or more imports failed" >&2
  exit 1
fi
echo "=============================================="
