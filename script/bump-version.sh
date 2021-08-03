#!/bin/bash

set -eu

WORKDIR=$(pwd)
SVU_BIN="${WORKDIR}/svu"

echo "+++ :construction:  Installing 'svu' tool"
curl -sfL https://install.goreleaser.com/github.com/caarlos0/svu.sh | bash -s -- -b $WORKDIR

git fetch --tags 

RELEASE_VERSION=$($SVU_BIN minor)

echo "+++ :boom: Bumping to version $RELEASE_VERSION"

git config --global --add url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

buildkite-agent meta-data set "release-version" "$RELEASE_VERSION"

git tag "$RELEASE_VERSION"
git push origin "$RELEASE_VERSION"

echo "✅"
