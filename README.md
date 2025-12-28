# msh — My Parallel SSH

msh is a command-line SSH fleet-management tool for Linux servers. It is
being built step by step as a Go learning project.

## Current features

- `msh hosts list` — list inventory servers, selected by `--host`,
  `--group`, `--tag` or `--all`, as a table or JSON.
- `msh config show` — effective settings and where each value came from.
- `msh config validate` — validate the configuration and inventory.
- `msh version`, `msh --version`, `msh --help`.

See [docs/configuration.md](docs/configuration.md) for configuration.

## Layout

```
cmd/msh/             the executable (tiny main package)
internal/app/        command-line interface
internal/buildinfo/  name, version, commit, build date
internal/cliflag/    custom flag types
internal/config/     settings, layers and precedence
internal/decode/     strict JSON/YAML decoding
internal/exitcode/   documented exit codes
internal/inventory/  servers and selection
internal/output/     text tables and JSON
scripts/check.sh     runs every quality check
```

## Build and test

```bash
gofmt -w .
go vet ./...
go test ./...
go build -o msh ./cmd/msh
```

Or: `./scripts/check.sh --race`

## Layered steps

Each `stepNNN` directory contains the complete project at that step. The
file `.step` explains what changed and which Go concept it introduces.
