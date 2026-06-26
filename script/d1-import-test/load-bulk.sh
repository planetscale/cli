#!/usr/bin/env bash
# Load import-test D1 via merged SQL chunks (~50 MB each) instead of one wrangler call
# per seed batch. Use for multi-GB seeds; keep load.sh for quick smoke tests.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DB="${D1_DATABASE:-import-test}"
REMOTE="${D1_REMOTE:-true}"
FRESH="${D1_FRESH:-false}"
CHUNK_TARGET_MB="${CHUNK_TARGET_MB:-50}"

remote_flag=()
if [[ "$REMOTE" == "true" ]]; then
  remote_flag=(--remote)
fi

if [[ "$FRESH" == "true" ]]; then
  echo "==> Resetting existing tables"
  wrangler d1 execute "$DB" "${remote_flag[@]}" --file="$ROOT/reset.sql"
  echo "==> Applying schema to D1 database: $DB"
  wrangler d1 execute "$DB" "${remote_flag[@]}" --file="$ROOT/schema.sql"
else
  echo "==> Skipping reset/schema (set D1_FRESH=true for clean load)"
fi

if [[ "${SKIP_SEED_GENERATE:-false}" != "true" ]]; then
  echo "==> Generating seed SQL batches"
  python3 "$ROOT/generate_seed.py"
else
  echo "==> Skipping seed generation (SKIP_SEED_GENERATE=true)"
fi

if [[ "${SKIP_MERGE:-false}" != "true" ]]; then
  echo "==> Merging seed batches into ~${CHUNK_TARGET_MB} MB chunks"
  CHUNK_TARGET_MB="$CHUNK_TARGET_MB" python3 "$ROOT/merge_seed_chunks.py"
else
  echo "==> Skipping merge (SKIP_MERGE=true)"
fi

mapfile -t chunks < <(find "$ROOT/seed/chunks" -name 'chunk_*.sql' | sort)
total_chunks="${#chunks[@]}"
if [[ "$total_chunks" -eq 0 ]]; then
  echo "ERROR: no chunk files under $ROOT/seed/chunks" >&2
  exit 1
fi

echo "==> Loading ${total_chunks} chunk(s) to D1 (sequential — D1 is single-threaded per DB)"
chunk_num=0
for chunk in "${chunks[@]}"; do
  chunk_num=$((chunk_num + 1))
  size_bytes=$(stat -f%z "$chunk" 2>/dev/null || stat -c%s "$chunk")
  size_mb=$(python3 -c "print(round($size_bytes / (1024*1024), 1))")
  echo "    -> [${chunk_num}/${total_chunks}] $(basename "$chunk") (${size_mb} MB)"
  wrangler d1 execute "$DB" "${remote_flag[@]}" --file="$chunk"
done

echo "==> Done. Export with:"
echo "    wrangler d1 export $DB --remote --output ./import-test-export.sql"
echo "    pscale import d1 lint --input ./import-test-export.sql --format json"
