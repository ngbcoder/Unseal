package logging

// Guard logs the entry and the exit of a function.
//
// The returned function must be deferred:
//
//	func Work() {
//	    defer logging.Guard(LevelVerbose, "Work")()
//
//	    // ...
//	}
//
// The entry is written with a ">>> " prefix and the exit with a "<<< "
// prefix, so the two ends of a call nest visibly in the log.
func (l *Logger) Guard(level int, context string) func() {
	l.LogCtx(level, context, func() string { return inMarker })
	return func() {
		l.LogCtx(level, context, func() string { return outMarker })
	}
}

// GuardMsg logs the entry and the exit of a function with a message.
//
// The message is written with both markers, so it is clear which call the
// line belongs to.
func (l *Logger) GuardMsg(level int, context string, message string) func() {
	l.LogCtx(level, context, func() string { return inMarker + message })
	return func() {
		l.LogCtx(level, context, func() string { return outMarker + message })
	}
}

// Guardf is Guard with a lazily built message.
//
// The message function is called twice, once for the entry and once for the
// exit.
func (l *Logger) Guardf(level int, context string, message func() string) func() {
	if !l.Enabled(level) {
		return func() {}
	}
	l.LogCtx(level, context, func() string { return inMarker + message() })
	return func() {
		l.LogCtx(level, context, func() string { return outMarker + message() })
	}
}

// Guard logs the entry and the exit of a function through the package-level
// logger.
//
// The returned function must be deferred:
//
//	defer logging.Guard(LevelVerbose, "Work")()
func Guard(level int, context string) func() {
	l := Default()
	if l == nil {
		return func() {}
	}
	return l.Guard(level, context)
}

// GuardMsg logs the entry and the exit of a function with a message through
// the package-level logger.
func GuardMsg(level int, context string, message string) func() {
	l := Default()
	if l == nil {
		return func() {}
	}
	return l.GuardMsg(level, context, message)
}

// Guardf is Guard with a lazily built message, through the package-level
// logger.
func Guardf(level int, context string, message func() string) func() {
	l := Default()
	if l == nil {
		return func() {}
	}
	return l.Guardf(level, context, message)
}

// Markers for the entry and the exit of a guarded function.
const (
	inMarker  = ">>> "
	outMarker = "<<< "
)
