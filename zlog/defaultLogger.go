package zlog

import (
	"go.uber.org/zap/zapcore"
	"io"
	"log"
	"os"
	"slices"
	"sync"
)

// defaultLogger is a stdlib-backed logger used as a safe fallback when
// zap cannot be initialized. It prints key=value fields and never panics.
type defaultLogger struct {
	// base holds persistent structured fields included on every log call.
	base []Field

	// mu protects concurrent access to loggers.
	mu sync.RWMutex

	// loggers maps levels to dedicated *log.Logger instances.
	// Using per-level loggers avoids global SetOutput churn.
	loggers map[zapcore.Level]*log.Logger
}

// Ensure defaultLogger satisfies ZLogger at compile time.
var _ ZLogger = (*defaultLogger)(nil)

// newDefaultLogger constructs a defaultLogger prefilled with an "app" field
// (derived from the executable name) and per-level *log.Logger outputs.
func newDefaultLogger() *defaultLogger {
	return &defaultLogger{
		base: []Field{String("app", detectAppName())},
		loggers: map[zapcore.Level]*log.Logger{
			zapcore.DebugLevel: log.New(os.Stderr, "", log.LstdFlags),
			zapcore.InfoLevel:  log.New(os.Stdout, "", log.LstdFlags),
			zapcore.WarnLevel:  log.New(os.Stderr, "", log.LstdFlags),
			zapcore.ErrorLevel: log.New(os.Stderr, "", log.LstdFlags),
		},
	}
}

// With returns a child logger that carries base+fields for subsequent calls.
// It preserves the parent's per-level loggers so any RedirectOutput applied
// to the parent continues to affect the child.
func (d *defaultLogger) With(fields ...Field) ZLogger {
	d.mu.RLock()
	defer d.mu.RUnlock()
	all := append(slices.Clone(d.base), fields...)
	l := newDefaultLogger()
	l.base = all
	l.loggers = d.loggers
	return l
}

// RedirectStdLog redirects the global log package to this logger at the
// provided level. It returns a restore function that reverts output, flags,
// and prefix to their previous values.
func (d *defaultLogger) RedirectStdLog(level zapcore.Level) (restore func()) {
	if d == nil {
		d = newDefaultLogger()
	}
	prevOut := log.Writer()
	prevFlags := log.Flags()
	prevPrefix := log.Prefix()

	std := StdLoggerAt(d, level)
	log.SetOutput(std.Writer())
	log.SetFlags(0)
	log.SetPrefix("")

	return func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
		log.SetPrefix(prevPrefix)
	}
}

// RedirectOutput routes this logger's output at the specified level to w.
// It returns a restore function that restores the previous writer for that level.
func (d *defaultLogger) RedirectOutput(w io.Writer, level zapcore.Level) (restore func()) {
	if w == nil {
		w = io.Discard
	}
	d.mu.Lock()
	old := d.loggers[level]
	nl := log.New(w, "", log.LstdFlags)
	d.loggers[level] = nl
	d.mu.Unlock()
	return func() {
		d.mu.Lock()
		d.loggers[level] = old
		d.mu.Unlock()
	}
}

// Info logs msg at Info level with optional structured fields.
func (d *defaultLogger) Info(msg string, fields ...Field) {
	all := append(slices.Clone(d.base), fields...)
	d.mu.RLock()
	l := d.loggers[zapcore.InfoLevel]
	d.mu.RUnlock()
	l.Println("INFO:", msg, flatten(all...))
}

// Error logs msg at Error level with optional structured fields.
func (d *defaultLogger) Error(msg string, fields ...Field) {
	all := append(slices.Clone(d.base), fields...)
	d.mu.RLock()
	l := d.loggers[zapcore.ErrorLevel]
	d.mu.RUnlock()
	l.Println("ERROR:", msg, flatten(all...))
}

// Sync is a no-op for the stdlib-backed logger.
func (d *defaultLogger) Sync() error { return nil }

// Debug logs msg at Debug level with optional structured fields.
func (d *defaultLogger) Debug(msg string, fields ...Field) {
	all := append(slices.Clone(d.base), fields...)
	d.mu.RLock()
	l := d.loggers[zapcore.DebugLevel]
	d.mu.RUnlock()
	l.Println("DEBUG:", msg, flatten(all...))
}

// Warn logs msg at Warn level with optional structured fields.
func (d *defaultLogger) Warn(msg string, fields ...Field) {
	all := append(slices.Clone(d.base), fields...)
	d.mu.RLock()
	l := d.loggers[zapcore.WarnLevel]
	d.mu.RUnlock()
	l.Println("WARN:", msg, flatten(all...))
}
