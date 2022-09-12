#!/bin/bash

set -eu

WORKDIR=$(pwd)

export GORELEASER_CURRENT_TAG=$(buildkite-agent meta-data get "release-version")

cd $WORKDIR
make release
