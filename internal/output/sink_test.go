package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/runner"
)

func TestPrefixSink(t *testing.T) {
	var buf bytes.Buffer
	s := NewSink(&buf, Text, SinkOptions{Prefix: true})
	s.Write(executor.Result{Server: "web01", Status: executor.StatusOK, Stdout: "a\nb\n", Stderr: "warn\n"})
	s.Write(executor.Result{Server: "db01", Status: executor.StatusFailed, ExitCode: 2})
	s.Write(executor.Result{Server: "x", Status: executor.StatusError, Err: errors.New("refused")})
	s.Close(runner.Summary{Total: 3, OK: 1, Failed: 1, Error: 1, Problems: []string{"db01", "x"}})
	want := "web01: a\nweb01: b\nweb01 [stderr]: warn\n" +
		"db01: [failed: exit 2]\n" +
		"x: [error: refused]\n" +
		"--- 3 servers: 1 ok, 1 failed, 1 error in 0s\n" +
		"--- problems: db01, x\n"
	if buf.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestTextSinkSingleResultHasNoSummary(t *testing.T) {
	var buf bytes.Buffer
	s := NewSink(&buf, Text, SinkOptions{})
	s.Write(executor.Result{Server: "a", Status: executor.StatusOK, Stdout: "x\n"})
	s.Close(runner.Summary{Total: 1, OK: 1})
	if bytes.Contains(buf.Bytes(), []byte("---")) {
		t.Errorf("unexpected summary: %q", buf.String())
	}
}

func TestJSONSinkDocument(t *testing.T) {
	var buf bytes.Buffer
	s := NewSink(&buf, JSON, SinkOptions{})
	s.Write(executor.Result{Server: "a", Status: executor.StatusOK})
	s.Close(runner.Summary{Total: 1, OK: 1, Duration: 1500 * time.Millisecond})
	var doc RunView
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Results) != 1 || doc.Summary.Total != 1 || doc.Summary.DurationMS != 1500 || doc.Summary.Problems == nil {
		t.Errorf("doc = %+v", doc)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"problems": []`)) {
		t.Errorf("problems must be [] not null:\n%s", buf.String())
	}
}
