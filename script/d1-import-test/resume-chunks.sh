#!/usr/bin/env bash
# Resume bulk chunk upload from a given chunk number (inclusive).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DB="${D1_DATABASE:-import-test}"
START="${CHUNK_START:-24}"
LOG="${D1_RESUME_LOG:-/tmp/d1-chunk-resume.log}"
EXPORT="${D1_EXPORT_OUTPUT:-/tmp/import-test-9gb-export.sql}"

remote_flag=(--remote)

exec > >(tee -a "$LOG") 2>&1

echo "==> Resume from chunk_${START} at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
wrangler d1 list | grep "$DB" || wrangler d1 list

mapfile -t chunks < <(find "$ROOT/seed/chunks" -name 'chunk_*.sql' | sort)
total="${#chunks[@]}"
done_count=0
failures=0
max_retries=3

for chunk in "${chunks[@]}"; do
  num=$(basename "$chunk" .sql | sed 's/chunk_//')
  num=$((10#$num))
  if (( num < START )); then
    continue
  fi
  done_count=$((done_count + 1))
  remaining=$((total - num + 1))
  size_mb=$(python3 -c "import os; print(round(os.path.getsize('$chunk') / (1024*1024), 1))")

  attempt=0
  while true; do
    attempt=$((attempt + 1))
    echo "    -> [${num}/${total}] $(basename "$chunk") (${size_mb} MB) attempt ${attempt}"
    if wrangler d1 execute "$DB" "${remote_flag[@]}" --file="$chunk"; then
      failures=0
      break
    fi
    failures=$((failures + 1))
    if (( attempt >= max_retries )); then
      echo "ERROR: failed chunk ${num} after ${max_retries} attempts" >&2
      exit 1
    fi
    echo "    !! retrying in 10s..."
    sleep 10
  done

  if (( num % 10 == 0 )); then
    wrangler d1 list | grep "$DB" || true
  fi
done

echo ""
echo "==> All chunks loaded at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
wrangler d1 list | grep "$DB" || wrangler d1 list

echo ""
echo "==> Timed export"
D1_EXPORT_OUTPUT="$EXPORT" "$ROOT/time-export.sh"

echo ""
echo "==> Complete. Export: $EXPORT"
