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
		{name: "no arguments", args: nil, wantCode: exitOK, wantStdout: buildinfo.Banner()},
		{name: "short help", args: []string{"-h"}, wantCode: exitOK, wantStdout: "Usage:"},
		{name: "long help", args: []string{"--help"}, wantCode: exitOK, wantStdout: "Usage:"},
		{name: "version", args: []string{"--version"}, wantCode: exitOK, wantStdout: buildinfo.Version},
		{name: "unknown argument", args: []string{"bogus"}, wantCode: exitUsage, wantStderr: `unknown argument "bogus"`},
		{name: "unknown flag", args: []string{"--bogus"}, wantCode: exitUsage, wantStderr: "--help"},
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

// checkOutput is a test helper. t.Helper makes failures point at the
// caller's line instead of this function.
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
