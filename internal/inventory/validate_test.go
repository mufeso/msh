package inventory

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateAcceptsSample(t *testing.T) {
	if err := sampleInventory().Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
}

func TestValidateReportsAllProblems(t *testing.T) {
	inv := &Inventory{Servers: []Server{
		{Name: "", Address: "192.0.2.1"},
		{Name: "web01", Address: ""},
		{Name: "web01", Address: "192.0.2.2", Port: 70000},
		{Name: "bad name", Address: "192.0.2.3", Groups: []string{"ok", "not ok"}},
	}}

	err := inv.Validate()

	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("Validate() = %v, want *ValidationError", err)
	}
	wantSubstrings := []string{
		"servers[0]: name is required",
		`server "web01": address is required`,
		`server "web01": duplicate name`,
		"port 70000 out of range",
		`server "bad name": name must match`,
		`invalid group name "not ok"`,
	}
	msg := err.Error()
	for _, want := range wantSubstrings {
		if !strings.Contains(msg, want) {
			t.Errorf("error does not mention %q:\n%s", want, msg)
		}
	}
}

func TestValidateEmpty(t *testing.T) {
	err := (&Inventory{}).Validate()
	if err == nil || !strings.Contains(err.Error(), "no servers") {
		t.Errorf("Validate() = %v", err)
	}
}
