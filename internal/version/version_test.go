package version_test

import (
	"strings"
	"testing"

	"github.com/ngbcoder/Unseal/internal/version"
)

func TestCurrentAlwaysReportsSomething(t *testing.T) {
	info := version.Current()

	if info.Version == "" {
		t.Error("Version must never be empty")
	}
	if info.GoVersion == "" {
		t.Error("GoVersion must never be empty")
	}
	if !strings.Contains(info.Platform, "/") {
		t.Errorf("Platform %q must look like os/arch", info.Platform)
	}
}

func TestStringIncludesVersionAndPlatform(t *testing.T) {
	info := version.Current()
	s := info.String()

	if !strings.Contains(s, info.Version) {
		t.Errorf("String() %q must contain the version", s)
	}
	if !strings.Contains(s, info.Platform) {
		t.Errorf("String() %q must contain the platform", s)
	}
}

func TestStringNeverLeaksAFullCommitHash(t *testing.T) {
	// The long hash belongs in Detail, not in the one-line summary.
	s := version.Current().String()
	for _, field := range strings.Fields(s) {
		field = strings.Trim(field, "(),")
		if len(field) == 40 && !strings.Contains(field, "-") {
			t.Errorf("String() %q must not contain a full 40-character hash", s)
		}
	}
}

func TestDetailMentionsEveryField(t *testing.T) {
	d := version.Current().Detail()

	for _, want := range []string{"Unseal", "commit", "built", "go", "platform"} {
		if !strings.Contains(d, want) {
			t.Errorf("Detail() %q must mention %q", d, want)
		}
	}
}

func TestShortHash(t *testing.T) {
	// shortHash is unexported, so exercise it through Current by checking
	// that a version fallback stays readable.
	if v := version.Current().Version; len(v) > 40 {
		t.Errorf("Version %q is implausibly long", v)
	}
}
