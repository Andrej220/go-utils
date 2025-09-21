package zlog

import (
	"log"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	if zl, ok := lg.(*zapLogger); ok && zl != nil {
		if std, err := zap.NewStdLogAt(zl.l, lvl); err == nil {
			return std
		}
		return zap.NewStdLog(zl.l) 
	}
	return log.New(zlogWriter{L: lg, Level: lvl}, "", 0)
}
