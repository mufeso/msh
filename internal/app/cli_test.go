package app

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"example.com/msh/internal/buildinfo"
	"example.com/msh/internal/exitcode"
)

// update rewrites golden files instead of comparing against them:
//
//	go test ./internal/app -run Golden -update
var update = flag.Bool("update", false, "rewrite golden files")

// runCLI runs msh with args and returns everything it produced.
func runCLI(t *testing.T, args ...string) (stdout, stderr string, code exitcode.Code) {
	t.Helper()
	var out, errOut bytes.Buffer
	c := Run(args, &out, &errOut)
	return out.String(), errOut.String(), exitcode.Code(c)
}

// golden compares got with testdata/<name>.golden.
func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden file (run with -update to create it): %v", err)
	}
	if got != string(want) {
		t.Errorf("output differs from %s\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}

func TestGoldenHelp(t *testing.T) {
	stdout, stderr, code := runCLI(t, "--help")
	if code != exitcode.OK || stderr != "" {
		t.Fatalf("code=%v stderr=%q", code, stderr)
	}
	golden(t, "help", stdout)
}

func TestVersionJSONRoundTrip(t *testing.T) {
	stdout, _, code := runCLI(t, "version", "--output", "json")
	if code != exitcode.OK {
		t.Fatalf("code = %v", code)
	}
	var got buildinfo.Info
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, stdout)
	}
	if got != buildinfo.Get() {
		t.Errorf("decoded %+v, want %+v", got, buildinfo.Get())
	}
}
