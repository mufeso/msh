# Installed packages

`msh packages` reads package data from servers that use **rpm** (Red Hat,
Rocky, AlmaLinux, Fedora, SUSE) or **dpkg** (Debian, Ubuntu). msh detects
which one a server has, and the commands work the same on both, also in a
mixed fleet. They only read: nothing is installed or removed.

```bash
msh packages list kernel root@web01
msh packages list --all --output csv > inventory.csv
msh packages check --group web --package 'nginx,openssl>=3.0,!telnet'
msh packages check --all --file baseline.txt --only-problems
msh packages find openssl --all
msh packages compare --group web --baseline web01
msh packages updates --all
```

They accept the usual selection, `--where` and output flags.

## Patterns

`list`, `find`, `updates` and `compare` take a pattern that works like grep:
a name only has to contain it, and it is a regular expression.

| Pattern | Finds |
|---------|-------|
| `xerox` | every name containing xerox |
| `^open` | names that start with open |
| `ssl$` | names that end with ssl |
| `lib.*ssl` | lib, later followed by ssl |

`--ignore-case` is `grep -i`. The same rule applies to `msh hosts list`,
`msh task list`, `msh runbook list` and `msh history list`, which search their
whole row.

## check

A requirement is one of:

| Form | Meaning |
|------|---------|
| `nginx` | must be installed |
| `openssl>=3.0` | must be installed in this version range (`=`, `!=`, `<`, `<=`, `>`, `>=`) |
| `!telnet` | must not be installed |

Quote requirements in the shell, because `>` and `!` mean something there.
The exit code is 1 if a requirement is not met, which makes it easy to use
from scripts and cron. Statuses: `ok`, `missing`, `wrong-version`,
`unwanted`.

`--file` reads requirements from a file, one per line, with `#` comments
(see examples/packages-baseline.txt).

## Version comparison

Versions are not compared as plain text: 1.10 is newer than 1.9. msh
uses ported versions of the algorithms of rpm (`rpmvercmp`) and dpkg
(`verrevcmp`), including `~` for pre-releases, and picks the right one for
each server, so one requirement works across a mixed fleet.

A requirement compares only the parts it names. An installed
`1:3.0.7-16.el9` satisfies `openssl>=3.0`, because that requirement has no
epoch and no release. Write them to compare them: `openssl>=1:3.0.7-16.el9`.
Distributions backport security fixes without changing the upstream
version, so check the release when that matters.

## compare

Lists packages that are missing, extra or at another version compared with
the baseline server, and exits 1 when servers differ. Comparing rpm with
dpkg servers prints a warning, because their versions can't match.

## updates

Lists pending updates with `dnf check-update`, `yum check-update` or
`apt list --upgradable`. Nothing is installed. dnf may refresh its metadata
cache, so this can take a while; apt shows what its last `apt update` found.
To install updates, write a runbook.
