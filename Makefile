.PHONY: all
all: build test lint

.PHONY: test
test:
	go test ./...

.PHONY: build
build:
	go build ./...

.PHONY: licensed
licensed:
	licensed cache
	licensed status

.PHONY: lint
lint: 
	@script/lint
