package app

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"example.com/msh/internal/buildinfo"
	"example.com/msh/internal/exitcode"
)

// update rewrites golden files instead of comparing against them:
//
//	go test ./internal/app -run Golden -update
var update = flag.Bool("update", false, "rewrite golden files")

// runCLI runs msh with args in an isolated environment that only
// contains HOME (a fresh temporary directory).
func runCLI(t *testing.T, args ...string) (stdout, stderr string, code exitcode.Code) {
	t.Helper()
	return runCLIEnv(t, nil, args...)
}

// runCLIEnv runs msh with the given environment variables plus an
// isolated HOME. The real environment is never consulted, so a developer's
// own ~/.config/msh/config.yaml cannot change test results.
func runCLIEnv(t *testing.T, env map[string]string, args ...string) (stdout, stderr string, code exitcode.Code) {
	t.Helper()
	return runCLICtx(t, context.Background(), env, args...)
}

// runCLICtx is runCLIEnv with an explicit root context, used to simulate
// CTRL+C.
func runCLICtx(t *testing.T, ctx context.Context, env map[string]string, args ...string) (stdout, stderr string, code exitcode.Code) {
	t.Helper()
	vars := map[string]string{"HOME": t.TempDir()}
	for k, v := range env {
		vars[k] = v
	}
	var out, errOut bytes.Buffer
	c := &cli{
		stdout: &out,
		stderr: &errOut,
		lookupEnv: func(k string) (string, bool) {
			v, ok := vars[k]
			return v, ok
		},
	}
	c.root = ctx
	c.term = &testTerminal{in: strings.NewReader(input(ctx)), out: &out, errOut: &errOut, tty: true}
	result := c.finish(c.run(args))
	return out.String(), errOut.String(), result
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

// testTerminal is a fake Terminal backed by buffers.
type testTerminal struct {
	in     io.Reader
	out    io.Writer
	errOut io.Writer
	tty    bool

	mu       sync.Mutex
	raw      int // MakeRaw calls
	restored int // restore calls
}

func (t *testTerminal) Stdin() io.Reader  { return t.in }
func (t *testTerminal) Stdout() io.Writer { return t.out }
func (t *testTerminal) Stderr() io.Writer { return t.errOut }
func (t *testTerminal) IsTerminal() bool  { return t.tty }
func (t *testTerminal) Size() (int, int, error) {
	return 100, 30, nil
}
func (t *testTerminal) Resizes() (<-chan struct{}, func()) {
	ch := make(chan struct{})
	return ch, func() { close(ch) }
}

func (t *testTerminal) MakeRaw() (func(), error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.raw++
	return func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		t.restored++
	}, nil
}

// inputKey carries test terminal input in a context.
type inputKey struct{}

// withInput returns a context whose test terminal reads input.
func withInput(ctx context.Context, input string) context.Context {
	return context.WithValue(ctx, inputKey{}, input)
}

func input(ctx context.Context) string {
	s, _ := ctx.Value(inputKey{}).(string)
	return s
}
