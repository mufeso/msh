package decode

import (
	"strings"
	"testing"
)

type sample struct {
	Name  string   `json:"name" yaml:"name"`
	Port  int      `json:"port" yaml:"port"`
	Items []string `json:"items" yaml:"items"`
}

func TestReaderYAMLAndJSON(t *testing.T) {
	inputs := map[Format]string{
		YAML: "name: web01\nport: 22\nitems: [a, b]\n",
		JSON: `{"name":"web01","port":22,"items":["a","b"]}`,
	}
	for format, in := range inputs {
		var got sample
		if err := Reader(strings.NewReader(in), format, &got); err != nil {
			t.Fatalf("format %d: %v", format, err)
		}
		if got.Name != "web01" || got.Port != 22 || len(got.Items) != 2 {
			t.Errorf("format %d: got %+v", format, got)
		}
	}
}

func TestReaderRejectsUnknownFields(t *testing.T) {
	var v sample
	if err := Reader(strings.NewReader("name: x\nnmae: y\n"), YAML, &v); err == nil {
		t.Error("YAML unknown field accepted")
	}
	if err := Reader(strings.NewReader(`{"nmae":"y"}`), JSON, &v); err == nil {
		t.Error("JSON unknown field accepted")
	}
}

func TestReaderEmptyYAML(t *testing.T) {
	var v sample
	if err := Reader(strings.NewReader(""), YAML, &v); err == nil {
		t.Error("empty YAML accepted")
	}
}

func TestFormatForPath(t *testing.T) {
	tests := map[string]Format{"a.json": JSON, "a.JSON": JSON, "a.yaml": YAML, "a.yml": YAML, "a": YAML}
	for path, want := range tests {
		if got := FormatForPath(path); got != want {
			t.Errorf("FormatForPath(%q) = %d, want %d", path, got, want)
		}
	}
}
