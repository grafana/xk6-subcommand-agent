.PHONY: prepare vet test lint fmt tidy

# prepare resolves module dependencies so downstream jobs (lint, vet, build)
# can run offline from a warmed cache.
prepare:
	go mod download

# vet runs the standard Go analyzer across every package.
vet:
	go vet ./...

# test runs the full test suite with the race detector enabled, matching
# what the "Test Go Code" CI job does.
test:
	go test -race ./...

# lint runs golangci-lint using the repository's .golangci.yml.
lint:
	golangci-lint run ./...

# fmt rewrites Go files in place.
fmt:
	gofmt -w .

# tidy reconciles go.mod / go.sum against the source tree.
tidy:
	go mod tidy
