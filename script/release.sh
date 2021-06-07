#!/bin/bash

set -eu

WORKDIR=$(pwd)
SVU_BIN="${WORKDIR}/svu"

echo "+++ :construction:  Installing 'svu' tool"
curl -sfL https://install.goreleaser.com/github.com/caarlos0/svu.sh | bash -s -- -b $WORKDIR

# TODO: change patch to minor
RELEASE_VERSION=$($SVU_BIN patch)

echo "+++ :boom: Bumping to version $RELEASE_VERSION"

git config --global --add url."https://${ACTIONS_BOT_TOKEN}@github.com/".insteadOf "https://github.com/"

git tag "$RELEASE_VERSION"
git push origin "$RELEASE_VERSION"

echo "✅"
