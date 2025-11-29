package output

import (
	"bytes"
	"testing"
)

func TestTable(t *testing.T) {
	var buf bytes.Buffer
	err := Table(&buf, []string{"NAME", "ADDRESS"}, [][]string{
		{"web01", "192.0.2.10"},
		{"database01", "192.0.2.20"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "NAME        ADDRESS\n" +
		"web01       192.0.2.10\n" +
		"database01  192.0.2.20\n"
	if buf.String() != want {
		t.Errorf("Table() =\n%s\nwant\n%s", buf.String(), want)
	}
}

func TestKeyValues(t *testing.T) {
	var buf bytes.Buffer
	if err := KeyValues(&buf, []Pair{{"Name", "msh"}, {"Version", "1.0"}}); err != nil {
		t.Fatal(err)
	}
	want := "Name:     msh\nVersion:  1.0\n"
	if buf.String() != want {
		t.Errorf("KeyValues() = %q, want %q", buf.String(), want)
	}
}
