GOCACHE ?= $(CURDIR)/.cache/go-build
GOMODCACHE ?= $(CURDIR)/.cache/go-mod

.PHONY: build extension test

build:
	mkdir -p bin
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o bin/gh-workspace ./cmd/gh-workspace

extension:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o gh-workspace ./cmd/gh-workspace

test:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go test ./...
