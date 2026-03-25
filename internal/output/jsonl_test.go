package output

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/runner"
)

func TestJSONLSink(t *testing.T) {
	var buf bytes.Buffer
	s := NewSink(&buf, JSONL, SinkOptions{})
	s.Write(executor.Result{Server: "a", Status: executor.StatusOK, Stdout: "x\ny\n"})
	// The first line must be complete before the run ends.
	if bytes.Count(buf.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("result not streamed: %q", buf.String())
	}
	s.Write(executor.Result{Server: "b", Status: executor.StatusFailed, ExitCode: 2})
	s.Close(runner.Summary{Total: 2, OK: 1, Failed: 1, Problems: []string{"b"}})

	var types []string
	sc := bufio.NewScanner(&buf)
	for sc.Scan() {
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatalf("line is not JSON: %q", sc.Text())
		}
		types = append(types, m["type"].(string))
	}
	if len(types) != 3 || types[0] != "result" || types[2] != "summary" {
		t.Errorf("types = %v", types)
	}
}
