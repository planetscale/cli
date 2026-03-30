#!/bin/bash

set -eu

WORKDIR=$(pwd)

if [ -z "${GORELEASER_CURRENT_TAG:-}" ]; then
  echo "error: GORELEASER_CURRENT_TAG must be set" >&2
  exit 1
fi
export GORELEASER_CURRENT_TAG

tmpdir=$(mktemp -d)
cat >"$tmpdir/docker.json" <<EOF
{
  "registries": [
    {
      "user" : "$DOCKER_USERNAME",
      "pass" : "$DOCKER_PASSWORD",
      "registry" : ""
    }
  ]
}
EOF
trap 'rm -rf -- "$tmpdir"' EXIT

cd "$WORKDIR"
make release DOCKER_CREDS_FILE="$tmpdir/docker.json"
