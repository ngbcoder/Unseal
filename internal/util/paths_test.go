package util_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ngbcoder/Unseal/internal/util"
)

func TestAsUncIfLongLeavesShortPathsAlone(t *testing.T) {
	got := util.AsUncIfLong("C:\\Music\\Audible\\file.m4b")

	if util.IsUnc(got) {
		t.Errorf("a short path must not be rewritten, got %q", got)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("the result must be absolute, got %q", got)
	}
}

func TestAsUncIfLongRewritesLongPaths(t *testing.T) {
	long := `C:\` + strings.Repeat("a", 300) + `\file.m4b`

	got := util.AsUncIfLong(long)

	if !util.IsUnc(got) {
		t.Fatalf("a long path must be rewritten, got %q", got)
	}
	if !strings.HasPrefix(got, `\\?\C:\`) {
		t.Errorf("the rewritten path must carry the extended prefix, got %q", got)
	}
}

func TestAsUncIfLongIsIdempotent(t *testing.T) {
	long := `C:\` + strings.Repeat("b", 300) + `\file.m4b`

	once := util.AsUncIfLong(long)
	twice := util.AsUncIfLong(once)

	if once != twice {
		t.Errorf("AsUncIfLong is not idempotent: %q then %q", once, twice)
	}
}

func TestAsUncHandlesNetworkPaths(t *testing.T) {
	got := util.AsUnc(`\\server\share\file.m4b`)

	if got != `\\?\UNC\server\share\file.m4b` {
		t.Errorf("AsUnc = %q, want %q", got, `\\?\UNC\server\share\file.m4b`)
	}
}

func TestIsUnc(t *testing.T) {
	if util.IsUnc(`C:\plain`) {
		t.Error("a plain path must not be reported as extended")
	}
	if !util.IsUnc(`\\?\C:\extended`) {
		t.Error("an extended path must be recognised")
	}
}
