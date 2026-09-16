package app

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/signals"
	"example.com/msh/internal/sshtest"
)

func TestValidateLogPath(t *testing.T) {
	allowed := []string{"/var/log"}
	good := []string{"/var/log/messages", "/var/log/nginx/access.log"}
	bad := []string{"", "var/log/messages", "/var/log/../etc/shadow", "/var/log", "/var/logs/x", "/etc/passwd", "/var/log/x\n", "/var/log//x"}
	for _, p := range good {
		if err := validateLogPath(p, allowed); err != nil {
			t.Errorf("%q rejected: %v", p, err)
		}
	}
	for _, p := range bad {
		if err := validateLogPath(p, allowed); err == nil {
			t.Errorf("%q accepted", p)
		}
	}
}

func TestLogsCommand(t *testing.T) {
	fleet := newTestFleet(t, 1, sshtest.Static("line\n", "", 0))
	_, stderr, code := fleet.run(t, []string{"logs"}, "--host", "s1", "--file", "/var/log/messages", "--lines", "20")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if cmds := fleet.servers[0].Commands(); len(cmds) != 1 || cmds[0] != "tail -n 20 -- /var/log/messages" {
		t.Errorf("remote command = %q", cmds)
	}
	_, stderr, code = fleet.run(t, []string{"logs"}, "--host", "s1", "--file", "/etc/shadow")
	if exitcode.Code(code) != exitcode.Usage || !contains(stderr, "outside allowed_log_paths") {
		t.Errorf("code = %d, stderr = %q", code, stderr)
	}
	_, _, code = fleet.run(t, []string{"logs"}, "--host", "s1", "--file", "/var/log/messages", "--lines", "0")
	if exitcode.Code(code) != exitcode.Usage {
		t.Errorf("--lines 0: code = %d", code)
	}
}

func TestLogsFollowStreamsUntilCtrlC(t *testing.T) {
	streaming := func(ctx context.Context, e *sshtest.Exec) int {
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return 143
			case <-time.After(20 * time.Millisecond):
				io.WriteString(e.Stdout, "event\n")
			}
		}
	}
	fleet := newTestFleet(t, 2, streaming)
	sigs := make(chan os.Signal, 1)
	ctx, stop := signals.Watch(context.Background(), sigs, io.Discard, func(int) {})
	defer stop()
	go func() {
		time.Sleep(300 * time.Millisecond)
		sigs <- os.Interrupt
	}()
	args := append([]string{"logs", "--all", "--file", "/var/log/messages", "--follow"}, fleet.args()...)
	stdout, _, code := runCLICtx(t, ctx, nil, args...)
	if code != exitcode.Canceled {
		t.Errorf("code = %v, want canceled", code)
	}
	for _, want := range []string{"s1_: event\n", "s2_: event\n"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
	if cmds := fleet.servers[0].Commands(); len(cmds) != 1 || cmds[0] != "tail -n 50 -F -- /var/log/messages" {
		t.Errorf("remote command = %q", cmds)
	}
}
