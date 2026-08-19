# PlanetScale CLI — agent guide

> **Developing the CLI?** The API client is vendored at `internal/planetscale/`;
> this repo **no longer depends on `planetscale-go`**. Read `doc/api-client.md`
> before touching API-facing code. The rest of this file is about *using* `pscale`.

For **any** automated agent or script using `pscale`. Always pass **`--format json`**. Substitute placeholders from the user's request or from prior command output (`org list`, `database list`, `branch list`).

If you only have the installed `pscale` binary, start here:

```bash
pscale agent-guide --format json
pscale auth check --format json
```

Use direct CLI automation for shell commands and scripts. Use the hosted PlanetScale MCP server for MCP clients.

This file documents **how to invoke `pscale`**. For database assessment, safety review, and operational workflows, install the [PlanetScale skills pack](https://github.com/planetscale/skills) (`14-pscale-cli-automation` covers CLI automation; `00-safe-orchestrator` runs the full review). In application repositories, add a separate **project** `AGENTS.md` with org, database, branch, and approval rules (see skill `09-mcp-agent-operating-model` in that repo).

| Placeholder | Meaning |
|-------------|---------|
| `<org>` | Organization name |
| `<database>` | Database name |
| `<branch>` | Branch name (pick one with `"ready": true` from `branch list`) |

## Flag placement

- **`--org`** is a flag on resource subcommands (`database`, `branch`, `sql`, `api`, …). It is **not** on root `pscale` — `pscale --org …` fails.
- **`--format json`** is a global flag. It can go on `pscale` or on the subcommand.
- Commands with positional args (`sql`, `branch list`, …): put **positionals first**, then flags.

```bash
# Correct
pscale auth check --format json
pscale org list --format json
pscale database list --org <org> --format json
pscale branch list <database> --org <org> --format json
pscale sql <database> <branch> --org <org> --format json --query "SELECT 1"

# Also valid — global --format
pscale --format json database list --org <org>

# Wrong — unknown flag: --org
pscale --org <org> database list --format json
```

## Workflow

1. **Guide** — discover machine-readable conventions when you do not have this file:

   ```bash
   pscale agent-guide --format json
   ```

2. **Auth** — check before anything else:

   ```bash
   pscale auth check --format json
   ```

   `"status": "ok"` and `"authenticated": true` with no blocking `issues` means proceed. `"status": "action_required"` exits non-zero — log in, pick an org, or fix credentials (see `issues` and `next_steps`).

3. **Login** (when not authenticated):

   ```bash
   pscale auth login --format json
   ```

   Pending JSON is written to **stderr** while waiting; **stdout** has a single final JSON object when login completes (`status: ok` or `action_required` if org setup fails after credentials are saved). Fields include `verification_url`, `user_code`, and `browser_opened`. Open `verification_url` manually if the browser does not open. Do not retry login in a loop without browser access.

4. **Organization** — use `"organization"` from `auth check`, ask the user, or list orgs:

   ```bash
   pscale org list --format json
   ```

   Pass `--org <org>` on resource commands (`database`, `branch`, `sql`, `api`, …). Not on `org list`.

5. **Discover resources** before SQL:

   ```bash
   pscale database list --org <org> --format json
   pscale branch list <database> --org <org> --format json
   ```

6. **Query** (read-only default):

   ```bash
   pscale sql <database> <branch> --org <org> --format json --query "SELECT 1"
   ```

## Cloudflare-billed databases

To create a database billed to a Cloudflare account, Cloudflare must mint an HMAC billing proof. Pass the JSON proof directly:

```bash
pscale database create <database> --org <org> --format json \
  --cloudflare-billing '{"account_id":"<cloudflare_account_id>","timestamp":"<unix_timestamp>","signature":"<hmac_hex>"}'
```

For automation, pass `@-` to read the proof from stdin so the signature is not exposed in process arguments:

```bash
printf '%s' "$CLOUDFLARE_BILLING_JSON" | \
  pscale database create <database> --org <org> --format json --cloudflare-billing @-
```

The CLI does not mint the signature. The JSON object must contain non-empty `account_id`, `timestamp`, and `signature` strings.

## Flags

| Flag | Purpose |
|------|---------|
| `--format json` | JSON on stdout |
| `--org <org>` | Organization (on resource subcommands only) |
| `--api-url` | Non-production API base URL — pass on every command when not using production |
| `--cloudflare-billing` | Cloudflare billing proof JSON on `database create`; pass `@-` to read it from stdin |

## JSON errors

With `--format json`, any command that fails prints exactly one JSON envelope on **stdout**:

```json
{
  "status": "error",
  "error": "human-readable message",
  "issues": [{ "code": "NOT_FOUND", "message": "human-readable message" }],
  "next_steps": ["pscale org list --format json", "pscale database list --org <org> --format json"]
}
```

- `status` is `"error"` or `"action_required"`. `action_required` means an agent can recover by following `next_steps` (log in, ask the user for approval, fix the invocation). Exit code is `1` for `action_required` and `2` for `error`.
- `issues[].code` is stable and machine-readable; branch on it, not on message text.
- `next_steps` are concrete commands or instructions, ordered by likelihood.

Some commands add fields to this envelope (for example `query_kind` on destructive SQL or `migration_id` on imports) but `status`, `issues`, and `next_steps` are always present on failure.

| Code | Meaning |
|------|---------|
| `NO_AUTH` | Not authenticated or token expired; run `pscale auth login --format json` |
| `AUTH_INVALID` | Stored credentials rejected by the API; log in again |
| `SERVICE_TOKEN_INVALID` | Service token id/secret rejected; verify the values |
| `NO_ORG` | Authenticated but no organization configured |
| `INVALID_FLAG_PLACEMENT` | `--org` was passed on `pscale` root; move it to the subcommand |
| `INVALID_USAGE` | Missing arguments or required flags; the message names them |
| `UNKNOWN_COMMAND` | Command does not exist; check `pscale --help` |
| `UNKNOWN_FLAG` | Flag does not exist on this command; check `--help` |
| `TTY_REQUIRED` | Command needs an interactive terminal; use the JSON alternative in `next_steps` |
| `CONFIRMATION_REQUIRED` | Destructive or gated action; ask the user, then re-run with `--force` |
| `DESTRUCTIVE_SQL` | Query would delete data or schema; ask the user, then re-run with `--force` |
| `NOT_FOUND` | Org, database, branch, or resource does not exist; run the discovery commands |
| `NETWORK_ERROR` | Transport-level failure; check connectivity and `--api-url`, then retry |
| `COMMAND_FAILED` | Unclassified failure; read `error` and rule out auth first |

## Authentication

`pscale auth login` stores credentials in the OS keychain; agents on the same machine reuse them.

Headless / CI: pass `--service-token-id` and `--service-token` on the subcommand that needs auth.

## SQL

Non-interactive queries. Default **`--role` is `reader`** (unlike `pscale shell`, which defaults to admin). Use `pscale shell` for interactive sessions.

```bash
# Read (default)
pscale sql <database> <branch> --org <org> --format json --query "SELECT 1"

# Read from replica
pscale sql <database> <branch> --org <org> --format json --replica --query "SELECT 1"

# PostgreSQL — optional --dbname (default postgres)
pscale sql <database> <branch> --org <org> --format json --query "SELECT 1"

# MySQL multi-keyspace — optional --keyspace (default @primary)
pscale sql <database> <branch> --org <org> --format json --keyspace <keyspace> --query "SELECT 1"
```

| Flag | Purpose |
|------|---------|
| `--role` | `reader` (default), `writer`, `readwriter`, `admin` — same names as `pscale shell` |
| `--replica` | Route reads to replicas |
| `--dbname` | PostgreSQL database name (default `postgres`) |
| `--keyspace` | MySQL keyspace (default `@primary`); may include a shard and tablet type: `mykeyspace/-80`, `mykeyspace/-80@replica` |
| `--force` | Allow destructive SQL after explicit user approval |

**`--role` by engine** (same as `pscale shell`):

| `--role` | MySQL (Vitess) | PostgreSQL |
|----------|----------------|------------|
| `reader` | Branch password, reader role | Ephemeral role inheriting `pg_read_all_data` |
| `writer` | Branch password, writer role | Role inheriting `pg_write_all_data` |
| `readwriter` | Branch password, readwriter role | Role inheriting read + write |
| `admin` | Branch password, admin role | Role inheriting `postgres` |

### Destructive SQL

`DELETE`, `DROP`, and `TRUNCATE` anywhere in a query are blocked by default (word match, not substring — `deleted_at` is fine). Returns `"status": "action_required"` with `"query_kind": "destructive"`.

1. Ask the user to approve the query.
2. Re-run with `--force` only after they approve:

```bash
pscale sql <database> <branch> --org <org> --format json --force --query "DELETE FROM ..."
```

Never use `--force` without explicit user approval.

### SQL JSON

Success: `status`, `database`, `branch`, `kind` (`mysql` or `postgresql`), `role`, `row_count`, `columns`, `rows`; `replica` when `--replica` was used.

MySQL may return synthetic column names (e.g. `:vtg1 /* INT64 */`). PostgreSQL may use names like `?column?`.

Error: one JSON object on stdout with `status: "error"`, `error`, `issues`, and `next_steps` (see JSON errors above).

Destructive SQL without `--force`: `status: "action_required"`, `query_kind: "destructive"`, `issues`, and `next_steps` (includes `--force` retry command).

## Metrics

Query historical or current branch metrics through the public metrics API:

```bash
pscale metrics show <database> <branch> --org <org> --format json --metric queries --metric latency_p99 --period 1h
pscale metrics instant <database> <branch> --org <org> --format json --metric planetscale_volume_usage_percentage
pscale metrics report <database> <branch> --org <org> --format json --period 1d
```

- For `metrics show` and `metrics instant`, `--metric` is required and may be repeated or comma-separated.
- `metrics report` detects whether the database uses MySQL or PostgreSQL and queries a curated set of performance sections. It supports `--period`, custom `--from`/`--to` ranges, and `--steps`; JSON returns a composite report and CSV includes the section name on each row.
- Historical queries support `--period`, or a custom `--from`/`--to` ISO 8601 range, plus `--steps` and dimension filters such as `--tablet-type`, `--keyspace`, `--shard`, `--role`, `--pod`, and `--pods`.
- JSON preserves the API response: historical results contain `start_date`, `end_date`, `interval`, and `series`; each series contains `metric`, `label`, `labels`, and `[Unix timestamp, value]` points. Instant results contain current values grouped by their dimensions.
- Human output summarizes each historical series with latest/min/average/max values and a sparkline. CSV flattens historical samples or instant values to one row each.

## Diagnostics: insights + inspect

Two complementary read-only surfaces. When diagnosing database health or performance, **check both** — they see different things.

**`pscale insights`** — server-side analysis computed from production traffic (works even when you can't or don't want to connect to the database):

```bash
pscale insights queries <database> <branch> --org <org> --format json --sort totalTime   # top queries; sorts: totalTime, count, p99Latency, rowsRead, rowsReadPerReturned, errorCount, ...
pscale insights queries samples <database> <branch> <fingerprint> --org <org> --format json --keyspace <keyspace>  # recent executions; keyspace from queries list
pscale insights errors <database> <branch> --org <org> --format json                     # failing queries with error messages
pscale insights errors show <database> <branch> <fingerprint> --org <org> --format json  # individual queries behind one error fingerprint
pscale insights anomalies <database> <branch> --org <org> --format json                  # detected resource anomalies (CPU, memory, IOPS, rows)
pscale insights anomalies show <database> <branch> <id> --org <org> --format json        # one anomaly plus its correlated queries
pscale insights tags <database> <branch> --org <org> --format json                       # query tag keys (sqlcommenter / system); use names with summaries
pscale insights tags summaries <database> <branch> --org <org> --format json --tags username  # stats grouped by tag; names match the Insights UI Key picker
pscale insights recommendations <database> --org <org> --format json                     # schema recommendations with ready-to-apply DDL
pscale insights recommendations dismiss <database> <number> --org <org> --format json --force  # dismiss a recommendation
```

**`pscale inspect`** — live, point-in-time checks run over a direct connection (same credentials model as `pscale sql`, always read-only):

```bash
pscale inspect all <database> <branch> --org <org> --format json    # every applicable check, one report
pscale inspect <check> <database> <branch> --org <org> --format json
```

Checks: `table-sizes`, `index-sizes`, `unused-indexes`, `redundant-indexes`, `seq-scans`, `long-running-queries`, `locks`, `outliers`, `calls`, `bloat`, `vacuum-stats`, `replication-slots`, `subscriptions`. Checks adapt per engine; ones that don't apply explain the alternative. JSON results include `next_steps` pointing at the matching `insights` command — follow them.

Caveats:
- Statistics are since last server restart and per-connection-target: on sharded Vitess databases they reflect a single shard's MySQL instance. Use `--keyspace` to pick a keyspace, or pin an exact shard with `--keyspace 'mykeyspace/-80'` (enumerate with `pscale sql <database> <branch> --org <org> --format json --query "SHOW VITESS_SHARDS"`; rows are `keyspace/shard`). Databases can have hundreds of shards — inspect one shard at a time rather than fanning out. On PostgreSQL, stats are scoped to one database (use `--dbname`; if CONNECT is denied, retry with `--role admin`).
- `outliers`/`calls` need `pg_stat_statements` on PostgreSQL; if missing, use `pscale insights queries` instead (no extension needed).
- Rule of thumb: start with `insights` (traffic-aware, historical), use `inspect` for live state (locks, in-flight queries) and physical layout (sizes, bloat, index usage).

## Vitess database throttler

Database-level default for future deploy request migrations (not per-DR, not tablet/vtctld):

```bash
pscale database throttler show <database> --org <org> --format json
pscale database throttler update <database> --org <org> --format json --ratio 25
pscale database throttler update <database> --org <org> --format json \
  --configuration main=10 --configuration sharded=40
```

`--ratio` is 0–95 (0 disables throttling; 95 is slowest). Use either `--ratio` or `--configuration keyspace=ratio`, not both. Vitess only.

## Vitess deploy requests (inspect + throttler)

Core lifecycle is already covered (`list/create/show/diff/review/deploy/apply/edit/cancel/close/revert/skip-revert`). These inspect commands are read-only:

```bash
pscale deploy-request queue <database> --org <org> --format json                         # database deploy queue (first page)
pscale deploy-request operations <database> <number> --org <org> --format json           # per-table schema ops + progress
pscale deploy-request reviews <database> <number> --org <org> --format json              # existing reviews (create with review)
pscale deploy-request deployment <database> <number> --org <org> --format json           # deployment detail (cutover flags, queue state)
pscale deploy-request storage-check <database> <number> --org <org> --format json        # enough_storage / bytes needed
pscale deploy-request throttler show <database> <number> --org <org> --format json       # per-DR throttler ratios (not database throttler)
```

Throttler update mutates the deploy request (use after `throttler show`):

```bash
pscale deploy-request throttler update <database> <number> --org <org> --format json --ratio 25
pscale deploy-request throttler update <database> <number> --org <org> --format json \
  --configuration main=10 --configuration sharded=40
```

Alias: `pscale dr …` works the same. Vitess only. `--ratio` is 0–95 (0 disables throttling; 95 is slowest). Use either `--ratio` or `--configuration keyspace=ratio`, not both.

## Maintenance schedules (Vitess Enterprise)

Read-only visibility into planned maintenance windows for a Vitess database (Enterprise plans):

```bash
pscale maintenance list <database> --org <org> --format json
pscale maintenance show <database> <schedule-id> --org <org> --format json
pscale maintenance windows <database> <schedule-id> --org <org> --format json
```

`--org` is required. Schedules include next/last window times, frequency, and any pending Vitess/MySQL version updates. Vitess Enterprise only.

## Postgres branch changes (size, replicas, parameters)

`pscale branch resize` queues a single asynchronous **change request** for a Postgres branch covering cluster size, replica count, and configuration parameters in any combination. Track it with `resize status`; cancel it with `resize cancel` while queued.

```bash
# Read the parameter catalog first (names, current/default values, restart/immutable flags)
pscale branch parameters list <database> <branch> --org <org> --format json
pscale branch parameters list <database> <branch> --org <org> --format json --namespace pgconf

# Change parameters (repeat --parameters; keys are namespace.name)
pscale branch resize <database> <branch> --org <org> --format json --parameters pgconf.max_connections=200

# Combine size, replicas, and parameters into one change request
pscale branch resize <database> <branch> --org <org> --format json \
  --cluster-size PS_10_GCP_X86 --replicas 2 --parameters pgconf.max_connections=500

# Block until the change finishes (default timeout 10m; tune with --wait-timeout)
pscale branch resize <database> <branch> --org <org> --format json --parameters pgconf.work_mem=64MB --wait

# Inspect the latest change request / cancel a queued one
pscale branch resize status <database> <branch> --org <org> --format json
pscale branch resize cancel <database> <branch> --org <org> --format json
```

- At least one of `--cluster-size`, `--replicas`, or `--parameters` is required.
- `--parameters` values are validated against the catalog before submission; unknown or immutable parameters fail fast. Parameters with `"restart": true` in the catalog restart the database when applied — surface this to the user before changing them.
- Change request `state` is one of `queued`, `pending`, `resizing`, `completed`, `canceled`. Only `completed` and `canceled` are terminal. Without `--wait`, poll `resize status` instead of assuming completion.
- A no-op (branch already matches the requested configuration) prints `{"result": "no_change", "branch": "<branch>"}` in JSON mode instead of a change request.
- `resize cancel` prints `{"result": "canceled", "branch": "<branch>"}` in JSON mode.
- MySQL databases are rejected: use `pscale keyspace resize` for Vitess keyspaces.

## Imports (Cloudflare D1)

`pscale import d1` migrates a Cloudflare D1 (SQLite) export into a PlanetScale Postgres branch. Every subcommand supports `--format json` and returns `status`, `issues`, and `next_steps`; stateful steps return a `migration_id` — pass it back with `--migration-id` to resume.

```bash
pscale import d1 doctor --format json                              # check prerequisites (pgloader, psql)
pscale import d1 lint --input <file> --format json                 # pre-import checks; errors block import
pscale import d1 start <database> --org <org> --input <file> --dry-run --format json  # plan + migration ID, no writes
pscale import d1 start <database> --org <org> --input <file> --format json            # run the import
pscale import d1 verify <database> --org <org> --migration-id <id> --sqlite <file> --format json
pscale import d1 complete <database> --org <org> --migration-id <id> --format json
```

Branch is an optional second positional (defaults to the default branch). `status --migration-id <id>` shows saved migration state; `convert-schema --input <file>` converts schema only.

**`start` in JSON mode does not prompt.** The confirmation prompt is human-format only; with `--format json`, `start` loads data immediately. Run `start --dry-run` first, show the user the plan, and only run the real `start` after they approve.

## API passthrough

```bash
pscale api --org <org> organizations/<org>/databases
```

## MCP

For MCP clients, use the hosted PlanetScale MCP server:

```text
https://mcp.pscale.dev/mcp/planetscale
```

See the current MCP docs: https://planetscale.com/docs/connect/mcp

Do not use the deprecated local `pscale mcp server` path unless you explicitly need backward compatibility with an old setup.

## PlanetScale agent skills

Operational workflows (inventory, safety review, Insights, schema recommendations, Traffic Control) live in the public skills repo — not in this file.

```bash
git clone https://github.com/planetscale/skills.git && cd skills && script/setup
# or: npx skills add planetscale/skills -g -y
```

After installing skills, load `14-pscale-cli-automation` for CLI conventions (or re-run `pscale agent-guide --format json` from any `pscale` binary that includes agent onboarding). Use `00-safe-orchestrator` when the user asks for a full PlanetScale assessment.
