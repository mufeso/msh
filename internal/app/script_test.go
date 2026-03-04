package app

import (
	"os"
	"path/filepath"
	"testing"

	"example.com/msh/internal/exitcode"
)

func writeScript(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestShebangInterpreter(t *testing.T) {
	tests := map[string]string{
		"#!/bin/bash\necho":            "bash",
		"#!/usr/bin/env python3\nx":    "python3",
		"#!/usr/bin/env -S bash -e\nx": "bash",
		"#! /bin/sh\n":                 "sh",
		"echo no shebang\n":            "",
		"#!\n":                         "",
	}
	for in, want := range tests {
		if got := shebangInterpreter([]byte(in)); got != want {
			t.Errorf("shebangInterpreter(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestScriptRun(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	script := writeScript(t, "check.sh", "#!/bin/sh\necho \"checking $(echo ok)\"\n")
	stdout, stderr, code := fleet.run(t, []string{"script", "run"}, "--all", "--file", script)
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if !contains(stdout, "checking ok") {
		t.Errorf("stdout = %q", stdout)
	}
	if cmds := fleet.servers[0].Commands(); len(cmds) != 1 || cmds[0] != "sh -s --" {
		t.Errorf("remote command = %q: the script must travel on stdin, not in the command", cmds)
	}
}

func TestScriptRunErrors(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	if _, _, code := fleet.run(t, []string{"script", "run"}, "--host", "s1", "--file", filepath.Join(t.TempDir(), "missing.sh")); exitcode.Code(code) != exitcode.IO {
		t.Errorf("missing file: code = %d, want IO", code)
	}
	ruby := writeScript(t, "x.rb", "#!/usr/bin/env ruby\nputs 1\n")
	if _, _, code := fleet.run(t, []string{"script", "run"}, "--host", "s1", "--file", ruby); exitcode.Code(code) != exitcode.Usage {
		t.Errorf("unsupported interpreter: code = %d, want usage", code)
	}
	if _, _, code := fleet.run(t, []string{"script", "run"}, "--host", "s1", "--file", t.TempDir()); exitcode.Code(code) != exitcode.IO {
		t.Errorf("directory: code = %d, want IO", code)
	}
}

func TestScriptArguments(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	script := writeScript(t, "args.sh", "printf '[%s]\\n' \"$@\"\n")
	stdout, stderr, code := fleet.run(t, []string{"script", "run"}, "--host", "s1", "--interpreter", "sh", "--file", script,
		"--arg", "two words", "--arg", "$(id)", "--arg", "a,b")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if !contains(stdout, "[two words]\n[$(id)]\n[a,b]\n") {
		t.Errorf("stdout = %q", stdout)
	}
	if cmds := fleet.servers[0].Commands(); cmds[0] != `sh -s -- 'two words' '$(id)' a,b` {
		t.Errorf("remote command = %q", cmds[0])
	}
}
