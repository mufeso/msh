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
	"runtime/debug"
)

// Name is the executable name; LongName is what the name stands for.
const (
	Name     = "msh"
	LongName = "My Parallel SSH"
)

// defaultVersion is the version of builds without -ldflags.
const defaultVersion = "2.2.1"

// Values replaced at build time.
var (
	Version = defaultVersion
	Commit  = "unknown"
	Date    = "unknown"
)

// Info is a snapshot of the build information.
//
// The struct tags control the JSON field names. They are part of the
// machine-readable contract, so they use stable snake_case names.
type Info struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get returns the build information of the running binary. Values set
// with -ldflags win; missing ones come from the build information the Go
// toolchain embeds in every binary.
func Get() Info {
	info := Info{
		Name:      Name,
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		info = withBuildInfo(info, bi)
	}
	return info
}

// withBuildInfo fills defaulted fields from bi. It is a pure function so
// that tests can pass any BuildInfo.
func withBuildInfo(info Info, bi *debug.BuildInfo) Info {
	if info.Version == defaultVersion && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		info.Version = bi.Main.Version
	}
	settings := map[string]string{}
	for _, s := range bi.Settings {
		settings[s.Key] = s.Value
	}
	if info.Commit == "unknown" {
		if rev := settings["vcs.revision"]; rev != "" {
			if len(rev) > 12 {
				rev = rev[:12]
			}
			if settings["vcs.modified"] == "true" {
				rev += "-dirty"
			}
			info.Commit = rev
		}
	}
	if info.Date == "unknown" && settings["vcs.time"] != "" {
		info.Date = settings["vcs.time"]
	}
	return info
}

// Banner returns "msh — My Parallel SSH".
func Banner() string {
	return Name + " — " + LongName
}

// String returns the full version line printed by --version.
func String() string {
	return fmt.Sprintf("%s version %s (commit %s, built %s)", Name, Version, Commit, Date)
}
