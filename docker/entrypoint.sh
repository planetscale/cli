#!/bin/sh
set -eu

docker buildx inspect goreleaser >/dev/null 2>&1 \
  || docker buildx create --name goreleaser --driver docker-container --use
docker buildx use goreleaser

exec /base-entrypoint.sh "$@"
