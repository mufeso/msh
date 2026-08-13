# Release checklist

## 1. Quality gates

- [ ] `gofmt -l .` prints nothing
- [ ] `go vet ./...` passes
- [ ] `go test -count=1 ./...` passes
- [ ] `go test -count=3 -race ./...` passes
- [ ] `go mod tidy` leaves go.mod and go.sum unchanged
- [ ] `govulncheck ./...` reports no reachable vulnerabilities (optional tool)

## 2. Documentation

- [ ] README.md command list matches `msh --help`
- [ ] docs/ updated for every user-visible change
- [ ] examples/ pass `msh config validate` (covered by tests)
- [ ] CHANGELOG.md has an entry for the version
- [ ] `go generate ./...` leaves docs/ unchanged (covered by tests)

## 3. Build

- [ ] Version chosen (semantic versioning) and tagged: `git tag -a v2.0.0`
- [ ] `make dist VERSION=2.0.0`
- [ ] `dist/SHA256SUMS` generated and checked: `sha256sum -c SHA256SUMS`
- [ ] `./dist/msh-2.0.0-linux-amd64 --version` shows the version and commit

## 4. Smoke test against a real test server

- [ ] `msh hosts check` against a test inventory
- [ ] `msh exec --all --command uptime` in parallel and `--mode sequential`
- [ ] one server through a bastion
- [ ] `msh script run` with arguments
- [ ] `msh shell --host ...`, resize the window, exit with a status
- [ ] `msh shell --group ... --tmux` and CTRL+C during setup
- [ ] CTRL+C during a long `msh exec --all` exits 130 and leaves no sessions
- [ ] unknown host key and wrong key fail with exit 4
- [ ] `msh facts --all` on at least one rpm and one dpkg server
- [ ] `msh packages list`, `check`, `compare` and `updates` on real rpm and dpkg servers
- [ ] a rolling runbook with `--canary`, `--batch` and a failing server, then `--resume`
- [ ] `msh history diff` between two runs
- [ ] bash and zsh completion in a real shell, `man docs/man/msh.1`

## 5. Publish

- [ ] Push the tag, attach binaries and SHA256SUMS
- [ ] Announce changes from CHANGELOG.md
