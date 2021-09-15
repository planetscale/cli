#!/bin/bash

set -eu

WORKDIR=$(pwd)


export DEBIAN_FRONTEND=noninteractive

echo "--- installing docker cli"
apt-get update
apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

echo \
  "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian \
  $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

apt-get update
apt-get install -y docker-ce-cli 

echo "--- installing goreleaser"

curl -L -o /tmp/goreleaser_Linux_x86_64.tar.gz https://github.com/goreleaser/goreleaser/releases/download/v0.179.0/goreleaser_Linux_x86_64.tar.gz

cd /tmp && tar -zxvf goreleaser_Linux_x86_64.tar.gz

echo "--- running goreleaser"

echo "Login to the docker..."
echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin

export GORELEASER_CURRENT_TAG=$(buildkite-agent meta-data get "release-version")

cd $WORKDIR
/tmp/goreleaser release --rm-dist 
