package app

import (
	"bytes"
	"strings"
	"testing"

	"example.com/msh/internal/buildinfo"
	"example.com/msh/internal/exitcode"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   exitcode.Code
		wantStdout string // substring expected on stdout ("" = must be empty)
		wantStderr string // substring expected on stderr ("" = must be empty)
	}{
		{name: "no arguments", args: nil, wantCode: exitcode.OK, wantStdout: "Commands:"},
		{name: "long help", args: []string{"--help"}, wantCode: exitcode.OK, wantStdout: "version"},
		{name: "version flag", args: []string{"--version"}, wantCode: exitcode.OK, wantStdout: buildinfo.Version},
		{name: "version command", args: []string{"version"}, wantCode: exitcode.OK, wantStdout: buildinfo.Version},
		{name: "version extra arg", args: []string{"version", "x"}, wantCode: exitcode.Usage, wantStderr: "no arguments"},
		{name: "version short", args: []string{"version", "--short"}, wantCode: exitcode.OK, wantStdout: buildinfo.Version + "\n"},
		{name: "version help", args: []string{"version", "--help"}, wantCode: exitcode.OK, wantStderr: "-short"},
		{name: "version bad flag", args: []string{"version", "--nope"}, wantCode: exitcode.Usage, wantStderr: "flag provided but not defined"},
		{name: "hosts list help", args: []string{"hosts", "list", "-h"}, wantCode: exitcode.OK, wantStderr: "Usage: msh hosts list"},
		{name: "version json", args: []string{"version", "--output", "json"}, wantCode: exitcode.OK, wantStdout: `"go_version":`},
		{name: "version bad output", args: []string{"version", "--output", "xml"}, wantCode: exitcode.Usage, wantStderr: "invalid output format"},
		{name: "help command", args: []string{"help"}, wantCode: exitcode.OK, wantStdout: "Commands:"},
		{name: "help for version", args: []string{"help", "version"}, wantCode: exitcode.OK, wantStdout: "Usage: msh version"},
		{name: "help unknown", args: []string{"help", "nope"}, wantCode: exitcode.Usage, wantStderr: `unknown command "nope"`},
		{name: "hosts without subcommand", args: []string{"hosts"}, wantCode: exitcode.Usage, wantStderr: "Subcommands:"},
		{name: "hosts list without inventory", args: []string{"hosts", "list"}, wantCode: exitcode.Config, wantStderr: "no inventory file"},
		{name: "hosts list", args: []string{"hosts", "list", "--inventory", "testdata/inventory.json"}, wantCode: exitcode.OK, wantStdout: "web02  192.0.2.11        2222  admin    web,staging"},
		{name: "hosts list json", args: []string{"hosts", "list", "--inventory", "testdata/inventory.json", "--output", "json"}, wantCode: exitcode.OK, wantStdout: `"name": "db01"`},
		{name: "hosts list missing file", args: []string{"hosts", "list", "--inventory", "testdata/nope.json"}, wantCode: exitcode.IO, wantStderr: "no such file"},
		{name: "hosts list invalid inventory", args: []string{"hosts", "list", "--inventory", "testdata/invalid.json"}, wantCode: exitcode.Config, wantStderr: "duplicate name"},
		{name: "hosts check", args: []string{"hosts", "check"}, wantCode: exitcode.OK, wantStdout: "nothing to check"},
		{name: "hosts help", args: []string{"hosts", "--help"}, wantCode: exitcode.OK, wantStdout: "Subcommands:"},
		{name: "hosts unknown", args: []string{"hosts", "nope"}, wantCode: exitcode.Usage, wantStderr: `unknown hosts subcommand "nope"`},
		{name: "typo suggestion", args: []string{"verison"}, wantCode: exitcode.Usage, wantStderr: `Did you mean "version"?`},
		{name: "unknown command", args: []string{"bogus"}, wantCode: exitcode.Usage, wantStderr: `unknown command "bogus"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(tt.args, &stdout, &stderr)
			if exitcode.Code(code) != tt.wantCode {
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
