package inventory

import (
	"errors"
	"slices"
	"testing"
)

func names(servers []Server) []string {
	out := make([]string, 0, len(servers))
	for _, s := range servers {
		out = append(out, s.Name)
	}
	return out
}

func TestSelectByName(t *testing.T) {
	got, err := sampleInventory().Select(Selector{Hosts: []string{"db01"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "db01" {
		t.Errorf("Select = %v", names(got))
	}
}

func TestSelectUnknownHost(t *testing.T) {
	_, err := sampleInventory().Select(Selector{Hosts: []string{"nope"}})
	if !errors.Is(err, ErrUnknownHost) {
		t.Errorf("err = %v, want ErrUnknownHost", err)
	}
}

func TestSelectMultipleDeduplicated(t *testing.T) {
	got, err := sampleInventory().Select(Selector{Hosts: []string{"db01", "web01", "db01"}})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"db01", "web01"}; !slices.Equal(names(got), want) {
		t.Errorf("Select = %v, want %v", names(got), want)
	}
}
