package app

import (
	"os"
	"path/filepath"
	"strconv"

	"example.com/msh/internal/interactive"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/shellquote"
)

// paneArgv returns the command a tmux pane runs for server: this msh
// binary with "shell --host NAME" and every resolved setting passed
// explicitly, so the pane connects exactly as the parent would even
// though tmux may give it a different working directory or environment.
func (c *cli) paneArgv(f *shellFlags, st *setup, server inventory.Server, iopts interactive.Options) []string {
	exe, err := os.Executable()
	if err != nil {
		exe = "msh"
	}
	s := st.settings
	argv := []string{exe, "shell", "--host", server.Name,
		"--inventory", absPath(s.Inventory),
		"--known-hosts", absPath(s.KnownHostsFile),
		"--port", strconv.Itoa(s.Port),
		"--connect-timeout", s.ConnectTimeout.String(),
	}
	if st.configFile != "" {
		argv = append(argv, "--config", absPath(st.configFile))
	}
	if s.User != "" {
		argv = append(argv, "--user", s.User)
	}
	if s.IdentityFile != "" {
		argv = append(argv, "--identity", absPath(s.IdentityFile))
	}
	bastion := server.Bastion
	if bastion == "" {
		bastion = "none"
	}
	argv = append(argv, "--bastion", bastion)
	if f.ro.insecureIgnoreHostKey {
		argv = append(argv, "--insecure-ignore-host-key")
	}
	if iopts.Command != "" {
		argv = append(argv, "--command", iopts.Command)
	}
	if iopts.NoTTY {
		argv = append(argv, "--no-tty")
	}
	return argv
}

// paneCommand is the shell command line tmux runs in a pane. tmux passes
// it to the user's shell, so the argv is quoted word by word. When msh
// fails, the pane stays open until Enter is pressed.
func paneCommand(argv []string) string {
	return shellquote.Join(argv) +
		`; status=$?; if [ "$status" -ne 0 ]; then printf '\nmsh exited with status %s; press Enter to close this pane. ' "$status"; read _; fi`
}

// absPath makes a path absolute so it works from any directory.
func absPath(p string) string {
	if p == "" {
		return p
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}
