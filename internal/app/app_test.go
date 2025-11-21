package app

import (
	"bytes"
	"strings"
	"testing"

	"example.com/msh/internal/buildinfo"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string // substring expected on stdout ("" = must be empty)
		wantStderr string // substring expected on stderr ("" = must be empty)
	}{
		{name: "no arguments", args: nil, wantCode: exitOK, wantStdout: "Commands:"},
		{name: "long help", args: []string{"--help"}, wantCode: exitOK, wantStdout: "version"},
		{name: "version flag", args: []string{"--version"}, wantCode: exitOK, wantStdout: buildinfo.Version},
		{name: "version command", args: []string{"version"}, wantCode: exitOK, wantStdout: buildinfo.Version},
		{name: "version extra arg", args: []string{"version", "x"}, wantCode: exitUsage, wantStderr: "no arguments"},
		{name: "help command", args: []string{"help"}, wantCode: exitOK, wantStdout: "Commands:"},
		{name: "help for version", args: []string{"help", "version"}, wantCode: exitOK, wantStdout: "Usage: msh version"},
		{name: "help unknown", args: []string{"help", "nope"}, wantCode: exitUsage, wantStderr: `unknown command "nope"`},
		{name: "hosts without subcommand", args: []string{"hosts"}, wantCode: exitUsage, wantStderr: "Subcommands:"},
		{name: "hosts list", args: []string{"hosts", "list"}, wantCode: exitOK, wantStdout: "No inventory"},
		{name: "hosts check", args: []string{"hosts", "check"}, wantCode: exitOK, wantStdout: "nothing to check"},
		{name: "hosts help", args: []string{"hosts", "--help"}, wantCode: exitOK, wantStdout: "Subcommands:"},
		{name: "hosts unknown", args: []string{"hosts", "nope"}, wantCode: exitUsage, wantStderr: `unknown hosts subcommand "nope"`},
		{name: "unknown command", args: []string{"bogus"}, wantCode: exitUsage, wantStderr: `unknown command "bogus"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(tt.args, &stdout, &stderr)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			checkOutput(t, "stdout", stdout.String(), tt.wantStdout)
			checkOutput(t, "stderr", stderr.String(), tt.wantStderr)
		})
	}
}

func TestUsageListsEveryCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	Run([]string{"--help"}, &stdout, &stderr)
	for _, cmd := range commands() {
		if !strings.Contains(stdout.String(), cmd.name) {
			t.Errorf("usage does not mention command %q", cmd.name)
		}
	}
}

func checkOutput(t *testing.T, stream, got, want string) {
	t.Helper()
	if want == "" {
		if got != "" {
			t.Errorf("%s = %q, want empty", stream, got)
		}
		return
	}
	if !strings.Contains(got, want) {
		t.Errorf("%s = %q, want it to contain %q", stream, got, want)
	}
}
