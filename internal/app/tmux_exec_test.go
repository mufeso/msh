//go:build unix

package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"example.com/msh/internal/exitcode"
)

// fakeTmuxScript logs each invocation's arguments (NUL-separated, one
// invocation per line) and answers like tmux would.
const fakeTmuxScript = `#!/bin/sh
for a in "$@"; do printf '%s\0' "$a" >> "$FAKE_TMUX_LOG"; done
printf '\n' >> "$FAKE_TMUX_LOG"
case "$1" in
  -V) echo "tmux 3.4" ;;
  has-session) exit 1 ;;
  new-session|split-window) n=$(wc -l < "$FAKE_TMUX_LOG"); echo "%$n" ;;
esac
exit 0
`

func TestTmuxWithFakeExecutable(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "tmux"), []byte(fakeTmuxScript), 0o755); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_TMUX_LOG", log)

	configure := func(c *cli) { c.lookPath = exec.LookPath } // the real lookup
	args := append([]string{"shell", "--all", "--tmux", "--tmux-detached", "--tmux-session", "e2e", "--command", "echo 'quoted arg'"}, fleet.args()...)
	_, stderr, code := runCLIWith(t, context.Background(), nil, configure, args...)
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}

	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		calls = append(calls, strings.Split(strings.TrimSuffix(line, "\x00"), "\x00"))
	}
	var cmds []string
	for _, c := range calls {
		cmds = append(cmds, c[0])
	}
	want := []string{"-V", "has-session", "new-session", "set-option", "select-pane", "set-option", "split-window", "select-pane", "select-layout", "select-layout"}
	if strings.Join(cmds, " ") != strings.Join(want, " ") {
		t.Errorf("tmux commands:\n got %v\nwant %v", cmds, want)
	}
	newSession := calls[2]
	if newSession[3] != "e2e" || !strings.Contains(newSession[len(newSession)-1], `--command 'echo '\''quoted arg'\'''`) {
		t.Errorf("new-session args = %q", newSession)
	}
}
