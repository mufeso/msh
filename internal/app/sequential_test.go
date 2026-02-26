package app

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
	"example.com/msh/internal/sshtest"
)

func TestSequentialIntegration(t *testing.T) {
	failing := func(_ context.Context, e *sshtest.Exec) int {
		io.WriteString(e.Stderr, "disk full\n")
		return 3
	}
	fleet := newFleet(t, fleetOptions{servers: 5, handlers: map[int]sshtest.Handler{
		2: failing,
		4: sshtest.Blocking(""),
	}})
	fleet.servers[2].Close() // s3 is unreachable

	stdout, stderr, code := fleet.run(t, []string{"exec"}, "--all", "--output", "json", "--command-timeout", "300ms", "--command", "echo ok")
	if exitcode.Code(code) != exitcode.Timeout {
		t.Errorf("code = %d, want timeout (it outranks connection and failure)", code)
	}
	var run output.RunView
	if err := json.Unmarshal([]byte(stdout), &run); err != nil {
		t.Fatalf("invalid JSON: %v\n%s\n%s", err, stdout, stderr)
	}
	want := []struct {
		server   string
		status   string
		exitCode int
	}{
		{"s1", "ok", 0},
		{"s2", "failed", 3},
		{"s3", "error", -1},
		{"s4", "timeout", -1},
		{"s5", "ok", 0},
	}
	if len(run.Results) != len(want) {
		t.Fatalf("results = %d", len(run.Results))
	}
	for i, w := range want {
		r := run.Results[i] // sequential mode keeps selection order
		if r.Server != w.server || r.Status != w.status || r.ExitCode != w.exitCode {
			t.Errorf("result %d = %s/%s/%d, want %s/%s/%d", i, r.Server, r.Status, r.ExitCode, w.server, w.status, w.exitCode)
		}
	}
	if run.Results[1].Stderr != "disk full\n" {
		t.Errorf("stderr of s2 = %q", run.Results[1].Stderr)
	}
	s := run.Summary
	if s.Total != 5 || s.OK != 2 || s.Failed != 1 || s.Error != 1 || s.Timeout != 1 {
		t.Errorf("summary = %+v", s)
	}
	if stderr != "" {
		t.Errorf("diagnostics must not be mixed into stdout; stderr = %q", stderr)
	}
}
