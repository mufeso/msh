package app

import (
	"bytes"
	"strings"
	"testing"

	"example.com/msh/internal/exitcode"
)

func TestCommonOptionsValidate(t *testing.T) {
	for _, out := range []string{"text", "json"} {
		o := commonOptions{outputName: out}
		if err := o.validate(); err != nil {
			t.Errorf("validate(%q) = %v, want nil", out, err)
		}
	}
	o := commonOptions{outputName: "xml"}
	if err := o.validate(); err == nil {
		t.Error("validate(xml) = nil, want error")
	}
}

func TestVerboseGoesToStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"hosts", "list", "--verbose", "--inventory", "testdata/inventory.json"}, &stdout, &stderr)
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "loading inventory testdata/inventory.json") {
		t.Errorf("stderr = %q, want verbose diagnostic", stderr.String())
	}
	if strings.Contains(stdout.String(), "loading inventory") {
		t.Errorf("diagnostic leaked to stdout: %q", stdout.String())
	}
}

func TestAllWithHostIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"hosts", "list", "--all", "--host", "web01"}, &stdout, &stderr)
	if exitcode.Code(code) != exitcode.Usage {
		t.Fatalf("exit code = %d, want %d", code, exitcode.Usage)
	}
	if !strings.Contains(stderr.String(), "--all cannot be combined") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestInvalidOutputIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"hosts", "list", "--output", "xml"}, &stdout, &stderr)
	if exitcode.Code(code) != exitcode.Usage {
		t.Fatalf("exit code = %d, want %d", code, exitcode.Usage)
	}
}
