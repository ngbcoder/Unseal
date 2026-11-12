package logging_test

import (
	"strings"
	"testing"

	"github.com/ngbcoder/Unseal/internal/logging"
)

func TestConfigureInstallsThePackageLogger(t *testing.T) {
	l := logging.Configure(logging.Options{
		Dir:          t.TempDir(),
		AppName:      "unseal",
		Level:        logging.LevelNormal,
		InstantFlush: true,
		Quiet:        true,
	})
	t.Cleanup(func() {
		_ = logging.Close()
		_ = l.Close()
	})

	if logging.Default() == nil {
		t.Fatal("Configure must install a package-level logger")
	}
	if logging.Level() != logging.LevelNormal {
		t.Errorf("the level is %d, want %d", logging.Level(), logging.LevelNormal)
	}

	logging.Log(logging.LevelNormal, "a message")
}

func TestLoggingWithoutALoggerIsHarmless(t *testing.T) {
	// Close whatever a previous test installed.
	_ = logging.Close()

	if logging.Default() != nil {
		t.Fatal("the package logger must be gone")
	}
	if logging.Enabled(logging.LevelError) {
		t.Error("nothing must be enabled without a logger")
	}

	// These must not panic.
	logging.Log(logging.LevelError, "no logger")
	logging.Logf(logging.LevelError, func() string { return "no logger" })
	logging.LogCtx(logging.LevelError, "ctx", "no logger")
	logging.SetLevel(logging.LevelVerbose)
	logging.SetInstantFlush(true)
}

func TestPackageLevelHelpersRespectTheLevel(t *testing.T) {
	dir := t.TempDir()
	logging.Configure(logging.Options{
		Dir:          dir,
		AppName:      "unseal",
		Level:        logging.LevelError,
		InstantFlush: true,
		Quiet:        true,
	})
	t.Cleanup(func() { _ = logging.Close() })

	built := 0
	logging.Logf(logging.LevelVerbose, func() string {
		built++
		return "verbose"
	})

	if built != 0 {
		t.Errorf("the message was built %d times, want 0", built)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	logging.Configure(logging.Options{
		Dir:     t.TempDir(),
		AppName: "unseal",
		Level:   logging.LevelNormal,
	})
	t.Cleanup(func() { _ = logging.Close() })

	if err := logging.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if err := logging.Close(); err != nil {
		t.Fatalf("the second Close failed: %v", err)
	}
}

func TestCallerContextIsDerivedFromTheCallSite(t *testing.T) {
	dir := t.TempDir()
	l := logging.Configure(logging.Options{
		Dir:          dir,
		AppName:      "unseal",
		Level:        logging.LevelNormal,
		InstantFlush: true,
		Quiet:        true,
	})
	t.Cleanup(func() {
		_ = logging.Close()
		_ = l.Close()
	})

	logging.Log(logging.LevelNormal, "derived")

	out := readLog(t, l)
	if !strings.Contains(out, "TestCallerContextIsDerivedFromTheCallSite") {
		t.Errorf("the context must name the calling function: %q", out)
	}
}
