#!/bin/bash

set -eu

WORKDIR=$(pwd)

GORELEASER_CURRENT_TAG=$(buildkite-agent meta-data get "release-version")
export GORELEASER_CURRENT_TAG

tmpdir=$(mktemp -d)
cat >"$tmpdir/docker.json" <<EOF
{
  "registries": [
    {
      "user" : "$DOCKER_USERNAME",
      "pass" : "$DOCKER_PASSWORD",
      "registry" : "index.docker.io"
    }
  ]
}
EOF
trap 'rm -rf -- "$tmpdir"' EXIT

cd "$WORKDIR"
make release DOCKER_CREDS_FILE="$tmpdir/docker.json"
