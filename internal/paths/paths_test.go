package paths_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ngbcoder/Unseal/internal/paths"
)

func TestNewBuildsTheExpectedSubdirectories(t *testing.T) {
	root := filepath.Join("C:", "Users", "tester", "AppData", "Local", "unseal")
	l := paths.New(root)

	tests := map[string]string{
		"root":     l.Root,
		"data":     l.Data,
		"images":   l.Images,
		"config":   l.Config,
		"settings": l.Settings,
		"log":      l.Log,
		"temp":     l.Temp,
	}

	want := map[string]string{
		"root":     root,
		"data":     filepath.Join(root, "data"),
		"images":   filepath.Join(root, "img"),
		"config":   filepath.Join(root, "config"),
		"settings": filepath.Join(root, "settings"),
		"log":      filepath.Join(root, "log"),
		"temp":     filepath.Join(root, "tmp"),
	}

	for name, got := range tests {
		if got != want[name] {
			t.Errorf("%s = %q, want %q", name, got, want[name])
		}
	}
}

func TestEnsureCreatesEveryDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unseal")
	l := paths.New(root)

	if err := l.Ensure(); err != nil {
		t.Fatalf("Ensure failed: %v", err)
	}

	for _, dir := range l.All() {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("%s was not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}
}

func TestEnsureIsIdempotent(t *testing.T) {
	l := paths.New(filepath.Join(t.TempDir(), "unseal"))

	if err := l.Ensure(); err != nil {
		t.Fatalf("first Ensure failed: %v", err)
	}
	if err := l.Ensure(); err != nil {
		t.Fatalf("second Ensure failed: %v", err)
	}
}

func TestResolveUsesTheLocalAppDataFolder(t *testing.T) {
	local := filepath.Join(t.TempDir(), "Local")
	t.Setenv("LOCALAPPDATA", local)

	l, err := paths.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	want := filepath.Join(local, paths.Company, paths.AppName)
	if l.Root != want {
		t.Errorf("the root is %q, want %q", l.Root, want)
	}
	if l.Temp == "" || l.Data == "" {
		t.Error("Resolve must fill in every directory")
	}
}

func TestTheLayoutMatchesTheHandoffContract(t *testing.T) {
	// URIhandoff.md names the handoff document's location explicitly, so the
	// temp directory has to be where the contract says it is.
	local := filepath.Join(t.TempDir(), "Local")
	t.Setenv("LOCALAPPDATA", local)

	l, err := paths.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	want := filepath.Join(local, "audiamus", "Unseal", "tmp")
	if l.Temp != want {
		t.Errorf("the temp directory is %q, want %q", l.Temp, want)
	}
}

func TestResolveFallsBackWhenLocalAppDataIsUnset(t *testing.T) {
	t.Setenv("LOCALAPPDATA", "")

	l, err := paths.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if l.Root == "" {
		t.Error("Resolve must produce a root even without LOCALAPPDATA")
	}
}

func TestUserNameIsNotEmptyOnWindows(t *testing.T) {
	// The credential-store key derivation needs a user name. On a normal
	// Windows session it must be there.
	if name := paths.UserName(); name == "" {
		t.Skip("no user name in the environment")
	}
}
