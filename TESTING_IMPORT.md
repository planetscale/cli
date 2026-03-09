# PostgreSQL Import Testing Guide

Hey! Here's how to test the new `pscale branch import` commands. This is pretty straightforward - just follow along and make sure stuff works.

## What You Need

1. A source PostgreSQL database (Supabase, Neon, local Postgres, whatever)
   - Needs to be version 10+ (logical replication support)
   - You need connection details (host, port, username, password, database name)
   - **Important:** Don't use a pooler connection! It has to be a direct database connection (port 5432, not 6543)

2. A PlanetScale PostgreSQL database and branch
   - Just create a test db/branch for this

3. The CLI built from this branch

## Basic Happy Path Test

This is the main workflow - start an import, watch it, then complete it.

### 1. Start an Import

**Using a connection URI:**

```bash
pscale branch import start mydb main --source "postgresql://user:pass@hostname:5432/dbname"
```

**Or with individual flags:**

```bash
pscale branch import start mydb main \
  --host hostname \
  --port 5432 \
  --username myuser \
  --password mypass \
  --source-database dbname
```

**What should happen:**

- CLI checks for `psql` and `pg_dump` on your system
- Connects to source and runs pre-flight checks (WAL level, permissions, etc.)
- Shows you a summary and asks for confirmation
- Creates a publication on source
- Imports the schema (via pg_dump | psql)
- Creates a subscription on destination
- Prints success message with next steps

**Look out for:**

- Clear error if `psql` isn't installed
- Helpful error if you accidentally used a pooler connection (port 6543 or hostname with "pooler" in it)
- Should reject IPv6 addresses
- Password should be redacted in any output

### 2. Check Status

```bash
pscale branch import status mydb main
```

**What should happen:**

- Shows subscription info (enabled, slot name, LSN positions, lag)
- Lists all tables with their replication state:
  - `initializing` → `copying data` → `synchronized` → `ready`
- Summary counts at the bottom

**Try these too:**

```bash
# Watch mode (updates every 2 seconds)
pscale branch import status mydb main --watch

# JSON output
pscale branch import status mydb main --format json
```

### 3. Complete the Import

```bash
pscale branch import complete mydb main
```

**What should happen:**

- Verifies all tables are replicating
- Shows replication lag
- Asks you to type the branch name to confirm (safety check)
- Fast-forwards sequence values (samples for 10 seconds, adds 60-second buffer)
- Drops the subscription
- Drops the publication on source
- Deletes the replication role
- Clears stored credentials

**Look out for:**

- Two-step confirmation (type branch name, then "yes")
- Should error if tables aren't all ready yet
- Should warn about replication lag if it's > 5 seconds

## Other Scenarios to Test

### List Imports

```bash
pscale branch import list mydb

# Or for a specific branch
pscale branch import list mydb main

# With connection info
pscale branch import list mydb main --show-credentials
```

Should show all active imports with status.

### Cancel an Import

```bash
pscale branch import cancel mydb main
```

**Try with different flags:**

```bash
# Drop all the imported tables too
pscale branch import cancel mydb main --drop-schema

# Skip confirmation
pscale branch import cancel mydb main --force
```

**What should happen:**

- Disables and drops subscription
- Tries to clean up publication on source (gracefully continues if source is unreachable)
- Deletes replication role
- Clears credentials
- Optionally drops imported tables

### Dry Run

```bash
pscale branch import start mydb main --source "..." --dry-run
```

Should validate everything but not actually create anything. Good for testing with real sources without side effects.

### Table Filtering

**Include specific tables:**

```bash
pscale branch import start mydb main --source "..." \
  --include-tables "users,posts,comments"
```

**Exclude specific tables:**

```bash
pscale branch import start mydb main --source "..." \
  --skip-tables "audit_logs,temp_data"
```

**With schema qualifiers:**

```bash
pscale branch import start mydb main --source "..." \
  --include-tables "public.users,auth.sessions"
```

### Multiple Schemas

```bash
pscale branch import start mydb main --source "..." \
  --schemas "public,app,auth"

# Or exclude specific ones
pscale branch import start mydb main --source "..." \
  --exclude-schemas "temp,staging"
```

## Edge Cases to Check

### Error Handling

1. **No psql installed** - Should give clear error message
2. **Bad connection string** - Should fail gracefully with helpful error
3. **Pooler connection** - Should reject with message about needing direct connection
4. **WAL level not 'logical'** - Pre-flight checks should catch this
5. **Insufficient permissions** - Should error during pre-flight checks
6. **No replication slots available** - Should error with clear message

### Conflicts

Try starting an import when tables already exist in the destination branch:

```bash
pscale branch import start mydb main --source "..." --force
```

Should detect conflicting tables and ask if you want to drop them.

### Multiple Imports

Try running two imports to the same branch with different publications/subscriptions. The commands should handle storing/retrieving multiple imports correctly.

### Connection Issues

- Start an import
- Kill the source database
- Try to complete/cancel - should handle gracefully

## Things That Should Work Smoothly

1. **Password handling** - If you omit `--password`, it should prompt you securely (not echo to terminal)
2. **Credential storage** - After starting an import, status/complete/cancel should work without needing to provide source credentials again
3. **Watch mode** - Should update cleanly without weird terminal artifacts
4. **Force flags** - All commands with `--force` should skip confirmations
5. **JSON output** - Should be valid JSON (pipe to `jq` to verify)

## Quick Smoke Test

If you just want to verify nothing is broken:

1. Start a local postgres with docker: `docker run -e POSTGRES_PASSWORD=pass -p 5432:5432 postgres:15`
2. Create a test table: `psql postgresql://postgres:pass@localhost:5432/postgres -c "CREATE TABLE test (id serial primary key, name text)"`
3. Run: `pscale branch import start mydb testbranch --source postgresql://postgres:pass@localhost:5432/postgres --dry-run`
4. Should succeed and show what it would do

## Common Issues

- **"psql not found"** - Install PostgreSQL client tools
- **"WAL level is not logical"** - Your source DB needs `wal_level=logical` in postgresql.conf
- **"Connection refused"** - Check firewall, make sure source allows connections
- **"Too many replication slots"** - Source has hit max_replication_slots limit

## What Good Looks Like

A successful full import should:

1. Start without errors
2. Show progress with `status --watch`
3. All tables eventually reach "ready" state
4. Replication lag should be < 1 second for idle databases
5. Complete successfully with sequences fast-forwarded
6. Leave no traces on source or destination (except the actual data)
