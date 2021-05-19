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
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@staticcheck ./...

.PHONY: licensed
licensed:
	licensed cache
	licensed status

