COMMIT := $(shell git rev-parse --short=7 HEAD 2>/dev/null)
VERSION := $(shell git describe --abbrev=0 HEAD 2>/dev/null)
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

ifeq ($(strip $(shell git status --porcelain 2>/dev/null)),)
  GIT_TREE_STATE=clean
else
  GIT_TREE_STATE=dirty
endif

REPO=planetscale
NAME=pscale
BUILD_PKG=github.com/planetscale/cli/cmd/pscale
GORELEASE_CROSS_VERSION ?= v1.18.3

.PHONY: all
all: build test lint

.PHONY: test
test:
	@go test ./...

.PHONY: build
build:
	@go build -trimpath ./...

.PHONY: lint
lint: 
	@go install honnef.co/go/tools/cmd/staticcheck@HEAD
	@staticcheck ./...

.PHONY: licensed
licensed:
	licensed cache
	licensed status

.PHONY: build-image
build-image: 
	@echo "==> Building docker image ${REPO}/${NAME}:$(VERSION)"
	@# Permit building only if the Git tree is clean
	@echo "${GIT_TREE_STATE}" | grep -Eq "^clean" || ( echo "Git tree state is not clean"; exit 1 )
	@docker build --build-arg VERSION=$(VERSION:v%=%) --build-arg COMMIT=$(COMMIT) --build-arg DATE=$(DATE) -t ${REPO}/${NAME}:$(VERSION) .
	@docker tag ${REPO}/${NAME}:$(VERSION) ${REPO}/${NAME}:latest

.PHONY: push
push:
	@# Permit releasing only if VERSION adheres to semver.
	@echo "${VERSION}" | grep -Eq "^v[0-9]+\.[0-9]+\.[0-9]+$$" || ( echo "VERSION \"${VERSION}\" does not adhere to semver"; exit 1 )
	@echo "==> Pushing docker image ${REPO}/${NAME}:$(VERSION)"
	@docker push ${REPO}/${NAME}:latest
	@docker push ${REPO}/${NAME}:$(VERSION)
	@echo "==> Your image is now available at $(REPO)/${NAME}:$(VERSION)"

.PHONY: clean
clean:
	@echo "==> Cleaning artifacts"
	@rm ${NAME}

.PHONY: release
release:
	@docker run \
		--rm \
		-e GITHUB_TOKEN=${GITHUB_TOKEN} \
		-e DOCKER_USERNAME=${DOCKER_USERNAME} \
		-e DOCKER_PASSWORD=${DOCKER_PASSWORD} \
		-e GORELEASER_CURRENT_TAG=${GORELEASER_CURRENT_TAG} \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/${REPO}/${NAME} \
		-w /go/src/${REPO}/${NAME} \
		goreleaser/goreleaser-cross:${GORELEASE_CROSS_VERSION} \
		release --rm-dist
