# Installing msh

## Requirements

- Linux or macOS on the machine where you run msh (servers only need sshd
  and a POSIX shell).
- Go 1.24 or newer to build from source.
- Optional: tmux, for `msh shell --tmux`.

## Build from source

```bash
make build                 # ./msh with version, commit and date injected
./msh --version
```

or, without make:

```bash
go build -trimpath -o msh ./cmd/msh
```

Release binaries for several platforms:

```bash
make dist VERSION=1.0.0    # dist/msh-1.0.0-OS-ARCH and dist/SHA256SUMS
```

## Install the binary

```bash
sudo install -m 0755 msh /usr/local/bin/msh
```

or for your user only:

```bash
mkdir -p ~/.local/bin && install -m 0755 msh ~/.local/bin/msh
```

With `go install ./cmd/msh` the binary goes to `$(go env GOBIN)` or
`$(go env GOPATH)/bin`.

## First-time setup

```bash
msh config init            # writes ~/.config/msh/config.yaml and inventory.yaml (0600)
$EDITOR ~/.config/msh/inventory.yaml
msh config validate
```

SSH authentication uses your keys and ssh-agent:

```bash
eval "$(ssh-agent)"
ssh-add ~/.ssh/id_ed25519   # passphrase-protected keys must be in the agent
```

Host keys are always verified against `~/.ssh/known_hosts`. Add a server's
key after checking its fingerprint out of band, for example:

```bash
ssh-keyscan -t ed25519 server01.example.com >> ~/.ssh/known_hosts
```

Then:

```bash
msh hosts check
msh uptime --all
```

## Shell completion and man page

```bash
echo 'source <(msh completion bash)' >> ~/.bashrc     # or zsh, after compinit
sudo install -m 0644 docs/man/msh.1 /usr/local/share/man/man1/msh.1
```

Completion offers commands, server, group and bastion names, tasks and
runbooks from your own configuration.

## Uninstall

```bash
rm /usr/local/bin/msh      # or ~/.local/bin/msh
rm -r ~/.config/msh        # configuration and inventory
rm -r ~/.local/state/msh ~/.cache/msh   # history, runbook progress, cached facts
```
