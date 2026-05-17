GOCACHE ?= $(CURDIR)/.cache/go-build
GOMODCACHE ?= $(CURDIR)/.cache/go-mod

.PHONY: build test

build:
	mkdir -p bin
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o bin/gh-repos ./cmd/gh-repos

test:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go test ./...
