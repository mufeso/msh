# Makefile for msh — My Parallel SSH.
BINARY  := msh
PKG     := example.com/msh/internal/buildinfo
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 2.5.1)
COMMIT  ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X $(PKG).Version=$(VERSION) -X $(PKG).Commit=$(COMMIT) -X $(PKG).Date=$(DATE)

.PHONY: all check fmt vet test race build dist docs clean

all: check

check: fmt vet test build

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test -count=1 ./...

race:
	go test -count=1 -race ./...

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/msh

docs:
	go generate ./...

dist:
	./scripts/release.sh $(VERSION)

clean:
	rm -f $(BINARY)
	rm -rf dist
