#!/bin/bash

set -eu

WORKDIR=$(pwd)

echo "--- installing goreleaser"

curl -L -o /tmp/goreleaser_Linux_x86_64.tar.gz https://github.com/goreleaser/goreleaser/releases/download/v0.173.2/goreleaser_Linux_x86_64.tar.gz

cd /tmp && tar -zxvf goreleaser_Linux_x86_64.tar.gz

echo "--- running goreleaser"

export GORELEASER_CURRENT_TAG=$(buildkite-agent meta-data get "release-version")

cd $WORKDIR
/tmp/goreleaser release --rm-dist 
