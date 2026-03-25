package output

import (
	"bytes"
	"encoding/csv"
	"testing"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/runner"
)

func TestCSVSink(t *testing.T) {
	var buf bytes.Buffer
	s := NewSink(&buf, CSV, SinkOptions{})
	s.Write(executor.Result{Server: "web01", Address: "192.0.2.10", Status: executor.StatusOK, Stdout: "a, \"b\"\nc\n",
		Started: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), Duration: 42 * time.Millisecond})
	s.Close(runner.Summary{Total: 1, OK: 1})

	records, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || records[0][0] != "server" {
		t.Fatalf("records = %q", records)
	}
	row := records[1]
	if row[0] != "web01" || row[2] != "ok" || row[4] != "42" || row[5] != "2026-01-02T03:04:05Z" || row[7] != "a, \"b\"\nc\n" {
		t.Errorf("row = %q", row)
	}
}

func TestCSVHeaderWithoutResults(t *testing.T) {
	var buf bytes.Buffer
	s := NewSink(&buf, CSV, SinkOptions{})
	s.Close(runner.Summary{})
	if buf.String() != "server,address,status,exit_code,duration_ms,started,error,stdout,stderr\n" {
		t.Errorf("got %q", buf.String())
	}
}

func TestWriteRows(t *testing.T) {
	type item struct {
		Name string `json:"name"`
	}
	headers := []string{"NAME"}
	rows := [][]string{{"a,b"}}
	items := []item{{"a,b"}}
	var c, j bytes.Buffer
	WriteRows(&c, CSV, headers, rows, items)
	WriteRows(&j, JSON, headers, rows, items)
	if c.String() != "NAME\n\"a,b\"\n" {
		t.Errorf("csv = %q", c.String())
	}
	if j.String() != "[\n  {\n    \"name\": \"a,b\"\n  }\n]\n" {
		t.Errorf("json = %q", j.String())
	}
}
