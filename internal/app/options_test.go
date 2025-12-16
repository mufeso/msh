package app

import (
	"flag"
	"io"
	"strings"
	"testing"

	"example.com/msh/internal/exitcode"
)

func TestSettingsLayerOnlyContainsSetFlags(t *testing.T) {
	var opts commonOptions
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	opts.register(fs)
	if err := fs.Parse([]string{"--user", "ops", "--port", "2200"}); err != nil {
		t.Fatal(err)
	}
	l := opts.settingsLayer()
	if l.User == nil || *l.User != "ops" {
		t.Errorf("User = %v", l.User)
	}
	if l.Port == nil || *l.Port != 2200 {
		t.Errorf("Port = %v", l.Port)
	}
	if l.Output != nil || l.Inventory != nil {
		t.Errorf("unset flags leaked into layer: %+v", l)
	}
}

func TestVerboseGoesToStderr(t *testing.T) {
	stdout, stderr, code := runCLI(t, "hosts", "list", "--verbose", "--inventory", "testdata/inventory.json")
	if code != exitcode.OK {
		t.Fatalf("exit code = %v", code)
	}
	if !strings.Contains(stderr, "loading inventory testdata/inventory.json") {
		t.Errorf("stderr = %q, want verbose diagnostic", stderr)
	}
	if strings.Contains(stdout, "loading inventory") {
		t.Errorf("diagnostic leaked to stdout: %q", stdout)
	}
}

func TestAllWithHostIsUsageError(t *testing.T) {
	_, stderr, code := runCLI(t, "hosts", "list", "--all", "--host", "web01")
	if code != exitcode.Usage {
		t.Fatalf("exit code = %v, want %v", code, exitcode.Usage)
	}
	if !strings.Contains(stderr, "--all cannot be combined") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestInvalidOutputFlagIsUsageError(t *testing.T) {
	_, _, code := runCLI(t, "hosts", "list", "--inventory", "testdata/inventory.json", "--output", "xml")
	if code != exitcode.Usage {
		t.Fatalf("exit code = %v, want %v", code, exitcode.Usage)
	}
}
