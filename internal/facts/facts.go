// Package facts collects information about servers, such as the operating
// system, kernel, CPUs and memory. Facts are used for reporting, for
// selecting servers (--where) and for conditions in runbooks (when:).
package facts

import (
	"slices"
	"strings"

	"example.com/msh/internal/inventory"
)

// Command is the remote command that reads Script from standard input.
const Command = "sh -s --"

// Script prints one key=value line per fact. It only reads information
// and works with any POSIX shell; missing tools leave a fact empty.
const Script = `
if [ -r /etc/os-release ]; then . /etc/os-release; fi
echo "hostname=$(hostname 2>/dev/null)"
echo "kernel_name=$(uname -s 2>/dev/null)"
echo "kernel=$(uname -r 2>/dev/null)"
echo "arch=$(uname -m 2>/dev/null)"
echo "os_id=${ID:-}"
echo "os_like=${ID_LIKE:-}"
echo "os_name=${PRETTY_NAME:-}"
echo "os_version=${VERSION_ID:-}"
echo "cpus=$(getconf _NPROCESSORS_ONLN 2>/dev/null)"
if [ -r /proc/meminfo ]; then
  echo "memory_mb=$(awk '/^MemTotal:/ {print int($2 / 1024)}' /proc/meminfo)"
fi
if [ -r /proc/uptime ]; then
  echo "uptime_seconds=$(cut -d. -f1 /proc/uptime)"
fi
echo "disk_root_pct=$(df -P / 2>/dev/null | awk 'NR == 2 {print $5 + 0}')"
if command -v dpkg-query >/dev/null 2>&1 && [ -f /var/lib/dpkg/status ]; then
  echo "package_manager=dpkg"
elif command -v rpm >/dev/null 2>&1; then
  echo "package_manager=rpm"
else
  echo "package_manager=none"
fi
if [ -d /run/systemd/system ]; then echo "init=systemd"; else echo "init=other"; fi
`

// Facts maps fact names to values.
type Facts map[string]string

// Parse reads the output of Script. Lines without "=" are ignored, so
// a message printed by a login script does no harm. os_family is derived.
func Parse(output string) Facts {
	f := Facts{}
	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if ok && key != "" && !strings.ContainsAny(key, " \t") {
			f[key] = value
		}
	}
	f["os_family"] = family(f["os_id"], f["os_like"], f["kernel_name"])
	return f
}

// families maps distribution IDs to families.
var families = map[string]string{
	"rhel": "redhat", "fedora": "redhat", "centos": "redhat", "rocky": "redhat",
	"almalinux": "redhat", "ol": "redhat", "amzn": "redhat",
	"debian": "debian", "ubuntu": "debian", "linuxmint": "debian", "raspbian": "debian",
	"sles": "suse", "opensuse": "suse", "opensuse-leap": "suse", "suse": "suse",
	"arch": "arch", "manjaro": "arch",
	"alpine": "alpine",
}

// family finds the family from ID, then the words of ID_LIKE.
func family(id, like, kernelName string) string {
	for _, word := range slices.Concat([]string{id}, strings.Fields(like)) {
		if f, ok := families[strings.ToLower(word)]; ok {
			return f
		}
	}
	if kernelName != "" && kernelName != "Linux" {
		return strings.ToLower(kernelName)
	}
	return "unknown"
}

// Env returns the names an expression can use for a server: its facts
// (also as facts.NAME), name, address, user and tag.KEY.
func Env(s inventory.Server, f Facts) func(string) (string, bool) {
	return func(name string) (string, bool) {
		switch name {
		case "name":
			return s.Name, true
		case "address":
			return s.Address, true
		case "user":
			return s.User, true
		}
		if key, ok := strings.CutPrefix(name, "tag."); ok {
			v, ok := s.Tags[key]
			return v, ok
		}
		v, ok := f[strings.TrimPrefix(name, "facts.")]
		return v, ok
	}
}
