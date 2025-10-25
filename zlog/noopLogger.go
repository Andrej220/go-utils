package zlog

import (
	"go.uber.org/zap/zapcore"
	"io"
)

// noopLogger implements ZLogger but discards all logs.
//
// It’s useful when you need to satisfy a logging dependency without
// producing any output (e.g. in tests, benchmarks, or disabled components).
type noopLogger struct{}

// Ensure that noopLogger satisfies the ZLogger interface at compile time.
var _ ZLogger = (*noopLogger)(nil)

func (noopLogger) Info(_ string, _ ...Field)                          {}
func (noopLogger) Debug(_ string, _ ...Field)                         {}
func (noopLogger) Error(_ string, _ ...Field)                         {}
func (noopLogger) Warn(_ string, _ ...Field)                          {}
func (noopLogger) With(_ ...Field) ZLogger                            { return noopLogger{} }
func (noopLogger) Sync() error                                        { return nil }
func (noopLogger) RedirectStdLog(_ zapcore.Level) func()              { return func() {} }
func (noopLogger) RedirectOutput(_ io.Writer, _ zapcore.Level) func() { return func() {} }

// Discard is a ZLogger that drops all logs. It can be used globally.
var Discard ZLogger = noopLogger{}

// NewDiscard returns a new no-op ZLogger that drops all logs.
func NewDiscard() ZLogger {
	return noopLogger{}
}
