#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DB="${D1_DATABASE:-import-test}"
REMOTE="${D1_REMOTE:-true}"
FRESH="${D1_FRESH:-false}"

remote_flag=()
if [[ "$REMOTE" == "true" ]]; then
  remote_flag=(--remote)
fi

if [[ "$FRESH" == "true" ]]; then
  echo "==> Resetting existing tables"
  wrangler d1 execute "$DB" "${remote_flag[@]}" --file="$ROOT/reset.sql"
fi

echo "==> Applying schema to D1 database: $DB"
wrangler d1 execute "$DB" "${remote_flag[@]}" --file="$ROOT/schema.sql"

echo "==> Generating seed SQL batches"
python3 "$ROOT/generate_seed.py"

echo "==> Loading seed data"
seed_order=(
  000_bootstrap.sql
  organizations
  users
  teams
  team_members
  projects
  tags
  project_tags
  tasks
  task_dependencies
  comments
  attachments
  audit_log
  api_keys
  sessions
  notifications
  invoices
  line_items
  external_entities
  entity_links
)

for prefix in "${seed_order[@]}"; do
  if [[ "$prefix" == *.sql ]]; then
    files=("$ROOT/seed/$prefix")
  else
    mapfile -t files < <(find "$ROOT/seed" -name "${prefix}_*.sql" | sort)
  fi
  for file in "${files[@]}"; do
    [[ -f "$file" ]] || continue
    echo "    -> $(basename "$file")"
    wrangler d1 execute "$DB" "${remote_flag[@]}" --file="$file"
  done
done

echo "==> Done. Export with:"
echo "    wrangler d1 export $DB --remote --output ./import-test-export.sql"
echo "    pscale import d1 lint --input ./import-test-export.sql --format json"
