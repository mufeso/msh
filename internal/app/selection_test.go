package app

import "testing"

func TestSelectionValidate(t *testing.T) {
	s := selectionOptions{all: true}
	if err := s.validate(); err != nil {
		t.Errorf("--all alone: %v", err)
	}
	s.hosts = append(s.hosts, "web01")
	if err := s.validate(); err == nil {
		t.Error("--all with --host: want error")
	}
	if !(&selectionOptions{}).isEmpty() {
		t.Error("zero selection should be empty")
	}
}
