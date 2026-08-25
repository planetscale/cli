# PlanetScale CLI — agent guide

For **any** automated agent or script using `pscale`. Always pass **`--format json`**. Substitute placeholders from the user's request or from prior command output (`org list`, `database list`, `branch list`).

If you only have the installed `pscale` binary and this guide is not already in your context, load it and check auth:

```bash
pscale --skill
pscale auth check --format json
```

Use direct CLI automation for shell commands and scripts. Use the hosted PlanetScale MCP server for MCP clients.

This file documents **how to invoke `pscale`**. For database assessment, safety review, and operational workflows, install the [PlanetScale skills pack](https://github.com/planetscale/skills) (`14-pscale-cli-automation` covers CLI automation; `00-safe-orchestrator` runs the full review). In application repositories, add a separate **project** `AGENTS.md` with org, database, branch, and approval rules (see skill `09-mcp-agent-operating-model` in that repo).

## Public repository safety

This repository is public. Do not include internal or sensitive information in commits, commit messages, pull request titles, or pull request descriptions. Do not name or link private repositories, internal issues or pull requests, Slack conversations, customer details, credentials, or private infrastructure. Include references only when the referenced resource is public.

## Concepts

PlanetScale is a serverless database platform for **MySQL** (via Vitess) and **PostgreSQL**. Resources are namespaced: an **organization** (org) owns **databases**, and each database contains **branches** — isolated copies of schema (and, for Postgres, data) that work like git branches. The default branch is typically `main` (production). Most commands target a database + branch and take `--org` to say which organization they belong to. Throughout this guide, `<org>`, `<database>`, and `<branch>` are placeholders for those names — pick a branch with `"ready": true` from `branch list`.

On Vitess/MySQL, schema changes ship via **deploy requests**: online, non-blocking migrations you review and then deploy.

Many commands are engine-specific, and some operations use different commands per engine. Schema changes: Vitess/MySQL uses `deploy-request`; Postgres branches apply DDL directly. Access: Vitess/MySQL uses `password`; Postgres uses `role`. Resize: Vitess/MySQL uses `keyspace resize`; Postgres uses `branch resize`. Vitess/MySQL-only: `deploy-request`, `keyspace`, `workflow`, `connect`, `password`. Postgres-only: `role`, `traffic-control`, branch `switchover`/`maintenance`/`parameters`, and `import d1`. The rest (`database`, `branch`, `sql`, `shell`, `insights`, `metrics`, `backup`, `org`, `auth`, `api`) work on both.

When a database is "weird" (slow, erroring, locked, bloated):

- **`insights`** — historical analysis computed from production traffic. Start here.
- **`inspect`** — live, point-in-time state over a direct connection (locks, in-flight queries, sizes).
- **`metrics`** — time-series for latency, throughput, connections.

See "Diagnostics: insights + inspect" below for the full commands.

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

1. **Guide** — if this file is not already in your context, load the skill:

   ```bash
   pscale --skill
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

   Organization members (email or the USER_ID from list):

   ```bash
   pscale org member list --org <org> --format json
   pscale org member show user@example.com --org <org> --format json
   pscale org member update user@example.com --org <org> --format json --role member
   pscale org member remove user@example.com --org <org> --format json --force
   ```

   Only org admins can change another member's role or remove someone else. Nobody can change their own role. `--role` is `admin`, `member`, or `analyst`.

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
pscale metrics queries <database> <branch> --org <org> --format json --metric latency_p99 --query-id <query-id> --period 1h
pscale metrics tables <database> <branch> --org <org> --format json
pscale metrics keyspace-tables <database> <branch> --org <org> --format json
pscale metrics tablets <database> <branch> --org <org> --format json --metric replication_lag --period 1h
pscale metrics tablets instant <database> <branch> --org <org> --format json --metric replication_lag
pscale metrics tags <database> <branch> --org <org> --format json --metric queries --tag-set Busername=alice --tag-set Busername=bob --period 1h
```

- For `metrics show` and `metrics instant`, `--metric` is required and may be repeated or comma-separated.
- Specialized query, tablet, and tag metrics accept the filters exposed by their API endpoints. `--metric` and `--query-id` may be repeated or comma-separated. `--tag-set` is repeated for independent series; comma-separate keys inside one set (`Busername=alice,Senv=production`). Tag keys need the Insights type prefix from `insights tags`.
- `metrics tables` and `metrics keyspace-tables` preserve the untyped storage-metrics API response in JSON.
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
pscale insights errors show <database> <branch> <fingerprint> --org <org> --format json  # individual queries behind one error fingerprint (use error_fingerprint from the errors list)
pscale insights anomalies <database> <branch> --org <org> --format json                  # detected resource anomalies (CPU, memory, IOPS, rows)
pscale insights anomalies show <database> <branch> <id> --org <org> --format json        # one anomaly plus its correlated queries
pscale insights tags <database> <branch> --org <org> --format json                       # query tag keys (sqlcommenter / system); use names with summaries
pscale insights tags summaries <database> <branch> --org <org> --format json --tags username  # stats grouped by tag; names match the Insights UI Key picker
pscale insights recommendations <database> --org <org> --format json                     # schema recommendations with ready-to-apply DDL
pscale insights recommendations dismiss <database> <number> --org <org> --format json --force  # dismiss a recommendation
pscale branch query-patterns list <database> <branch> --org <org> --format json
pscale branch query-patterns show <database> <branch> <report-id> --org <org> --format json
pscale branch query-patterns delete <database> <branch> <report-id> --org <org> --format json --force
```

`branch query-patterns download` generates a new report, waits, and writes CSV. Use list/show/delete for reports that already exist.

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

## Vitess aggressive cutover

Database-level setting for future deploy requests (not the same as `deploy-request force-cutover`):

```bash
pscale database aggressive-cutover show <database> --org <org> --format json
pscale database aggressive-cutover enable <database> --org <org> --format json
pscale database aggressive-cutover disable <database> --org <org> --format json
```

Vitess only. See https://planetscale.com/docs/vitess/schema-changes/aggressive-cutover

## Vitess deploy requests (inspect + throttler)

Core lifecycle is already covered (`list/create/show/diff/review/deploy/apply/unblock/update/cancel/close/revert/skip-revert`). `update` (`edit` is an alias) sets auto-apply and auto-delete-branch. `unblock` clears the queue after a failed deploy or revert (dashboard “Unblock deploy queue”); it is not `apply`. These inspect commands are read-only:

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

```bash
pscale deploy-request update <database> <number> --org <org> --format json --enable-auto-apply
pscale deploy-request update <database> <number> --org <org> --format json --auto-delete-branch=false
```

After a failed deploy or revert (`complete_error` / `complete_revert_error`), unblock the queue. This is not `apply` (gated cutover) and it cannot fix a deploy-check `error`:

```bash
pscale deploy-request unblock <database> <number> --org <org> --format json
```


## Maintenance schedules (Vitess Enterprise)

Read-only visibility into planned maintenance windows for a Vitess database (Enterprise plans):

```bash
pscale maintenance list <database> --org <org> --format json
pscale maintenance show <database> <schedule-id> --org <org> --format json
pscale maintenance windows <database> <schedule-id> --org <org> --format json
```

`--org` is required. Schedules include next/last window times, frequency, and any pending Vitess/MySQL version updates. Vitess Enterprise only.

## Postgres branch changes (size, replicas, parameters) — Postgres only

`pscale branch resize` queues a single asynchronous **change request** for a Postgres branch covering cluster size, replica count, and configuration parameters in any combination. Track it with `resize status`; cancel it with `resize cancel` while queued.

```bash
# Read the parameter catalog first (names, current/default values, restart/immutable flags)
pscale branch parameters list <database> <branch> --org <org> --format json
pscale branch parameters list <database> <branch> --org <org> --format json --namespace pgconf

# Extensions available on the cluster image (not CREATE EXTENSION state)
pscale branch extensions list <database> <branch> --org <org> --format json

# Default postgres role (read-only; reset-default rotates the password)
pscale role default <database> <branch> --org <org> --format json
pscale role reset-default <database> <branch> --org <org> --format json --force

# Role connection details for a branch replica, regional read-only replica, or PgBouncer
pscale role get <database> <branch> <role-id> --org <org> --format json --replica
pscale role get <database> <branch> <role-id> --org <org> --format json --read-only-replica <region-slug>
pscale role get <database> <branch> <role-id> --org <org> --format json --bouncer <bouncer-name>

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
- The `role get` connection target flags are mutually exclusive. Targeted role responses keep the same shape while changing `username`, `access_host_url`, and `database_url` as needed.
- `--parameters` values are validated against the catalog before submission; unknown or immutable parameters fail fast. Parameters with `"restart": true` in the catalog restart the database when applied — surface this to the user before changing them.
- Change request `state` is one of `queued`, `pending`, `resizing`, `completed`, `canceled`. Only `completed` and `canceled` are terminal. Without `--wait`, poll `resize status` instead of assuming completion.
- A no-op (branch already matches the requested configuration) prints `{"result": "no_change", "branch": "<branch>"}` in JSON mode instead of a change request.
- `resize cancel` prints `{"result": "canceled", "branch": "<branch>"}` in JSON mode.
- MySQL databases are rejected: use `pscale keyspace resize` for Vitess keyspaces.

## Postgres switchovers

`pscale branch switchover` moves the primary of a Postgres branch to a replica. It is Postgres-only; Vitess/MySQL databases are rejected before any API call.

```bash
# Promote an automatically selected replica
pscale branch switchover <database> <branch> --org <org> --format json

# Promote a specific replica (names from `pscale branch infra`)
pscale branch switchover <database> <branch> --org <org> --format json --candidate <replica-name>
```

- The command returns the created switchover (`id`, `state`, `method`) and exits; it does not wait. A fresh switchover is `pending` and `method` is empty until the operator picks one.
- `method` is `switchover` (replica promoted) for branches with replicas, or `restart` for single-node branches, which are restarted in place and unreachable while they come back. Warn the user before running this against a single-node branch.
- Writes are briefly interrupted while the switch completes. A branch accepts one switchover at a time.
- A switchover that ends in `failed` has an unconfirmed outcome: the primary may still have moved and nothing is rolled back. Check the current primary with `pscale branch infra <database> <branch> --org <org> --format json` before retrying.
- `--candidate` is rejected for branches without replicas. Poll status with `pscale api organizations/<org>/databases/<database>/branches/<branch>/switchovers/<id>`.

## Postgres branch maintenance

`pscale branch maintenance run` upgrades a Postgres branch to the latest cluster image. This is how regular version bumps, bugfixes, and quality-of-life improvements reach a branch; PlanetScale otherwise upgrades images only in emergencies, such as patching security issues.

```bash
# Run maintenance now
pscale branch maintenance run <database> <branch> --org <org> --format json

# Also upgrade to the latest PostgreSQL minor version
pscale branch maintenance run <database> <branch> --org <org> --format json --update-postgres-minor-version
```

- The upgrade is applied to the replicas first, followed by a switchover from the old primary to an upgraded replica. That failover leads to a short period of database unavailability (seconds) and terminates all direct connections. A branch running a single instance has no replica to switch over to and is unavailable until it comes back. Warn the user before running this.
- The command returns `{"result": "maintenance started", "branch": "<branch>"}` and exits; it does not wait. Check progress with `pscale branch infra <database> <branch> --org <org> --format json`.
- Rejected while a change request from `pscale branch resize` is still in progress — check `pscale branch resize status` first.
- `--update-postgres-minor-version` is rejected when the branch is already on the latest minor version or the upgrade is unavailable for it.
- Postgres only; Vitess/MySQL databases are rejected before any API call.
- See https://planetscale.com/docs/postgres/operations-philosophy

## Billing payment methods

Update the organization's card through Stripe-hosted Checkout. The CLI never collects PAN; the human must finish Checkout in a browser. There is only one current card.

```bash
pscale billing payment-method update --org <org> --format json
pscale billing payment-method status <setup-id> --org <org> --format json
pscale billing payment-method show --org <org> --format json
pscale billing payment-method delete --org <org> --format json --force
```

`--org` is required (same as other resource commands). `status <setup-id>` takes the `id` returned by `update` (pending JSON on stderr, or the setup object on stdout). That id is a **setup** id, not the saved card id. `show` and `delete` operate on the organization's current card.

**Happy path — leave `update` running.** It creates a setup, opens Checkout when possible, and **polls until a terminal state**. Do not poll `status` yourself unless `update` was interrupted.

Same JSON shape as `auth login`: pending object on **stderr** immediately; a single setup object on **stdout** when polling finishes.

```json
{
  "status": "pending",
  "id": "<setup-id>",
  "checkout_url": "https://checkout.stripe.com/...",
  "browser_opened": true,
  "message": "Complete Stripe Checkout in the browser to continue",
  "next_steps": [
    "Complete Stripe Checkout",
    "pscale billing payment-method status <setup-id> --org <org> --format json"
  ]
}
```

1. Surface `checkout_url` to the user. If `browser_opened` is false, tell them to open it. You cannot complete Checkout.
2. Keep `update` running. Do not start a second `update` while the first is waiting — that creates a new Checkout session.
3. When `update` exits, read **stdout**. Branch on `state`: `completed`, `failed`, or `expired`. `failed` includes a user-facing `error`.

A non-`completed` terminal state exits non-zero and prints an error envelope on **stdout** carrying `id`, `state`, and the recovery command:

| Code | Meaning |
|------|---------|
| `PAYMENT_METHOD_SETUP_FAILED` | Verification failed; `error` has the user-facing reason. The Checkout session is spent — run `update` again. |
| `PAYMENT_METHOD_SETUP_EXPIRED` | Checkout was not completed in time; run `update` again. |
| `PAYMENT_METHOD_SETUP_INTERRUPTED` | Polling stopped while the setup was still pending. `action_required`; resume with `status <setup-id>`, do not run `update` again. |

Follow `next_steps`. Only `PAYMENT_METHOD_SETUP_INTERRUPTED` is resumable; for the other two, `status` will keep returning the same terminal state.

**If `update` is killed or times out**, take `id` from the pending stderr object (do not call `update` again) and GET once:

```bash
pscale billing payment-method status <setup-id> --org <org> --format json
```

`status` does **not** poll. If `state` is still `pending`, wait and call `status` again with the same setup id. Same terminal states as `update`.

**Current card.** `show` returns the saved card. `delete` requires user approval, then `--force` in JSON (`CONFIRMATION_REQUIRED` without it). Failures print the same envelope as other agent commands (`status`, `error`, `issues`, `next_steps`):

| Code | Meaning |
|------|---------|
| `NOT_FOUND` | No current card. Next step is `pscale billing payment-method update --org <org> --format json`. |
| `UNPAID_INVOICES` | `delete` rejected because the organization has unpaid invoices. Pay them, then retry. |

## Imports (Cloudflare D1) — Postgres only

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

## PlanetScale agent skills

Operational workflows (inventory, safety review, Insights, schema recommendations, Traffic Control) live in the public skills repo — not in this file.

```bash
git clone https://github.com/planetscale/skills.git && cd skills && script/setup
# or: npx skills add planetscale/skills -g -y
```

After installing skills, load `14-pscale-cli-automation` for CLI conventions (or run `pscale --skill` from any `pscale` binary to print this reference). Use `00-safe-orchestrator` when the user asks for a full PlanetScale assessment.
