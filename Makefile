all: build test

.PHONY: build test
test:
	go test ./...

.PHONY: build
build:
	go build ./...

