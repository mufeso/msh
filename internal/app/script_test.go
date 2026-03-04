package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestLargeScriptArrivesIntact(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	for b.Len() < 1<<20 {
		b.WriteString("# padding padding padding padding padding padding padding\n")
	}
	b.WriteString("echo end-of-large-script\n")
	script := writeScript(t, "large.sh", b.String())
	stdout, stderr, code := fleet.run(t, []string{"script", "run"}, "--host", "s1", "--file", script)
	if exitcode.Code(code) != exitcode.OK || !contains(stdout, "end-of-large-script") {
		t.Errorf("code = %d, stderr = %s, stdout tail = %q", code, stderr, stdout[max(0, len(stdout)-200):])
	}
}

func TestScriptCancellationClosesSessions(t *testing.T) {
	fleet := newTestFleet(t, 3, nil)
	script := writeScript(t, "slow.sh", "sleep 30\necho never\n")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	args := append([]string{"script", "run", "--all", "--file", script, "--command-timeout", "0s"}, fleet.args()...)
	start := time.Now()
	stdout, _, _ := runCLICtx(t, ctx, nil, args...)
	if time.Since(start) > 5*time.Second {
		t.Error("cancellation did not stop the scripts")
	}
	if contains(stdout, "never") {
		t.Error("script finished despite cancellation")
	}
	for _, s := range fleet.servers {
		if !s.WaitIdle(3 * time.Second) {
			t.Error("a remote session is still open")
		}
	}
}

func TestScriptCreatesNoLocalTempFiles(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp) // os.TempDir would point here
	script := writeScript(t, "x.sh", "echo hi\n")
	if _, _, code := fleet.run(t, []string{"script", "run"}, "--host", "s1", "--file", script); exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d", code)
	}
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("temporary files were created: %v", entries)
	}
}
