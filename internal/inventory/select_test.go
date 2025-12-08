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

func TestSelect(t *testing.T) {
	tests := []struct {
		name    string
		sel     Selector
		want    []string
		wantErr error
	}{
		{name: "one host", sel: Selector{Hosts: []string{"db01"}}, want: []string{"db01"}},
		{name: "hosts keep order and dedupe", sel: Selector{Hosts: []string{"db01", "web01", "db01"}}, want: []string{"db01", "web01"}},
		{name: "all", sel: Selector{All: true}, want: []string{"web01", "web02", "db01"}},
		{name: "group", sel: Selector{Groups: []string{"web"}}, want: []string{"web01", "web02"}},
		{name: "host then group, deduped", sel: Selector{Hosts: []string{"web02"}, Groups: []string{"web"}}, want: []string{"web02", "web01"}},
		{name: "tag only", sel: Selector{Tags: map[string]string{"environment": "production"}}, want: []string{"web01"}},
		{name: "group filtered by tag", sel: Selector{Groups: []string{"web"}, Tags: map[string]string{"environment": "production"}}, want: []string{"web01"}},
		{name: "unknown host", sel: Selector{Hosts: []string{"nope"}}, wantErr: ErrUnknownHost},
		{name: "unknown group", sel: Selector{Groups: []string{"nope"}}, wantErr: ErrUnknownGroup},
		{name: "no match", sel: Selector{Tags: map[string]string{"environment": "nope"}}, wantErr: ErrNoMatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sampleInventory().Select(tt.sel)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(names(got), tt.want) {
				t.Errorf("Select = %v, want %v", names(got), tt.want)
			}
		})
	}
}

func TestSelectAllReturnsCopy(t *testing.T) {
	inv := sampleInventory()
	got, _ := inv.Select(Selector{All: true})
	got[0].Name = "changed"
	if inv.Servers[0].Name == "changed" {
		t.Error("Select(All) returned the inventory's own slice")
	}
}
