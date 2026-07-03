package pkgs

import (
	"fmt"
	"regexp"
	"strings"
)

// Check statuses.
const (
	StatusOK      = "ok"
	StatusMissing = "missing"
)

var specName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9+._:-]*$`)

// Spec is a requirement on a package.
type Spec struct {
	Name string
}

// String returns the requirement as text.
func (s Spec) String() string { return s.Name }

// ParseSpec parses a requirement.
func ParseSpec(text string) (Spec, error) {
	name := strings.TrimSpace(text)
	if !specName.MatchString(name) {
		return Spec{}, fmt.Errorf("invalid package name %q", name)
	}
	return Spec{Name: name}, nil
}

// CheckResult is the outcome of a requirement on one server.
type CheckResult struct {
	Status  string `json:"status"`
	Version string `json:"version"` // installed version, if any
}

// Check tests a requirement against a server's packages.
func (s Spec) Check(l *List) CheckResult {
	found := l.Find(s.Name)
	if len(found) == 0 {
		return CheckResult{Status: StatusMissing}
	}
	return CheckResult{Status: StatusOK, Version: found[0].Version}
}
