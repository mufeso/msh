package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"example.com/msh/internal/executor"
)

var sampleResults = []executor.Result{
	{Server: "web01", Address: "192.0.2.10", Status: executor.StatusOK, ExitCode: 0,
		Stdout: "Linux web01\n", Started: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), Duration: 120 * time.Millisecond},
	{Server: "db01", Address: "192.0.2.20", Status: executor.StatusError, ExitCode: -1,
		Err: errors.New("connection refused"), Duration: 5 * time.Millisecond},
}

func TestWriteResultsText(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteResults(&buf, Text, sampleResults); err != nil {
		t.Fatal(err)
	}
	want := "=== web01: ok (exit 0, 120ms)\nLinux web01\n" +
		"=== db01: error (5ms)\nERROR: [error: db01: connection refused]\n"
	if buf.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestWriteResultsJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteResults(&buf, JSON, sampleResults); err != nil {
		t.Fatal(err)
	}
	var views []ResultView
	if err := json.Unmarshal(buf.Bytes(), &views); err != nil {
		t.Fatal(err)
	}
	if views[0].DurationMS != 120 || views[0].Status != "ok" || views[1].Error != "connection refused" {
		t.Errorf("views = %+v", views)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"started": "2026-01-02T03:04:05Z"`)) {
		t.Errorf("started not RFC 3339:\n%s", buf.String())
	}
}

func TestTruncationMarker(t *testing.T) {
	var buf bytes.Buffer
	r := executor.Result{Server: "a", Status: executor.StatusOK, Stdout: "abc", StdoutTruncated: true}
	if err := WriteResultText(&buf, r); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("abc\n[output truncated]\n")) {
		t.Errorf("got %q", buf.String())
	}
}
