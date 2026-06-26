#!/usr/bin/env bash
# D1 9 GB dataset prep (load + export). CLI import is run separately via run-cli-import.sh.
#
# Usage:
#   ./script/d1-import-test/run-9gb-benchmark.sh              # load + export only
#   RUN_CLI_IMPORT=true ./script/d1-import-test/run-9gb-benchmark.sh  # then CLI import
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXPORT="${D1_EXPORT:-/tmp/import-test-9gb-export.sql}"
LOG="${D1_BENCHMARK_LOG:-/tmp/d1-9gb-benchmark.log}"

exec > >(tee -a "$LOG") 2>&1

echo "=============================================="
echo "D1 9GB benchmark started $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "Log: $LOG"
echo "=============================================="

load_start=$(date +%s)
D1_FRESH=true SEED_TARGET_GB=9 SKIP_SEED_GENERATE=true SKIP_MERGE=true \
  "$ROOT/load-bulk.sh"
load_end=$(date +%s)
load_sec=$((load_end - load_start))

echo ""
echo "==> Load phase complete: ${load_sec}s ($(python3 -c "print(round($load_sec/60,1))") min)"
wrangler d1 list | grep import-test || wrangler d1 list

echo ""
echo "==> Export benchmark"
export_start=$(date +%s)
D1_EXPORT_OUTPUT="$EXPORT" "$ROOT/time-export.sh"
export_end=$(date +%s)
export_sec=$((export_end - export_start))

echo ""
echo "=============================================="
echo "D1 prep summary $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "  Load (upload chunks):  ${load_sec}s"
echo "  Export (remote→local): ${export_sec}s"
echo "  Export file:           $EXPORT"
echo "=============================================="

if [[ "${RUN_CLI_IMPORT:-false}" == "true" ]]; then
  echo ""
  echo "==> Running CLI import test"
  IMPORT_PROFILE=9gb D1_EXPORT="$EXPORT" SKIP_DB_CREATE="${SKIP_DB_CREATE:-true}" \
    "$ROOT/run-local-import.sh"
fi
