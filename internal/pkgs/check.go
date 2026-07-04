package pkgs

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Check statuses.
const (
	StatusOK           = "ok"
	StatusMissing      = "missing"
	StatusWrongVersion = "wrong-version"
	StatusUnwanted     = "unwanted"
)

var specName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9+._:-]*$`)

// Spec is a requirement on a package.
type Spec struct {
	Name    string
	Op      string // "", =, ==, !=, <, <=, >, >=
	Version string
	Absent  bool // "!name": the package must not be installed
}

// String returns the requirement as text.
func (s Spec) String() string {
	if s.Absent {
		return "!" + s.Name
	}
	return s.Name + s.Op + s.Version
}

// operators are tried in this order: longer ones first, so ">=" is not
// read as ">" followed by "=3.0".
var operators = []string{">=", "<=", "==", "!=", ">", "<", "="}

// ParseSpec parses "name", "name>=1.2" or "!name".
func ParseSpec(text string) (Spec, error) {
	text = strings.TrimSpace(text)
	var s Spec
	if rest, ok := strings.CutPrefix(text, "!"); ok {
		s.Absent = true
		text = rest
	}
	s.Name = text
	if !s.Absent {
		for _, op := range operators {
			if name, version, ok := strings.Cut(text, op); ok {
				s.Name, s.Op, s.Version = strings.TrimSpace(name), op, strings.TrimSpace(version)
				if s.Version == "" {
					return Spec{}, fmt.Errorf("missing version in %q", text)
				}
				break
			}
		}
	}
	if !specName.MatchString(s.Name) {
		return Spec{}, fmt.Errorf("invalid package name %q", s.Name)
	}
	return s, nil
}

// CheckResult is the outcome of a requirement on one server.
type CheckResult struct {
	Status  string `json:"status"`
	Version string `json:"version"` // installed version, if any
}

// Check tests a requirement against a server's packages.
func (s Spec) Check(l *List) CheckResult {
	found := l.Find(s.Name)
	if s.Absent {
		if len(found) == 0 {
			return CheckResult{Status: StatusOK}
		}
		return CheckResult{Status: StatusUnwanted, Version: found[0].Version}
	}
	if len(found) == 0 {
		return CheckResult{Status: StatusMissing}
	}
	// With several architectures, the highest version counts.
	best := found[0].Version
	for _, p := range found[1:] {
		if Compare(l.Manager, p.Version, best) > 0 {
			best = p.Version
		}
	}
	if s.Op != "" && !Satisfies(l.Manager, best, s.Op, s.Version) {
		return CheckResult{Status: StatusWrongVersion, Version: best}
	}
	return CheckResult{Status: StatusOK, Version: best}
}

// ParseBaseline reads requirements, one per line. Empty lines and lines
// starting with # are ignored.
func ParseBaseline(r io.Reader) ([]Spec, error) {
	var specs []Spec
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		s, err := ParseSpec(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", n, err)
		}
		specs = append(specs, s)
	}
	return specs, sc.Err()
}
