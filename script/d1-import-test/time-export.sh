#!/usr/bin/env bash
# Time wrangler d1 export from remote D1 to a local SQL file.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DB="${D1_DATABASE:-import-test}"
OUTPUT="${D1_EXPORT_OUTPUT:-/tmp/import-test-export.sql}"
REMOTE="${D1_REMOTE:-true}"

remote_flag=()
if [[ "$REMOTE" == "true" ]]; then
  remote_flag=(--remote)
fi

echo "==> D1 database: $DB"
echo "==> Output file:  $OUTPUT"
echo "==> Remote size (wrangler d1 list):"
wrangler d1 list 2>/dev/null | grep -E "$DB|file_size" || wrangler d1 list

echo ""
echo "==> Starting export at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
start_ns=$(python3 -c "import time; print(time.time_ns())")

wrangler d1 export "$DB" "${remote_flag[@]}" --output "$OUTPUT"

end_ns=$(python3 -c "import time; print(time.time_ns())")
elapsed_sec=$(python3 -c "print(round(($end_ns - $start_ns) / 1e9, 2))")

bytes=$(stat -f%z "$OUTPUT" 2>/dev/null || stat -c%s "$OUTPUT")
mb=$(python3 -c "print(round($bytes / (1024*1024), 2))")
gb=$(python3 -c "print(round($bytes / (1024**3), 3))")
throughput=$(python3 -c "print(round($bytes / (1024*1024) / $elapsed_sec, 2)) if $elapsed_sec > 0 else 0")

echo ""
echo "==> Finished at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "    Elapsed:     ${elapsed_sec}s"
echo "    File size:   ${mb} MB (${gb} GB)"
echo "    Throughput:  ${throughput} MB/s"
echo "    Path:        $OUTPUT"
