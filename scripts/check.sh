#!/usr/bin/env bash
# check.sh runs every quality check for this step of msh.
#
#   ./scripts/check.sh          format check, vet, test, build
#   ./scripts/check.sh --race   the same plus the race detector
set -euo pipefail

cd "$(dirname "$0")/.."

echo "==> gofmt"
unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
	echo "These files need formatting (run: gofmt -w .):"
	echo "$unformatted"
	exit 1
fi

echo "==> go vet"
go vet ./...

echo "==> go test"
go test -count=1 ./...

if [[ "${1:-}" == "--race" ]]; then
	echo "==> go test -race"
	go test -count=1 -race ./...
fi

echo "==> go build"
go build -o msh ./cmd/msh

echo "All checks passed. Binary: ./msh"
