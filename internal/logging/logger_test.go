package logging_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ngbcoder/Unseal/internal/logging"
)

// readLog returns the contents of the log file the logger wrote to.
func readLog(t *testing.T, l *logging.Logger) string {
	t.Helper()

	path := l.CurrentFile()
	if path == "" {
		t.Fatal("the logger has no current file")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read the log file: %v", err)
	}
	return string(data)
}

// newLogger creates a logger in a temporary directory.
func newLogger(t *testing.T, level int) *logging.Logger {
	t.Helper()

	l := logging.New(logging.Options{
		Dir:          filepath.Join(t.TempDir(), "log"),
		AppName:      "unseal",
		Level:        level,
		InstantFlush: true,
		Quiet:        true,
	})
	t.Cleanup(func() { _ = l.Close() })
	return l
}

func TestLogWritesTheExpectedFormat(t *testing.T) {
	l := newLogger(t, logging.LevelNormal)

	l.Log(logging.LevelNormal, "Api.GetLibrary", "started")

	out := readLog(t, l)
	line := strings.TrimRight(out, "\n")

	// HH:mm:ss.fff TTTT [Context.Method] message
	parts := strings.SplitN(line, " ", 4)
	if len(parts) != 4 {
		t.Fatalf("unexpected line layout: %q", line)
	}
	if _, err := time.Parse("15:04:05.000", parts[0]); err != nil {
		t.Errorf("the timestamp %q is not in the expected format: %v", parts[0], err)
	}
	if len(parts[1]) != 4 {
		t.Errorf("the goroutine id %q must be four digits", parts[1])
	}
	if parts[2] != "[Api.GetLibrary]" {
		t.Errorf("the context is %q, want %q", parts[2], "[Api.GetLibrary]")
	}
	if parts[3] != "started" {
		t.Errorf("the message is %q, want %q", parts[3], "started")
	}
}

func TestLogOmitsAnEmptyContext(t *testing.T) {
	l := newLogger(t, logging.LevelNormal)

	l.Log(logging.LevelNormal, "", "bare message")

	out := strings.TrimRight(readLog(t, l), "\n")
	if strings.Contains(out, "[]") {
		t.Errorf("an empty context must not produce brackets: %q", out)
	}
	if !strings.HasSuffix(out, "bare message") {
		t.Errorf("the message is missing: %q", out)
	}
}

func TestLevelGating(t *testing.T) {
	l := newLogger(t, logging.LevelWarning)

	l.Log(logging.LevelError, "c", "an error")
	l.Log(logging.LevelWarning, "c", "a warning")
	l.Log(logging.LevelInfo, "c", "an info")
	l.Log(logging.LevelVerbose, "c", "verbose noise")

	out := readLog(t, l)

	if !strings.Contains(out, "an error") {
		t.Error("an error must be written at warning level")
	}
	if !strings.Contains(out, "a warning") {
		t.Error("a warning must be written at warning level")
	}
	if strings.Contains(out, "an info") {
		t.Error("an info must not be written at warning level")
	}
	if strings.Contains(out, "verbose noise") {
		t.Error("a verbose message must not be written at warning level")
	}
}

func TestLogfOnlyBuildsTheMessageWhenEnabled(t *testing.T) {
	l := newLogger(t, logging.LevelError)

	built := 0
	l.Logf(logging.LevelVerbose, func() string {
		built++
		return "expensive"
	})

	if built != 0 {
		t.Errorf("the message was built %d times, want 0 when the level is off", built)
	}

	l.Logf(logging.LevelError, func() string {
		built++
		return "cheap"
	})

	if built != 1 {
		t.Errorf("the message was built %d times, want 1 when the level is on", built)
	}
}

func TestSetLevelChangesWhatIsWritten(t *testing.T) {
	l := newLogger(t, logging.LevelError)

	l.Log(logging.LevelVerbose, "c", "before")
	l.SetLevel(logging.LevelVerbose)
	l.Log(logging.LevelVerbose, "c", "after")

	out := readLog(t, l)

	if strings.Contains(out, "before") {
		t.Error("a verbose message must not be written before the level is raised")
	}
	if !strings.Contains(out, "after") {
		t.Error("a verbose message must be written after the level is raised")
	}
}

func TestEnabled(t *testing.T) {
	l := newLogger(t, logging.LevelInfo)

	if l.Enabled(logging.LevelError) != true {
		t.Error("an error must be enabled at info level")
	}
	if l.Enabled(logging.LevelVerbose) != false {
		t.Error("verbose must be disabled at info level")
	}
}

func TestTheFileNameCarriesTheDateAndACounter(t *testing.T) {
	l := newLogger(t, logging.LevelNormal)
	l.Log(logging.LevelNormal, "c", "hello")

	name := filepath.Base(l.CurrentFile())
	want := "unseal_" + time.Now().Format("2006-01-02") + "_001.log"

	if name != want {
		t.Errorf("the file name is %q, want %q", name, want)
	}
}

func TestTheFileIsRolledOverWhenItGetsTooBig(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "log")
	l := logging.New(logging.Options{
		Dir:          dir,
		AppName:      "unseal",
		Level:        logging.LevelNormal,
		InstantFlush: true,
		Quiet:        true,
		FileSize:     64, // a few lines
	})
	t.Cleanup(func() { _ = l.Close() })

	for i := 0; i < 20; i++ {
		l.Log(logging.LevelNormal, "c", "a line of log output that is long enough to roll over")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read the log directory: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected the log to roll over into several files, found %d", len(entries))
	}

	// The first file must not have exceeded the limit by much.
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Size() > 200 {
			t.Errorf("file %s is %d bytes, the limit was 64", e.Name(), info.Size())
		}
	}
}

func TestTheLogDirectoryIsCreatedOnDemand(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "not", "there", "yet")
	l := logging.New(logging.Options{
		Dir:          dir,
		AppName:      "unseal",
		Level:        logging.LevelNormal,
		InstantFlush: true,
		Quiet:        true,
	})
	t.Cleanup(func() { _ = l.Close() })

	l.Log(logging.LevelNormal, "c", "hello")

	if _, err := os.Stat(dir); err != nil {
		t.Errorf("the log directory was not created: %v", err)
	}
}

func TestGuardWritesBothMarkers(t *testing.T) {
	l := newLogger(t, logging.LevelNormal)

	func() {
		defer l.GuardMsg(logging.LevelNormal, "Api.Work", "step")()
		l.Log(logging.LevelNormal, "inner", "the work itself")
	}()

	out := readLog(t, l)

	if !strings.Contains(out, ">>> step") {
		t.Errorf("the entry marker is missing: %q", out)
	}
	if !strings.Contains(out, "<<< step") {
		t.Errorf("the exit marker is missing: %q", out)
	}
	if !strings.Contains(out, "the work itself") {
		t.Errorf("the inner message is missing: %q", out)
	}
}

func TestTheGoroutineIdStaysCorrectOverManyLines(t *testing.T) {
	l := newLogger(t, logging.LevelNormal)

	// A pooled buffer that is trimmed in place shrinks on every use, so a
	// regression here shows up only after a number of lines.
	for i := 0; i < 200; i++ {
		l.Log(logging.LevelNormal, "c", "a line")
	}

	out := readLog(t, l)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 200 {
		t.Fatalf("expected 200 lines, got %d", len(lines))
	}

	for _, line := range lines {
		parts := strings.SplitN(line, " ", 4)
		if len(parts) < 4 {
			t.Fatalf("unexpected line layout: %q", line)
		}
		if parts[1] == "0000" {
			t.Fatalf("the goroutine id became zero: %q", line)
		}
	}
}
