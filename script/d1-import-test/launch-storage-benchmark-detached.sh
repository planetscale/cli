#!/usr/bin/env bash
# Run full storage benchmark in a detached session (survives Cursor shell teardown).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI="$(cd "$ROOT/../.." && pwd)"
STATE_DIR="${BENCH_STATE_DIR:-/tmp/d1-bench-$(date +%Y%m%d-%H%M%S)}"
LOG="${D1_BENCHMARK_LOG:-/tmp/d1-storage-benchmark.log}"

mkdir -p "$STATE_DIR"

python3 - "$ROOT" "$CLI" "$STATE_DIR" "$LOG" <<'PY'
import os, subprocess, sys, time
from pathlib import Path

root, cli, state_dir, log = sys.argv[1:5]
state = Path(state_dir)
env = os.environ.copy()
env["BENCH_STATE_DIR"] = state_dir
env["D1_BENCHMARK_LOG"] = log
env["PSCALE_DISABLE_DEV_WARNING"] = "true"
env["PSCALE_TEST_MODE"] = "1"

fh = open(log, "a", buffering=1)
fh.write(f"\n==> detached storage benchmark {time.strftime('%Y-%m-%dT%H:%M:%SZ')} state={state_dir}\n")
fh.flush()

proc = subprocess.Popen(
    ["bash", f"{root}/run-storage-benchmark.sh"],
    stdin=subprocess.DEVNULL,
    stdout=fh,
    stderr=subprocess.STDOUT,
    env=env,
    cwd=cli,
    start_new_session=True,
    close_fds=True,
)
(state / "benchmark.pid").write_text(str(proc.pid))
print(f"benchmark pid={proc.pid}")
print(f"state_dir={state_dir}")
print(f"log={log}")
print(f"report={state_dir}/report.txt (when complete)")
PY

chmod +x "$ROOT/run-storage-benchmark.sh" "$ROOT/collect-benchmark-results.sh" "$ROOT/bench-watch-imports.sh"
