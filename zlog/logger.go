// Package zlog provides a minimal, production-ready logging facade over Uber's
// zap. It exposes a small interface (ZLogger) that works with a zap-backed
// implementation in production and a safe stdlib fallback when zap cannot be
// initialized.
//
// Key features:
//   - Simple interface: Info, Warn, Error, Debug, With, Sync
//   - Dev/Prod presets via env: APP_DEBUG, LOG_FORMAT
//   - JSON or console encoders
//   - Context helpers: Attach, FromContext, FromContextDiscard
//   - Stdlib integration: redirect the global log package to zlog
//   - No-op logger: Discard
//
// Environment variables:
//
//	APP_DEBUG  = "true" | "1" (enables development mode)
//	LOG_FORMAT = "json" | "console"
//
// Quick start:
//
//	lg := zlog.NewDefault("my-service")
//	defer lg.Sync()
//	lg.Info("started", zlog.String("port", "8080"))
//
// Redirect the global log package at a specific level:
//
//	restore := lg.RedirectStdLog(zapcore.WarnLevel)
//	defer restore()
//	log.Println("this becomes WARN in zlog")
//
// Context usage:
//
//	ctx := zlog.Attach(context.Background(), lg)
//	zlog.FromContext(ctx).Info("request", zlog.String("path", "/health"))
package zlog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

const (
	// ZLoggerJsonFormat selects JSON encoding for logs.
	ZLoggerJsonFormat = "json"
	// ZLoggerConsoleFormat selects console (human-friendly) encoding for logs.
	ZLoggerConsoleFormat = "console"
	samplingInitial      = 100
	samplingAfter        = 100
)

// Field is a structured log field, aliasing zapcore.Field.
// Use helper constructors like String, Int, Bool, etc. to create fields.
type Field = zapcore.Field

// String constructs a string field
func Any(key string, value any) Field { return zap.Any(key, value) }

// String constructs a string field.
func String(key, value string) Field { return zap.String(key, value) }

// Int constructs an int field.
func Int(key string, value int) Field { return zap.Int(key, value) }

// Int32 constructs an int32 field.
func Int32(key string, value int32) Field { return zap.Int32(key, value) }

// Bool constructs a bool field.
func Bool(key string, value bool) Field { return zap.Bool(key, value) }

// Float64 constructs a float64 field.
func Float64(key string, value float64) Field { return zap.Float64(key, value) }

// Time constructs a time.Time field.
func Time(key string, value time.Time) Field { return zap.Time(key, value) }

// Error constructs a named error field.
func Error(key string, value error) Field { return zap.NamedError(key, value) }

// ZLogger is the minimal interface implemented by zlog's backends.
// It supports structured fields, contextual enrichment via With, and Sync.
type ZLogger interface {
	Info(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	With(fields ...Field) ZLogger
	Sync() error
	Debug(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	RedirectStdLog(level zapcore.Level) (restore func())
	RedirectOutput(w io.Writer, level zapcore.Level) (restore func())
}

// Config holds logging configuration options for New.
type Config struct {
	// ServiceName is injected as initial structured field "service".
	ServiceName string
	// Debug enables development mode (colorized console, debug level).
	Debug bool
	// Format selects encoder: "json" or "console".
	Format string // "json" or "console"
	// ForceStderr routes all output to stderr when true.
	ForceStderr bool // route all logs to stderr
}

// DebugFromEnv returns true if APP_DEBUG is "true" (case-insensitive) or "1".
func DebugFromEnv() bool {
	v := os.Getenv("APP_DEBUG")
	return v == "1" || strings.EqualFold(v, "true")
}

// FormatFromEnv returns LOG_FORMAT if set; otherwise defaultFormat or "json".
func FormatFromEnv(defaultFormat string) string {
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		return format
	}
	if defaultFormat == "" {
		return ZLoggerJsonFormat
	}
	return defaultFormat
}

// New builds a zap-backed ZLogger using cfg. If zap initialization fails,
// New returns a stdlib-backed fallback logger that never panics.
func New(cfg *Config) ZLogger {
	var baseCfg zap.Config

	if !cfg.Debug {
		cfg.Debug = DebugFromEnv()
	}

	if cfg.Debug {
		baseCfg = zap.NewDevelopmentConfig()
		baseCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		baseCfg = zap.NewProductionConfig()
		baseCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Allow console or JSON output
	baseCfg.Encoding = FormatFromEnv(cfg.Format)
	baseCfg.EncoderConfig.TimeKey = "timestamp"
	baseCfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	baseCfg.InitialFields = map[string]any{"service": cfg.ServiceName}

	// Enable sampling for high-throughput logs
	baseCfg.Sampling = &zap.SamplingConfig{Initial: samplingInitial, Thereafter: samplingAfter}

	if cfg.ForceStderr {
		baseCfg.OutputPaths = []string{"stderr"}
		baseCfg.ErrorOutputPaths = []string{"stderr"}
	}

	logger, err := baseCfg.Build(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		// Fall back to standard log if zap fails
		log.Printf("[FATAL] cannot initialize zap logger: %v", err)
		return newDefaultLogger()
	}

	return &zLog{l: logger}
}

// NewDefault creates a logger with defaults derived from environment variables.
// It sets the "service" field to serviceName.
func NewDefault(serviceName string) ZLogger {
	return New(&Config{
		ServiceName: serviceName,
		Debug:       DebugFromEnv(),
		Format:      FormatFromEnv(ZLoggerJsonFormat),
	})
}

// zLog wraps a *zap.ZLogger to implement ZLogger.
type zLog struct{ l *zap.Logger }

// Info logs msg at Info level with optional structured fields.
func (z *zLog) Info(msg string, fields ...Field) {
	z.l.Info(msg, fields...)
}

// Error logs msg at Error level with optional structured fields.
func (z *zLog) Error(msg string, fields ...Field) {
	z.l.Error(msg, fields...)
}

// With returns a child logger enriched with fields that will be included
// on every subsequent log call from the returned logger.
func (z *zLog) With(fields ...Field) ZLogger {
	return &zLog{z.l.With(fields...)}
}

// Sync flushes any buffered log entries. It should be called before process exit.
func (z *zLog) Sync() error {
	return z.l.Sync()
}

// Debug logs msg at Debug level with optional structured fields.
func (z *zLog) Debug(msg string, fields ...Field) {
	z.l.Debug(msg, fields...)
}

// Warn logs msg at Warn level with optional structured fields.
func (z *zLog) Warn(msg string, fields ...Field) {
	z.l.Warn(msg, fields...)
}

// RedirectStdLog redirects the global log package to this logger at the given level.
// It returns a restore function that reverts log's output, flags, and prefix.
func (z *zLog) RedirectStdLog(level zapcore.Level) (restore func()) {

	prevOut := log.Writer()
	prevFlags := log.Flags()
	prevPrefix := log.Prefix()

	std := StdLoggerAt(z, level)
	log.SetOutput(std.Writer())
	log.SetFlags(0)
	log.SetPrefix("")

	return func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
		log.SetPrefix(prevPrefix)
	}
}

// RedirectOutput routes this logger's output at the given level to w by rebuilding
// the underlying zap core with a JSON encoder and a level enabler set to 'level'.
// It returns a restore function that restores the previous core.
func (z *zLog) RedirectOutput(w io.Writer, level zapcore.Level) (restore func()) {
	if w == nil {
		w = io.Discard
	}
	old := z.l

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.RFC3339TimeEncoder

	lvl := zap.NewAtomicLevelAt(level)
	newCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encCfg),
		zapcore.AddSync(w),
		lvl,
	)

	z.l = old.WithOptions(zap.WrapCore(func(_ zapcore.Core) zapcore.Core {
		return newCore
	}))

	return func() {
		z.l = old
	}
}
