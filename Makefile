BINARY := aid

.PHONY: build test fmt run

build:
	go build -o ./bin/$(BINARY) ./cmd/aid

test:
	go test ./...

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './bin/*')

run:
	go run ./cmd/aid --help

