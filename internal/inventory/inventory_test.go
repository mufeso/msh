package inventory

import (
	"slices"
	"testing"
)

// Test data uses documentation addresses (RFC 5737 / RFC 3849) and
// example.com names, never real hosts.
func sampleInventory() *Inventory {
	return &Inventory{
		Servers: []Server{
			{Name: "web01", Address: "192.0.2.10", Port: 22, Groups: []string{"web", "production"},
				Tags: map[string]string{"environment": "production"}},
			{Name: "web02", Address: "2001:db8::2", Port: 2222, Groups: []string{"web"}},
			{Name: "db01", Address: "db01.example.com", Port: 22, Groups: []string{"database"}},
		},
	}
}

func TestHostPort(t *testing.T) {
	inv := sampleInventory()
	tests := map[string]string{
		"web01": "192.0.2.10:22",
		"web02": "[2001:db8::2]:2222",
		"db01":  "db01.example.com:22",
	}
	for name, want := range tests {
		s, ok := inv.Find(name)
		if !ok {
			t.Fatalf("Find(%q) not found", name)
		}
		if got := s.HostPort(); got != want {
			t.Errorf("%s.HostPort() = %q, want %q", name, got, want)
		}
	}
}

func TestInGroupAndTags(t *testing.T) {
	web01, _ := sampleInventory().Find("web01")
	if !web01.InGroup("web") || web01.InGroup("database") {
		t.Error("InGroup gave wrong answer")
	}
	if !web01.HasTags(map[string]string{"environment": "production"}) {
		t.Error("HasTags(environment=production) = false")
	}
	if web01.HasTags(map[string]string{"environment": "staging"}) {
		t.Error("HasTags(environment=staging) = true")
	}
	if !web01.HasTags(nil) {
		t.Error("HasTags(nil) should match")
	}
}

func TestFindMissingAndNames(t *testing.T) {
	inv := sampleInventory()
	if _, ok := inv.Find("nope"); ok {
		t.Error("Find(nope) found something")
	}
	if got, want := inv.Names(), []string{"web01", "web02", "db01"}; !slices.Equal(got, want) {
		t.Errorf("Names() = %v, want %v", got, want)
	}
}
