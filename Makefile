# Makefile for msh — My Parallel SSH.
BINARY  := msh
PKG     := example.com/msh/internal/buildinfo
VERSION ?= 0.1.0-dev
COMMIT  ?= unknown
DATE    ?= unknown
LDFLAGS := -X $(PKG).Version=$(VERSION) -X $(PKG).Commit=$(COMMIT) -X $(PKG).Date=$(DATE)

.PHONY: all check fmt vet test race build clean

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

clean:
	rm -f $(BINARY)
