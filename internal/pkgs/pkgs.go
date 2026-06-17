// Package pkgs reads installed packages from servers that use rpm or dpkg.
// It only reads: msh never installs or removes packages by itself.
package pkgs

import (
	"strings"
)

// Package managers.
const (
	RPM  = "rpm"
	DPKG = "dpkg"
	None = "none"
)

// Command is the remote command that reads Script from standard input.
// The word after "--" only labels the run in process lists.
const Command = "sh -s -- msh-packages"

// Script prints "#manager=NAME" and one line per installed package:
// name, version and architecture, separated by tabs. dpkg also knows
// packages that were removed but kept their configuration; awk drops them.
const Script = `
if command -v dpkg-query >/dev/null 2>&1 && [ -f /var/lib/dpkg/status ]; then
  echo "#manager=dpkg"
  dpkg-query -W -f='${Package}\t${Version}\t${Architecture}\t${db:Status-Abbrev}\n' | awk -F'\t' '$4 ~ /^[ih]i/ {print $1 "\t" $2 "\t" $3}'
elif command -v rpm >/dev/null 2>&1; then
  echo "#manager=rpm"
  rpm -qa --queryformat '%{NAME}\t%{EPOCHNUM}:%{VERSION}-%{RELEASE}\t%{ARCH}\n'
else
  echo "#manager=none"
fi
`

// Package is one installed package.
type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Arch    string `json:"arch"`
}

// List is what one server has installed.
type List struct {
	Manager  string    `json:"manager"`
	Packages []Package `json:"packages"`
}

// Parse reads the output of Script.
func Parse(output string) *List {
	l := &List{Manager: None}
	for _, line := range strings.Split(output, "\n") {
		if m, ok := strings.CutPrefix(line, "#manager="); ok {
			l.Manager = strings.TrimSpace(m)
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 3 || fields[0] == "gpg-pubkey" {
			continue // noise, or rpm's pseudo package for signing keys
		}
		l.Packages = append(l.Packages, Package{
			Name: fields[0],
			// rpm prints epoch 0 explicitly; people never write it.
			Version: strings.TrimPrefix(fields[1], "0:"),
			Arch:    fields[2],
		})
	}
	return l
}

// Find returns the installed packages with the given name; a name can be
// installed for several architectures.
func (l *List) Find(name string) []Package {
	var out []Package
	for _, p := range l.Packages {
		if p.Name == name {
			out = append(out, p)
		}
	}
	return out
}
