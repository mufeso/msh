package pkgs

import (
	"maps"
	"slices"
	"strings"
)

// Difference is one package that differs from the baseline. An empty
// version means "not installed".
type Difference struct {
	Package  string `json:"package"`
	Baseline string `json:"baseline"`
	Version  string `json:"version"`
}

// versions maps package names to their installed versions; a package
// installed for several architectures counts its distinct versions.
func (l *List) versions() map[string]string {
	all := map[string][]string{}
	for _, p := range l.Packages {
		if !slices.Contains(all[p.Name], p.Version) {
			all[p.Name] = append(all[p.Name], p.Version)
		}
	}
	out := make(map[string]string, len(all))
	for name, versions := range all {
		slices.Sort(versions)
		out[name] = strings.Join(versions, ",")
	}
	return out
}

// Diff returns the differences of other from baseline, sorted by name.
func Diff(baseline, other *List) []Difference {
	base, have := baseline.versions(), other.versions()
	names := map[string]struct{}{}
	for n := range base {
		names[n] = struct{}{}
	}
	for n := range have {
		names[n] = struct{}{}
	}
	var out []Difference
	for _, n := range slices.Sorted(maps.Keys(names)) {
		if base[n] != have[n] {
			out = append(out, Difference{Package: n, Baseline: base[n], Version: have[n]})
		}
	}
	return out
}
