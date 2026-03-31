#!/bin/bash

set -eu

DRY_RUN="${DRY_RUN:-false}"
WORKDIR=$(pwd)

if [ -z "${GORELEASER_CURRENT_TAG:-}" ]; then
  echo "error: GORELEASER_CURRENT_TAG must be set" >&2
  exit 1
fi
export GORELEASER_CURRENT_TAG

VERSION="${GORELEASER_CURRENT_TAG#v}"

all_targets=(docker homebrew scoop aur)
skip=()

if docker manifest inspect "planetscale/pscale:${GORELEASER_CURRENT_TAG}" >/dev/null 2>&1; then
  echo "==> Docker image already exists, skipping docker"
  skip+=(docker)
fi

if gh api "repos/planetscale/homebrew-tap/contents/Formula/pscale.rb" --jq '.content' 2>/dev/null \
    | base64 --decode 2>/dev/null | grep -q "version \"${VERSION}\""; then
  echo "==> Homebrew formula already up to date, skipping homebrew"
  skip+=(homebrew)
fi

if gh api "repos/planetscale/scoop-bucket/contents/pscale.json" --jq '.content' 2>/dev/null \
    | base64 --decode 2>/dev/null | grep -q "\"version\": \"${VERSION}\""; then
  echo "==> Scoop manifest already up to date, skipping scoop"
  skip+=(scoop)
fi

if curl -fsSL --connect-timeout 5 "https://aur.archlinux.org/cgit/aur.git/plain/PKGBUILD?h=pscale-cli-bin" 2>/dev/null \
    | grep -q "pkgver=${VERSION}"; then
  echo "==> AUR package already up to date, skipping aur"
  skip+=(aur)
fi

should_skip() {
  local target=$1
  for s in "${skip[@]+"${skip[@]}"}"; do
    [ "$target" = "$s" ] && return 0
  done
  return 1
}

run=()
echo ""
echo "==> Release plan for ${GORELEASER_CURRENT_TAG}:"
for target in "${all_targets[@]}"; do
  if should_skip "$target"; then
    echo "      skip  ${target}"
  else
    echo "      run   ${target}"
    run+=("$target")
  fi
done
echo ""

if [ ${#run[@]} -eq 0 ]; then
  echo "==> All targets already published, nothing to do."
  exit 0
fi

if [ "$DRY_RUN" = "true" ]; then
  echo "==> Dry run, exiting."
  exit 0
fi

GORELEASER_SKIP=$(IFS=,; echo ${skip[*]+"${skip[*]}"})

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
make release DOCKER_CREDS_FILE="$tmpdir/docker.json" \
  GORELEASER_EXTRA_ARGS="${GORELEASER_SKIP:+--skip=${GORELEASER_SKIP}}"
