package zlog

import (
	"go.uber.org/zap/zapcore"
	"io"
	"log"
	"os"
	"slices"
	"sync"
)

// defaultLogger falls back to the standard log package.
type defaultLogger struct {
	base    []Field
	mu      sync.RWMutex
	loggers map[zapcore.Level]*log.Logger
}

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

func (d *defaultLogger) With(fields ...Field) ZLogger {
	d.mu.RLock()
	defer d.mu.RUnlock()
	all := append(slices.Clone(d.base), fields...)
	l := newDefaultLogger()
	l.base = all
	l.loggers = d.loggers
	return l
}

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

// base holds persistent structured fields for this logger instance.
// We clone it before appending new fields to avoid slice aliasing.
func (d *defaultLogger) Info(msg string, fields ...Field) {
	all := append(slices.Clone(d.base), fields...)
	d.mu.RLock()
	l := d.loggers[zapcore.InfoLevel]
	d.mu.RUnlock()
	l.Println("INFO:", msg, flatten(all...))
}

func (d *defaultLogger) Error(msg string, fields ...Field) {
	all := append(slices.Clone(d.base), fields...)
	d.mu.RLock()
	l := d.loggers[zapcore.ErrorLevel]
	d.mu.RUnlock()
	l.Println("ERROR:", msg, flatten(all...))
}

func (d *defaultLogger) Sync() error { return nil }

func (d *defaultLogger) Debug(msg string, fields ...Field) {
	all := append(slices.Clone(d.base), fields...)
	d.mu.RLock()
	l := d.loggers[zapcore.DebugLevel]
	d.mu.RUnlock()
	l.Println("DEBUG:", msg, flatten(all...))
}

func (d *defaultLogger) Warn(msg string, fields ...Field) {
	all := append(slices.Clone(d.base), fields...)
	d.mu.RLock()
	l := d.loggers[zapcore.WarnLevel]
	d.mu.RUnlock()
	l.Println("WARN:", msg, flatten(all...))
}
