package logging

import (
	"runtime"
	"strings"
	"sync"
)

// defaultLogger is the logger the package-level functions write to.
//
// It is nil until Configure is called. Writing to a nil logger is a no-op,
// so a program that never configures logging simply produces no output
// rather than crashing.
var (
	defaultMu     sync.RWMutex
	defaultLogger *Logger
)

// Configure installs the package-level logger.
//
// It replaces any logger installed before. The log file itself is opened
// lazily, on the first message.
func Configure(opts Options) *Logger {
	l := New(opts)

	defaultMu.Lock()
	defaultLogger = l
	defaultMu.Unlock()

	return l
}

// Default returns the package-level logger, or nil when none is configured.
func Default() *Logger {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultLogger
}

// SetLevel changes the level of the package-level logger.
func SetLevel(level int) {
	if l := Default(); l != nil {
		l.SetLevel(level)
	}
}

// Level returns the level of the package-level logger.
func Level() int {
	if l := Default(); l != nil {
		return l.Level()
	}
	return DefaultLevel
}

// SetInstantFlush turns instant flushing on or off.
func SetInstantFlush(on bool) {
	if l := Default(); l != nil {
		l.SetInstantFlush(on)
	}
}

// Enabled reports whether a message at the given level would be written.
func Enabled(level int) bool {
	l := Default()
	if l == nil {
		return false
	}
	return l.Enabled(level)
}

// Log writes a message at the given level.
//
// The context is derived from the call site, so the line reads
// "[Type.Method] message" without the caller having to say so.
func Log(level int, message string) {
	if l := Default(); l != nil {
		l.Log(level, callerContext(2), message)
	}
}

// Logf writes a message at the given level, building it only when the level
// is enabled.
//
// This is the form to use for an expensive message, because the
// interpolation only happens when the message will actually be written.
func Logf(level int, message func() string) {
	if l := Default(); l != nil {
		l.Logf(level, message)
	}
}

// LogCtx writes a message at the given level with an explicit context.
func LogCtx(level int, context string, message string) {
	if l := Default(); l != nil {
		l.Log(level, context, message)
	}
}

// LogfCtx writes a message at the given level with an explicit context,
// building it only when the level is enabled.
func LogfCtx(level int, context string, message func() string) {
	if l := Default(); l != nil {
		l.LogCtx(level, context, message)
	}
}

// Close closes the package-level logger.
func Close() error {
	defaultMu.Lock()
	l := defaultLogger
	defaultLogger = nil
	defaultMu.Unlock()

	if l == nil {
		return nil
	}
	return l.Close()
}

// callerContext derives a context string from the call site.
//
// It produces "Type.Method" for a method and "Function" for a plain
// function, matching the original implementation's context format.
//
// The skip argument is the number of stack frames to step over, so that the
// helper itself is not reported.
func callerContext(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return ""
	}

	return shortenContext(fn.Name())
}

// shortenContext turns a fully qualified function name into a short context.
func shortenContext(name string) string {
	// Strip the package path.
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}

	// A method with a pointer receiver reads "pkg.(*Type).Method".
	if open := strings.Index(name, "(*"); open >= 0 {
		if close := strings.Index(name[open:], ")"); close >= 0 {
			typ := name[open+2 : open+close]
			rest := strings.TrimPrefix(name[open+close+1:], ".")
			switch {
			case typ != "" && rest != "":
				return typ + "." + rest
			case typ != "":
				return typ
			}
		}
	}

	// A method with a value receiver reads "pkg.Type.Method".
	if i := strings.LastIndex(name, "."); i >= 0 {
		rest := name[i+1:]
		if rest == "" || strings.HasPrefix(rest, "func") {
			return ""
		}
		return rest
	}

	return name
}
