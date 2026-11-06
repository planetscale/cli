#!/bin/bash

set -eu

git fetch --tags

RELEASE_VERSION=$(go tool -modfile=tools/go.mod svu minor)

echo "+++ :boom: Bumping to version $RELEASE_VERSION"

echo "RELEASE_VERSION=$RELEASE_VERSION" >>"$GITHUB_OUTPUT"

git tag "$RELEASE_VERSION"
git push origin "$RELEASE_VERSION"

echo "✅"
