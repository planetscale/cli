#!/usr/bin/env bash
# Launch benchmark jobs in a new session so they survive Cursor/agent shell teardown.
#
# Usage:
#   BENCH_STATE_DIR=/tmp/d1-bench-20260625-204823 ./script/d1-import-test/launch-benchmark-detached.sh
#   START_5GB_EXPORT=true ./script/d1-import-test/launch-benchmark-detached.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI="$(cd "$ROOT/../.." && pwd)"
STATE_DIR="${BENCH_STATE_DIR:-/tmp/d1-bench-$(date +%Y%m%d-%H%M%S)}"
SIZES="${SIZES:-1 5 9}"
START_5GB_EXPORT="${START_5GB_EXPORT:-true}"

mkdir -p "$STATE_DIR"

if [[ ! -x "$CLI/pscale-test" ]]; then
  (cd "$CLI" && go build -o pscale-test ./cmd/pscale)
fi

rm -f "$STATE_DIR/import-"{1,5,9}gb.{pid,started,failed}

python3 - "$ROOT" "$CLI" "$STATE_DIR" "$SIZES" "$START_5GB_EXPORT" <<'PY'
import os, subprocess, sys, time
from pathlib import Path

root, cli, state_dir, sizes, start_5gb = sys.argv[1:6]
state = Path(state_dir)
state.mkdir(parents=True, exist_ok=True)

env = os.environ.copy()
env.update({
    "PSCALE_DISABLE_DEV_WARNING": "true",
    "PSCALE_TEST_MODE": "1",
    "PSCALE_ORG": env.get("PSCALE_ORG", "bb"),
})

def spawn(name, cmd, extra_env=None, log_name=None):
    e = env.copy()
    if extra_env:
        e.update(extra_env)
    log = state / (log_name or f"{name}.log")
    log.parent.mkdir(parents=True, exist_ok=True)
    fh = open(log, "a", buffering=1)
    fh.write(f"\n==> detached launch {name} pid-pending {time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())}\n")
    fh.flush()
    proc = subprocess.Popen(
        cmd,
        stdin=subprocess.DEVNULL,
        stdout=fh,
        stderr=subprocess.STDOUT,
        env=e,
        cwd=cli,
        start_new_session=True,
        close_fds=True,
    )
    (state / f"{name}.pid").write_text(str(proc.pid))
    print(f"started {name}: pid={proc.pid} log={log}")
    return proc.pid

pids = []

if start_5gb.lower() == "true" and not (state / "export-5gb.ready").exists():
    pids.append(spawn(
        "export-5gb",
        ["bash", f"{root}/build-local-export.sh", "5"],
        {
            "SEED_DIR": f"{root}/seed/local-5gb",
            "D1_EXPORT": "/tmp/import-test-5gb-export.sql",
            "EXPORT_READY_FILE": str(state / "export-5gb.ready"),
        },
    ))
else:
    print("skip 5gb export (ready or disabled)")

pids.append(spawn(
    "watcher",
    ["bash", f"{root}/bench-watch-imports.sh"],
    {
        "BENCH_STATE_DIR": state_dir,
        "BENCH_WATCH_LOG": str(state / "watch.log"),
        "SIZES": sizes,
        "POLL_SEC": env.get("POLL_SEC", "15"),
    },
    log_name="watch.log",
))

(state / "launcher.pids").write_text("\n".join(str(p) for p in pids) + "\n")
print(f"state_dir={state_dir}")
print("detached — safe to close Cursor agent shell")
PY

chmod +x "$ROOT/bench-watch-imports.sh" "$ROOT/build-local-export.sh"
