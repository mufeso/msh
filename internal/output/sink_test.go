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
		"db01: ERROR: [failed: db01: exit 2]\n" +
		"x: ERROR: [error: x: refused]\n" +
		"--- 3 servers: 1 ok, 1 failed, 1 error in 0s\n" +
		"--- problems: db01, x\n"
	if buf.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestPrefixSinkPadsNames(t *testing.T) {
	var buf bytes.Buffer
	s := NewSink(&buf, Text, SinkOptions{Prefix: true, NameWidth: NameWidth([]string{"web01", "server01"})})
	s.Write(executor.Result{Server: "web01", Status: executor.StatusOK, Stdout: "/boot\n"})
	s.Write(executor.Result{Server: "server01", Status: executor.StatusOK, Stdout: "/boot\n"})
	s.Write(executor.Result{Server: "web01", Status: executor.StatusOK})
	s.Write(executor.Result{Server: "web01", Status: executor.StatusFailed, ExitCode: 3, Stderr: "no\n"})
	want := "web01___: /boot\nserver01: /boot\nweb01___: [ok]\nweb01___ [stderr]: no\nweb01___: ERROR: [failed: web01: exit 3]\n"
	if buf.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestSummaryIsAligned(t *testing.T) {
	var buf bytes.Buffer
	s := NewSink(&buf, Text, SinkOptions{Prefix: true, NameWidth: NameWidth([]string{"web01", "192.0.2.20"})})
	s.Write(executor.Result{Server: "web01", Status: executor.StatusOK, Stdout: "x\n"})
	s.Close(runner.Summary{Total: 2, OK: 1, Failed: 1, Problems: []string{"192.0.2.20"}})
	want := "web01_____: x\n" +
		"##########: 2 servers: 1 ok, 1 failed in 0s\n" +
		"##########: problems: 192.0.2.20\n"
	if buf.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestErrorLine(t *testing.T) {
	tests := []struct {
		r    executor.Result
		want string
	}{
		{executor.Result{Server: "a", Status: executor.StatusFailed, ExitCode: 2}, "ERROR: [failed: a: exit 2]"},
		{executor.Result{Server: "a", Status: executor.StatusError, Err: errors.New("a: DNS lookup failed")}, "ERROR: [error: a: DNS lookup failed]"},
		{executor.Result{Server: "a", Status: executor.StatusTimeout, Err: errors.New("timed out")}, "ERROR: [timeout: a: timed out]"},
	}
	for _, tt := range tests {
		if got := errorLine(tt.r); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestSkippedIsNotAnError(t *testing.T) {
	var buf bytes.Buffer
	s := NewSink(&buf, Text, SinkOptions{Prefix: true})
	s.Write(executor.Result{Server: "a", Status: executor.StatusSkipped, Err: errors.New("not approved")})
	if got := buf.String(); got != "a: [skipped: not approved]\n" {
		t.Errorf("got %q", got)
	}
}

func TestPadName(t *testing.T) {
	tests := []struct {
		name  string
		width int
		want  string
	}{
		{"a", 3, "a__"},
		{"abc", 3, "abc"},
		{"abcd", 3, "abcd"},
		{"é", 2, "é_"},
		{"a", 0, "a"},
	}
	for _, tt := range tests {
		if got := PadName(tt.name, tt.width); got != tt.want {
			t.Errorf("PadName(%q, %d) = %q", tt.name, tt.width, got)
		}
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
