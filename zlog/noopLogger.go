package zlog

import (
	"go.uber.org/zap/zapcore"
	"io"
)

// noopLogger does absolutely nothing.
type noopLogger struct{}

func (noopLogger) Info(_ string, _ ...Field)                          {}
func (noopLogger) Debug(_ string, _ ...Field)                         {}
func (noopLogger) Error(_ string, _ ...Field)                         {}
func (noopLogger) Warn(_ string, _ ...Field)                          {}
func (noopLogger) With(_ ...Field) ZLogger                            { return noopLogger{} }
func (noopLogger) Sync() error                                        { return nil }
func (noopLogger) RedirectStdLog(_ zapcore.Level) func()              { return func() {} }
func (noopLogger) RedirectOutput(_ io.Writer, _ zapcore.Level) func() { return func() {} }

var Discard ZLogger = noopLogger{}

func NewDiscard() ZLogger {
	return noopLogger{}
}
