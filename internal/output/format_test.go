package output

import (
	"bytes"
	"testing"
)

func TestParseFormat(t *testing.T) {
	for _, s := range []string{"text", "json"} {
		f, err := ParseFormat(s)
		if err != nil || string(f) != s {
			t.Errorf("ParseFormat(%q) = %q, %v", s, f, err)
		}
	}
	if _, err := ParseFormat("yaml"); err == nil {
		t.Error("ParseFormat(yaml) = nil error, want error")
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	v := struct {
		Name string `json:"name"`
		Port int    `json:"port"`
	}{"web01", 22}
	if err := WriteJSON(&buf, v); err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"name\": \"web01\",\n  \"port\": 22\n}\n"
	if buf.String() != want {
		t.Errorf("WriteJSON() = %q, want %q", buf.String(), want)
	}
}
