// Package version exposes build identity for the Unseal binary.
//
// There are two ways a build gets its identity.
//
// A tagged release is stamped explicitly at link time:
//
//	go build -trimpath \
//	  -ldflags "-s -w \
//	    -X github.com/ngbcoder/Unseal/internal/version.Version=$(git describe --tags --always) \
//	    -X github.com/ngbcoder/Unseal/internal/version.Commit=$(git rev-parse --short HEAD) \
//	    -X github.com/ngbcoder/Unseal/internal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
//	  -o bin/unseal ./cmd/unseal
//
// A plain "go build" needs no ceremony: since Go 1.24 the toolchain embeds the
// VCS revision, the commit time and the dirty-tree flag into the binary
// automatically, and this package recovers them. A plain build therefore
// still reports a real commit hash, it just reports the version as
// "0.0.0-dev".
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// Injected at link time with -ldflags -X.
var (
	// Version is the release version, for example "0.1.0".
	Version = "0.0.0-dev"
	// Commit is the short git commit hash the binary was built from.
	Commit = ""
	// BuildDate is the RFC3339 UTC timestamp of the build.
	BuildDate = ""
)

// Info describes the running binary.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
	Modified  bool   `json:"modified"`
}

// Current returns the build identity of the running binary.
//
// Values that were not injected at link time are recovered from the embedded
// VCS stamp, so a binary produced by a bare "go build" still reports its
// commit, its build time and whether the working tree was dirty.
func Current() Info {
	info := Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}

	settings := make(map[string]string, len(bi.Settings))
	for _, s := range bi.Settings {
		settings[s.Key] = s.Value
	}

	if info.Commit == "" {
		info.Commit = settings["vcs.revision"]
	}
	if info.BuildDate == "" {
		info.BuildDate = settings["vcs.time"]
	}
	info.Modified = settings["vcs.modified"] == "true"

	if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		info.Version = bi.Main.Version
	}

	// Nothing was injected and nothing was embedded: fall back to a short
	// commit so the identity is still specific.
	if info.Version == "0.0.0-dev" && info.Commit != "" {
		info.Version = "0.0.0-dev+" + shortHash(info.Commit)
	}

	return info
}

// shortHash truncates a commit hash to the conventional seven characters.
func shortHash(hash string) string {
	if len(hash) <= 7 {
		return hash
	}
	return hash[:7]
}

// String renders the build identity on a single line.
func (i Info) String() string {
	commit := i.Commit
	if commit == "" {
		commit = "unknown"
	} else {
		commit = shortHash(commit)
	}
	if i.Modified {
		commit += "-dirty"
	}
	return fmt.Sprintf("%s (%s) %s", i.Version, commit, i.Platform)
}

// Detail renders a multi-line build identity, used by "unseal version".
func (i Info) Detail() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Unseal %s\n", i.Version)
	fmt.Fprintf(&b, "  commit     %s\n", orUnknown(i.Commit))
	fmt.Fprintf(&b, "  built      %s\n", orUnknown(i.BuildDate))
	fmt.Fprintf(&b, "  go         %s\n", i.GoVersion)
	fmt.Fprintf(&b, "  platform   %s\n", i.Platform)
	if i.Modified {
		b.WriteString("  modified   working tree had uncommitted changes\n")
	}
	return b.String()
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
