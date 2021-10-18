#!/bin/bash

set -eu

WORKDIR=$(pwd)

echo "+++ :construction:  Installing 'svu' tool"
curl -L -o /tmp/svu_linux_x86_64.tar.gz https://github.com/caarlos0/svu/releases/download/v1.8.0/svu_1.8.0_linux_amd64.tar.gz
cd /tmp && tar -zxvf svu_linux_x86_64.tar.gz
cd $WORKDIR

git fetch --tags 

RELEASE_VERSION=$(/tmp/svu minor)

echo "+++ :boom: Bumping to version $RELEASE_VERSION"

git config --global --add url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

buildkite-agent meta-data set "release-version" "$RELEASE_VERSION"

git tag "$RELEASE_VERSION"
git push origin "$RELEASE_VERSION"

echo "✅"
