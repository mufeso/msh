package task

import (
	"strings"
	"testing"
)

func TestNewSet(t *testing.T) {
	s, err := NewSet(map[string]Task{
		"uptime": {Argv: []string{"uptime"}, Aliases: []string{"up"}},
		"disk":   {Command: "df -h"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if tk, ok := s.Get("up"); !ok || tk.Name != "uptime" {
		t.Errorf("alias lookup failed: %+v", tk)
	}
	if list := s.List(); len(list) != 2 || list[0].Name != "disk" {
		t.Errorf("List() = %v", list)
	}
	if _, ok := s.Get("nope"); ok {
		t.Error("unknown task found")
	}
}

func TestNewSetProblems(t *testing.T) {
	_, err := NewSet(map[string]Task{
		"a":        {Command: "x", Aliases: []string{"z"}},
		"b":        {Command: "y", Aliases: []string{"z", "a"}},
		"bad name": {Command: "x"},
		"empty":    {},
	})
	if err == nil {
		t.Fatal("NewSet accepted invalid tasks")
	}
	for _, want := range []string{`alias "z" is already used`, `alias "a" is also a task name`, `"bad name"`, `task "empty": exactly one`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in:\n%v", want, err)
		}
	}
}
