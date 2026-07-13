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

## Flags

| Flag | Purpose |
|------|---------|
| `--format json` | JSON on stdout |
| `--org <org>` | Organization (on resource subcommands only) |
| `--api-url` | Non-production API base URL — pass on every command when not using production |

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
| `--keyspace` | MySQL keyspace (default `@primary`) |
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

## Cursor Cloud specific instructions

To install/rebuild the `pscale` binary from this repo (Go 1.26 is on `PATH`):

```bash
go build -trimpath -o /tmp/pscale ./cmd/pscale && sudo mv -f /tmp/pscale /usr/local/bin/pscale
```

`/usr/local/bin` is already on `PATH`, so `pscale --version` works afterward. The
build entry point is `./cmd/pscale` (`main.version`/`main.commit`/`main.date` can be
set via `-ldflags` if you want an accurate version string). `pscale auth check
--format json` reports `NO_AUTH` until a user runs `pscale auth login` — installing
the CLI does not authenticate it.

### `pscale auth login` hangs on the desktop keyring prompt

`pscale` stores its access token via `99designs/keyring`, preferring the
SecretService (gnome-keyring) backend. In the Cursor Cloud VM the desktop D-Bus
secret service is present but locked, so after you approve the device flow the CLI
**blocks on a "Choose password for new keyring" dialog on the desktop** and never
finishes writing the token (`auth check` keeps reporting `NO_AUTH`).

Work around it by disabling D-Bus so `pscale` falls back to its plaintext file store
at `~/.config/planetscale/access-token`. Prefix the login (and every subsequent
`pscale` command, since reads also open the keyring) with:

```bash
DBUS_SESSION_BUS_ADDRESS=unix:path=/dev/null pscale auth login --format json
DBUS_SESSION_BUS_ADDRESS=unix:path=/dev/null pscale auth check --format json
```

Run `login` non-interactively (e.g. in tmux) and hand the user the `verification_url`
+ `user_code` printed as pending JSON on **stderr**; the final result JSON lands on
**stdout** once approved.
