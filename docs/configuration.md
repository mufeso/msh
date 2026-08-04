# msh configuration

msh reads its settings from several layers. A higher layer overrides a
lower one, value by value:

```
command-line flag
  > environment variable
    > inventory "defaults:" block
      > configuration file
        > built-in default
```

Values set on an individual server in the inventory (`user`, `port`)
always win over all layers for that server.

Run `msh config show` to see every effective value and where it came from,
and `msh config validate` to check the configuration and inventory.

## Files

| File | Default location | Override |
|------|------------------|----------|
| Configuration | `$XDG_CONFIG_HOME/msh/config.yaml` or `~/.config/msh/config.yaml` (optional) | `--config FILE`, `MSH_CONFIG` |
| Inventory | `~/.config/msh/inventory.yaml` | `--inventory FILE`, `MSH_INVENTORY`, `inventory:` |

Files may be YAML (`.yaml`, `.yml`) or JSON (`.json`). Unknown keys are
errors, so typos are reported instead of silently ignored. A leading `~/`
in path settings is expanded to `$HOME`.

## Settings

| Key | Flag | Environment | Default | Meaning |
|-----|------|-------------|---------|---------|
| `inventory` | `--inventory` | `MSH_INVENTORY` | `~/.config/msh/inventory.yaml` | inventory file |
| `user` | `--user` | `MSH_USER` | local login name | SSH user for servers without one |
| `port` | `--port` | `MSH_PORT` | `22` | SSH port for servers without one |
| `identity_file` | `--identity` | `MSH_IDENTITY_FILE` | none (agent and standard keys) | private key file |
| `known_hosts_file` | `--known-hosts` | `MSH_KNOWN_HOSTS_FILE` | `~/.ssh/known_hosts` | host keys used for verification |
| `output` | `--output` | `MSH_OUTPUT` | `text` | output format |
| `connect_timeout` | `--connect-timeout` | `MSH_CONNECT_TIMEOUT` | `10s` | TCP connect and SSH handshake, per server |
| `command_timeout` | `--command-timeout` | `MSH_COMMAND_TIMEOUT` | `60s` | remote command, per server (`0s` = none) |
| `timeout` | `--timeout` | `MSH_TIMEOUT` | `0s` | whole run (`0s` = none) |
| `mode` | `--mode` | `MSH_MODE` | `parallel` | `sequential` or `parallel` |
| `workers` | `--workers` | `MSH_WORKERS` | `5` | concurrent servers in parallel mode (1-512) |
| `ordered` | `--ordered` | `MSH_ORDERED` | `false` | print results in selection order |
| `bastion` | `--bastion` | `MSH_BASTION` | none | default bastion (`none` = direct) |
| `allowed_log_paths` | — | `MSH_ALLOWED_LOG_PATHS` | `/var/log` | directories `msh logs` may read |
| `tmux_layout` | `--tmux-layout` | `MSH_TMUX_LAYOUT` | `tiled` | default tmux layout |
| `facts_ttl` | — | `MSH_FACTS_TTL` | `1h` | how long collected facts stay valid (`0s` = no cache) |
| `history` | `--no-history` | `MSH_HISTORY` | `true` | record runs in the history |
| `history_limit` | — | `MSH_HISTORY_LIMIT` | `200` | runs to keep (`0` = all) |

Durations use Go syntax: `500ms`, `30s`, `2m`, `1h30m`.

## Example configuration

```yaml
# ~/.config/msh/config.yaml
inventory: ~/.config/msh/inventory.yaml
user: admin
connect_timeout: 5s
command_timeout: 2m
workers: 10
```

## Example inventory

```yaml
defaults:
  port: 22
  user: admin
  connect_timeout: 10s

servers:
  - name: web01
    address: 192.0.2.10
    groups: [web, production]
    tags:
      environment: production
  - name: db01
    address: db01.example.com
    user: dbadmin
    groups: [database, production]
```

The `defaults:` block accepts every key from the table above except
`inventory`.

## Security

- Passwords are never read from configuration files. Authentication uses
  SSH keys and the SSH agent.
- Host keys are always verified against `known_hosts_file`.
- Exit code 3 means invalid configuration; 8 means a file could not be read.
