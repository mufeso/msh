// Package buildinfo describes the msh binary: its name, version, source
// commit and build date.
//
// Version, Commit and Date are variables so they can be replaced at build
// time without editing source code:
//
//	go build -ldflags "-X example.com/msh/internal/buildinfo.Version=1.0.0" ./cmd/msh
package buildinfo

import (
	"fmt"
	"runtime"
)

// Name is the executable name; LongName is what the name stands for.
const (
	Name     = "msh"
	LongName = "My Parallel SSH"
)

// Values replaced at build time.
var (
	Version = "0.1.0-dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// Info is a snapshot of the build information.
type Info struct {
	Name      string
	Version   string
	Commit    string
	Date      string
	GoVersion string
	Platform  string
}

// Get returns the build information of the running binary.
func Get() Info {
	return Info{
		Name:      Name,
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
}

// Banner returns "msh — My Parallel SSH".
func Banner() string {
	return Name + " — " + LongName
}

// String returns the full version line printed by --version.
func String() string {
	return fmt.Sprintf("%s version %s (commit %s, built %s)", Name, Version, Commit, Date)
}
