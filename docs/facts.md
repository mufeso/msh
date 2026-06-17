# Facts

Facts are read-only information msh collects from servers.

```bash
msh facts --all                                   # table
msh facts --group web --output csv
msh facts --all --refresh                         # ignore the cache
msh facts --all --group-by os_version             # how uniform is the fleet?
msh facts --all --expect 'disk_root_pct < 90'     # health gate: exit 1 on violations
msh uptime --where 'os_family == "redhat" && memory_mb < 4096'
```

| Fact | Meaning |
|------|---------|
| `hostname`, `kernel_name`, `kernel`, `arch` | from `hostname` and `uname` |
| `os_id`, `os_like`, `os_name`, `os_version` | from `/etc/os-release` |
| `os_family` | derived: redhat, debian, suse, arch, alpine, ... |
| `cpus`, `memory_mb` | online CPUs, total memory |
| `disk_root_pct` | usage of the root filesystem |
| `uptime_seconds` | since boot |
| `package_manager` | rpm, dpkg or none |
| `init` | systemd or other |

The script is streamed over SSH, like `msh script run`, and only reads.

## Cache

Facts are cached per server in `~/.cache/msh/facts` (mode 0600) for
`facts_ttl` (default 1h, `MSH_FACTS_TTL`; `0s` disables it). A cached entry
only counts for the same address. `--refresh` collects new data. Remember
that values such as `disk_root_pct` can be up to `facts_ttl` old.

## Expressions

`--where`, `--expect` and runbook `when:` share one small language. It can
only compare values; it can't run or change anything.

```
os_family == "redhat" && cpus >= 4
!(disk_root_pct > 90) || name ~ "^db"
tag.environment == "production"
```

| Element | Meaning |
|---------|---------|
| `==` `!=` `<` `<=` `>` `>=` | compare; as numbers if both sides are numbers, else as strings |
| `~` `!~` | regular expression match; the pattern must be quoted |
| `&&` `\|\|` `!` `( )` | logic; `!` binds tightest, then comparisons, `&&`, `\|\|` |
| `name`, `address`, `user`, `tag.KEY` | from the inventory |
| any other name | a fact (`facts.NAME` also works); unknown names are empty |

A bare name is true unless it is empty, `0` or `false`.

Comparing versions as numbers only works for simple values such as `9.4`;
for package versions use `msh packages check`.

## --where

`--where` works with `exec`, `task run`, `script run`, `logs`, shortcuts,
`facts` and `runbook run`. It filters the other selectors, or all servers
when used alone. Servers whose facts can't be collected are not selected
and a warning says so. Interactive sessions don't support it.

## Conditions in runbooks

```yaml
steps:
  - name: update redhat
    when: os_family == "redhat"
    command: "dnf -y update nginx"
  - name: update debian
    when: os_family == "debian"
    command: "apt-get -y install --only-upgrade nginx"
```

A false condition skips the step on that server. A server without facts
fails instead of skipping, so it can't end up half done without notice.
