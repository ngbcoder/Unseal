// Package logging provides Unseal's leveled logging facility.
//
// Levels run from 0 to 5:
//
//	0  errors only
//	1  warnings
//	2  informational
//	3  normal flow, the default
//	4  verbose, including HTTP dumps and diagnostics
//	5  reserved
//
// Messages are written to rolling daily files in the log directory:
//
//	<logdir>\<appname>_<yyyy-MM-dd>_<NNN>.log
//
// A new file is started when the calendar day changes or when the current
// file reaches 20 MB. Up to 1000 files are kept per day.
//
// Every line has the form
//
//	HH:mm:ss.fff TTTT [Context.Method] message
//
// where TTTT is the id of the goroutine that wrote it.
//
// The package also offers a level-gated form, Logf, which only builds the
// message when the level is enabled, so an expensive interpolation costs
// nothing when the level is off.
//
// Layer L0 (foundation).
package logging

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Log levels.
const (
	LevelError    = 0
	LevelWarning  = 1
	LevelInfo     = 2
	LevelNormal   = 3
	LevelVerbose  = 4
	LevelReserved = 5
)

// DefaultLevel is the level used when nothing else selects one.
const DefaultLevel = LevelNormal

// DefaultFileSize is the size at which a log file is rolled over.
const DefaultFileSize = 20 * 1024 * 1024

// maxFilesPerDay is the number of log files kept for one calendar day.
const maxFilesPerDay = 1000

// flushInterval is how often a buffered writer is flushed when instant
// flushing is off.
const flushInterval = 5 * time.Second

// ext is the log file extension.
const ext = ".log"

// Logger writes log messages to rolling files.
//
// A Logger is safe for concurrent use.
type Logger struct {
	mu sync.Mutex

	dir     string
	appName string

	level        int
	instantFlush bool
	quiet        bool

	file      *os.File
	writer    *bufio.Writer
	fileDate  time.Time
	fileCount int
	bytes     int64
	fileSize  int64
	announced bool

	flushTimer *time.Timer
	closed     bool
}

// Options configure a Logger.
type Options struct {
	// Dir is the directory the log files are written to.
	Dir string
	// AppName is the file name stub.
	AppName string
	// Level is the maximum level that is written.
	Level int
	// InstantFlush writes every message straight to disk.
	InstantFlush bool
	// FileSize is the size at which a file is rolled over. Zero means
	// DefaultFileSize.
	FileSize int64
	// Quiet suppresses the one-off notice that names the log file.
	Quiet bool
}

// New creates a logger.
//
// The log file is opened lazily, on the first message, so creating a logger
// never fails because of the filesystem.
func New(opts Options) *Logger {
	size := opts.FileSize
	if size <= 0 {
		size = DefaultFileSize
	}
	return &Logger{
		dir:          opts.Dir,
		appName:      opts.AppName,
		level:        opts.Level,
		instantFlush: opts.InstantFlush,
		fileSize:     size,
		quiet:        opts.Quiet,
	}
}

// SetLevel changes the maximum level that is written.
func (l *Logger) SetLevel(level int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// Level returns the current level.
func (l *Logger) Level() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// SetInstantFlush turns instant flushing on or off.
//
// Turning it on flushes every message immediately. Turning it off starts a
// timer that flushes every five seconds.
func (l *Logger) SetInstantFlush(on bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.instantFlush = on
	if !on && l.flushTimer == nil && !l.closed {
		l.flushTimer = time.AfterFunc(flushInterval, l.flushTick)
	}
	if on && l.flushTimer != nil {
		l.flushTimer.Stop()
		l.flushTimer = nil
	}
}

// Enabled reports whether a message at the given level would be written.
func (l *Logger) Enabled(level int) bool {
	return level <= l.Level()
}

// Log writes a message at the given level with an explicit context.
//
// The context is the part in square brackets, usually "Type.Method".
func (l *Logger) Log(level int, context, message string) {
	if !l.Enabled(level) {
		return
	}
	l.write(format(time.Now(), goid(), context, message))
}

// Logf writes a message at the given level, building it only when the level
// is enabled.
//
// The context is derived from the call site.
func (l *Logger) Logf(level int, message func() string) {
	if !l.Enabled(level) {
		return
	}
	l.write(format(time.Now(), goid(), callerContext(2), message()))
}

// LogCtx writes a message at the given level with an explicit context,
// building it only when the level is enabled.
func (l *Logger) LogCtx(level int, context string, message func() string) {
	if !l.Enabled(level) {
		return
	}
	l.write(format(time.Now(), goid(), context, message()))
}

// write appends a formatted line to the current file.
func (l *Logger) write(line string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return
	}

	l.ensureWriter()

	data := line + "\n"
	n, err := l.writer.WriteString(data)
	if err != nil {
		return
	}
	l.bytes += int64(n)

	if l.instantFlush {
		_ = l.writer.Flush()
	}
}

// ensureWriter opens or rolls the current file as needed.
func (l *Logger) ensureWriter() {
	if l.file == nil {
		l.openWriter(true)
		return
	}

	today := time.Now().Truncate(24 * time.Hour)
	if !sameDay(today, l.fileDate) {
		l.nextWriter(true)
		return
	}
	if l.bytes >= l.fileSize {
		l.nextWriter(false)
	}
}

// nextWriter closes the current file and opens the next one.
func (l *Logger) nextWriter(newDay bool) {
	l.closeWriter()
	l.openWriter(newDay)
}

// closeWriter closes and flushes the current file.
func (l *Logger) closeWriter() {
	if l.flushTimer != nil {
		l.flushTimer.Stop()
		l.flushTimer = nil
	}
	if l.writer != nil {
		_ = l.writer.Flush()
		l.writer = nil
	}
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
	l.bytes = 0
}

// openWriter opens a log file.
//
// When newDay is set the file counter starts over and the date in the file
// name is updated.
func (l *Logger) openWriter(newDay bool) {
	if l.dir == "" || l.appName == "" {
		return
	}

	if newDay {
		l.fileDate = time.Now().Truncate(24 * time.Hour)
		l.fileCount = 0
	}

	if err := os.MkdirAll(l.dir, 0o755); err != nil {
		return
	}

	stub := filepath.Join(l.dir, fmt.Sprintf("%s_%s_", l.appName, l.fileDate.Format("2006-01-02")))

	ignoreExisting := false
	for {
		l.fileCount++
		name := fmt.Sprintf("%s%03d%s", stub, l.fileCount, ext)

		if _, err := os.Stat(name); err == nil && !ignoreExisting {
			if l.fileCount < maxFilesPerDay {
				continue
			}
			// More than a thousand files for one day. Start over and
			// overwrite, rather than refusing to write at all.
			ignoreExisting = true
			l.fileCount = 1
			continue
		}

		flags := os.O_CREATE | os.O_WRONLY | os.O_EXCL
		if ignoreExisting {
			flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		}

		f, err := os.OpenFile(name, flags, 0o644)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return
		}

		l.file = f
		l.writer = bufio.NewWriter(f)
		l.bytes = 0

		if !l.instantFlush {
			l.flushTimer = time.AfterFunc(flushInterval, l.flushTick)
		}

		if !l.announced && !l.quiet {
			l.announced = true
			fmt.Printf("Logging written to %q.\n", name)
		}
		return
	}
}

// flushTick is the timer callback that flushes a buffered writer.
func (l *Logger) flushTick() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed || l.writer == nil {
		return
	}
	_ = l.writer.Flush()

	if !l.instantFlush {
		l.flushTimer = time.AfterFunc(flushInterval, l.flushTick)
	}
}

// Close flushes and closes the current log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}
	l.closed = true
	l.closeWriter()
	return nil
}

// CurrentFile returns the path of the file currently being written.
func (l *Logger) CurrentFile() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return ""
	}
	return l.file.Name()
}

// format renders one log line.
func format(ts time.Time, goroutineID uint64, context, message string) string {
	var b strings.Builder

	b.WriteString(ts.Format("15:04:05.000"))
	b.WriteByte(' ')
	b.WriteString(fmt.Sprintf("%04d", goroutineID))
	b.WriteByte(' ')

	if c := strings.TrimSpace(context); c != "" {
		b.WriteByte('[')
		b.WriteString(c)
		b.WriteString("] ")
	}

	b.WriteString(message)
	return b.String()
}

// sameDay reports whether two times fall on the same calendar day.
func sameDay(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// goidPool reuses the buffer used to read the goroutine id.
var goidPool = sync.Pool{
	New: func() any {
		b := make([]byte, 64)
		return &b
	},
}

// goid returns the id of the calling goroutine.
//
// Go does not expose it, so it is read out of the goroutine's own stack
// header. The buffer is pooled so a log line does not allocate.
//
// The pooled buffer must be put back untouched. Trimming it in place would
// return a sub-slice with a smaller capacity, and the pool would then hand
// out a buffer that shrinks a little more on every call, until it is too
// small to hold the header at all.
func goid() uint64 {
	bp := goidPool.Get().(*[]byte)
	buf := *bp

	n := runtime.Stack(buf, false)

	raw := bytes.TrimPrefix(buf[:n], []byte("goroutine "))
	if i := bytes.IndexByte(raw, ' '); i >= 0 {
		raw = raw[:i]
	}

	id, err := strconv.ParseUint(string(raw), 10, 64)
	if err != nil {
		id = 0
	}

	goidPool.Put(bp)

	return id
}
