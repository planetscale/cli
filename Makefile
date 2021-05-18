all: build test

.PHONY: build test
test:
	go test ./...

.PHONY: build
build:
	go build ./...


.PHONY: licensed
licensed:
	licensed cache
	licensed status
