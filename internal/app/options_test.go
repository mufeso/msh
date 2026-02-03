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

func TestTimeoutFlagsBecomeLayer(t *testing.T) {
	var opts commonOptions
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	opts.register(fs)
	if err := fs.Parse([]string{"--connect-timeout", "3s", "--timeout", "1m"}); err != nil {
		t.Fatal(err)
	}
	l := opts.settingsLayer()
	if l.ConnectTimeout == nil || l.ConnectTimeout.String() != "3s" {
		t.Errorf("ConnectTimeout = %v", l.ConnectTimeout)
	}
	if l.Timeout == nil || l.Timeout.String() != "1m0s" {
		t.Errorf("Timeout = %v", l.Timeout)
	}
	if l.CommandTimeout != nil {
		t.Error("unset --command-timeout leaked into the layer")
	}
}

func TestBadTimeoutFlagIsUsageError(t *testing.T) {
	_, _, code := runCLI(t, "hosts", "list", "--inventory", "testdata/inventory.json", "--connect-timeout", "0s")
	if code != exitcode.Usage {
		t.Errorf("code = %v, want usage", code)
	}
}

func TestParseInterspersed(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	v := fs.Bool("v", false, "")
	o := fs.String("o", "", "")
	pos, code, ok := parseInterspersed(fs, []string{"a", "-v", "b", "-o", "x", "--", "-c", "d"})
	if !ok || code != exitcode.OK {
		t.Fatalf("parse failed: %v", code)
	}
	if !*v || *o != "x" {
		t.Errorf("flags v=%v o=%q", *v, *o)
	}
	if strings.Join(pos, " ") != "a b -c d" {
		t.Errorf("positional = %q", pos)
	}
}
