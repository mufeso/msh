# msh — My Parallel SSH

msh is a command-line SSH fleet-management tool for Linux servers. It is
being built step by step as a Go learning project.

At this step msh can only print its banner, help and version. Later steps
add an inventory, configuration, SSH execution, bastion hosts, tasks,
parallel execution, interactive shells and tmux integration.

## Layout

```
cmd/msh/            the executable (tiny main package)
internal/app/       command-line interface
internal/buildinfo/ name, version, commit, build date
scripts/check.sh    runs every quality check
Makefile            fmt, vet, test, race, build, clean
```

## Build and test

```bash
gofmt -w .
go vet ./...
go test ./...
go build -o msh ./cmd/msh
./msh --help
./msh --version
```

Or run everything at once:

```bash
./scripts/check.sh --race
```

## Version injection

```bash
make VERSION=1.2.3 COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo unknown) build
./msh --version
```

## Layered steps

Each `stepNNN` directory contains the complete project at that step. The
file `.step` explains what changed and which Go concept it introduces.
