package pkgs

import (
	"context"
	"strings"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/runner"
)

// UpdatesCommand reads UpdatesScript from standard input.
const UpdatesCommand = "sh -s -- msh-updates"

// UpdatesScript lists pending updates without installing anything. dnf
// and yum exit 100 when updates exist, which is not a failure here. dnf
// may refresh its metadata cache; apt shows what its last update found.
const UpdatesScript = `
if command -v dnf >/dev/null 2>&1; then tool=dnf
elif command -v yum >/dev/null 2>&1; then tool=yum
elif command -v apt >/dev/null 2>&1; then tool=apt
else echo "#tool=none"; exit 0
fi
echo "#tool=$tool"
if [ "$tool" = apt ]; then
  apt list --upgradable 2>/dev/null
else
  "$tool" -q check-update
  rc=$?
  if [ "$rc" -eq 100 ]; then rc=0; fi
  exit "$rc"
fi
`

// Update is a pending update.
type Update struct {
	Name    string `json:"name"`
	Version string `json:"version"` // the new version
	Source  string `json:"source"`  // repository or suite
}

// ParseUpdates reads the output of UpdatesScript.
func ParseUpdates(output string) (tool string, updates []Update) {
	tool = None
	for _, line := range strings.Split(output, "\n") {
		if t, ok := strings.CutPrefix(line, "#tool="); ok {
			tool = strings.TrimSpace(t)
			continue
		}
		f := strings.Fields(line)
		switch {
		case tool == "apt" && len(f) >= 3 && strings.Contains(f[0], "/"):
			// nginx/noble-updates 1.24.0-2ubuntu7.1 amd64 [upgradable from: ...]
			name, suite, _ := strings.Cut(f[0], "/")
			updates = append(updates, Update{Name: name, Version: f[1], Source: suite})
		case (tool == "dnf" || tool == "yum") && len(f) == 3 && strings.Contains(f[0], "."):
			// nginx.x86_64  1:1.26.0-1.el9  appstream
			name := f[0][:strings.LastIndexByte(f[0], '.')]
			updates = append(updates, Update{Name: name, Version: f[1], Source: f[2]})
		}
	}
	return tool, updates
}

// CollectUpdates reads pending updates from each server. Listing can
// take a while because dnf may refresh metadata.
func CollectUpdates(ctx context.Context, ex executor.Executor, servers []inventory.Server, workers int, timeout time.Duration) (map[string][]Update, []executor.Result) {
	req := executor.Request{Command: UpdatesCommand, Stdin: []byte(UpdatesScript), Timeout: timeout, OutputLimit: outputLimit}
	report := runner.Run(ctx, ex, servers, req, runner.Options{Workers: workers}, nil)
	found := make(map[string][]Update, len(servers))
	var failed []executor.Result
	for _, r := range report.Results {
		if r.Status != executor.StatusOK {
			failed = append(failed, r)
			continue
		}
		_, found[r.Server] = ParseUpdates(r.Stdout)
	}
	return found, failed
}
