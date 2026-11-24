#!/bin/bash

set -eu

if [ -z "${RELEASE_VERSION:-}" ]; then
  echo "error: RELEASE_VERSION must be set" >&2
  exit 1
fi

if [ -z "${SLACK_WEBHOOK_URL:-}" ]; then
  echo "error: OPS_SLACK_WEBHOOK_URL is not set on the release environment" >&2
  exit 1
fi

version="${RELEASE_VERSION#v}"
head="pscale-${version}"

pr_url() {
  gh pr list --repo "planetscale/$1" --head "$head" --state open --json url --jq '.[0].url // empty'
}

homebrew=$(pr_url homebrew-tap)
scoop=$(pr_url scoop-bucket)

if [ -z "$homebrew" ] || [ -z "$scoop" ]; then
  echo "Could not find open pull requests for ${head}" >&2
  echo "homebrew: ${homebrew:-missing}" >&2
  echo "scoop: ${scoop:-missing}" >&2
  text=$(printf 'CLI %s is published, but the tap pull requests were not found. %s started it: %s' \
    "$RELEASE_VERSION" "${ACTOR:-someone}" "${RUN_URL:-}")
else
  text=$(printf 'CLI %s is published. %s still needs these merged:\n• %s\n• %s' \
    "$RELEASE_VERSION" "${ACTOR:-someone}" "$homebrew" "$scoop")
fi

payload=$(jq -n --arg text "$text" '{text: $text}')

curl -sfS -X POST -H 'Content-type: application/json' --data "$payload" "$SLACK_WEBHOOK_URL"
