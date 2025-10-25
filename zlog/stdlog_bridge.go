package zlog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"log"
	"strings"
)

type zlogWriter struct {
	L     ZLogger
	Level zapcore.Level
}

func (w zlogWriter) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))
	switch w.Level {
	case zapcore.DebugLevel:
		w.L.Debug(msg)
	case zapcore.WarnLevel:
		w.L.Warn(msg)
	case zapcore.InfoLevel:
		w.L.Info(msg)
	default:
		w.L.Error(msg)
	}
	return len(p), nil
}

func StdLoggerAt(lg ZLogger, lvl zapcore.Level) *log.Logger {
	if zl, ok := lg.(*zLog); ok && zl != nil {
		if std, err := zap.NewStdLogAt(zl.l, lvl); err == nil {
			return std
		}
		return zap.NewStdLog(zl.l)
	}
	return log.New(zlogWriter{L: lg, Level: lvl}, "", 0)
}

func RedirectStdLogger(lg ZLogger, lvl zapcore.Level) (restore func()) {
	prevOut, prevFlags, prevPrefix := log.Writer(), log.Flags(), log.Prefix()
	std := StdLoggerAt(lg, lvl)
	log.SetOutput(std.Writer())
	log.SetFlags(0)
	log.SetPrefix("")
	return func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
		log.SetPrefix(prevPrefix)
	}
}

func RedirectStdLogOutput(w io.Writer) (restore func()) {
	prevOut := log.Writer()
	prevFlags := log.Flags()
	prevPrefix := log.Prefix()

	if w == nil {
		w = io.Discard
	}
	log.SetOutput(w)
	log.SetFlags(0)
	log.SetPrefix("")

	return func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
		log.SetPrefix(prevPrefix)
	}
}
