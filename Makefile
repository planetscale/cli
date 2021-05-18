.PHONY: all
all: build test lint

.PHONY: test
test:
	go test ./...

.PHONY: build
build:
	go build ./...


.PHONY: lint
lint: 
	@script/lint
