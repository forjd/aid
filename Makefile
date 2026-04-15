BINARY := aid
COVERAGE_PROFILE := coverage.out
COVERAGE_THRESHOLD := 80.0

.PHONY: build test test-cover check-coverage fmt run

build:
	go build -o ./bin/$(BINARY) ./cmd/aid

test:
	go test ./...

test-cover:
	go test ./... -coverpkg=./... -coverprofile=$(COVERAGE_PROFILE)
	go tool cover -func=$(COVERAGE_PROFILE)

check-coverage: test-cover
	@total=$$(go tool cover -func=$(COVERAGE_PROFILE) | awk '/^total:/ {gsub("%","",$$3); print $$3}'); \
	awk -v total="$$total" -v threshold="$(COVERAGE_THRESHOLD)" 'BEGIN { \
		if (total + 0 < threshold + 0) { \
			printf("coverage %.1f%% is below threshold %.1f%%\n", total + 0, threshold + 0); \
			exit 1; \
		} \
	}'

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './bin/*')

run:
	go run ./cmd/aid --help
