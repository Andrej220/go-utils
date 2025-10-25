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
	ZLoggerJsonFormat    = "json"
	ZLoggerConsoleFormat = "console"
	samplingInitial      = 100
	samplingAfter        = 100
)

// Field is a structured log field, aliasing zapcore.Field
type Field = zapcore.Field

func Any(key string, value any) Field         { return zap.Any(key, value) }
func String(key, value string) Field          { return zap.String(key, value) }
func Int(key string, value int) Field         { return zap.Int(key, value) }
func Int32(key string, value int32) Field     { return zap.Int32(key, value) }
func Bool(key string, value bool) Field       { return zap.Bool(key, value) }
func Float64(key string, value float64) Field { return zap.Float64(key, value) }
func Time(key string, value time.Time) Field  { return zap.Time(key, value) }
func Error(key string, value error) Field     { return zap.NamedError(key, value) }

// ZLogger defines the minimal interface for structured logging.
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

// Config holds logging configuration options.
type Config struct {
	ServiceName string
	Debug       bool
	Format      string // "json" or "console"
	ForceStderr bool   // route all logs to stderr
}

func DebugFromEnv() bool {
	v := os.Getenv("APP_DEBUG")
	return v == "1" || strings.EqualFold(v, "true")
}

func FormatFromEnv(defaultFormat string) string {
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		return format
	}
	if defaultFormat == "" {
		return ZLoggerJsonFormat
	}
	return defaultFormat
}

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

// NewDefault creates a logger with default configuration
func NewDefault(serviceName string) ZLogger {
	return New(&Config{
		ServiceName: serviceName,
		Debug:       DebugFromEnv(),
		Format:      FormatFromEnv(ZLoggerJsonFormat),
	})
}

// zLog wraps a *zap.ZLogger to implement ZLogger.
type zLog struct{ l *zap.Logger }

func (z *zLog) Info(msg string, fields ...Field) {
	z.l.Info(msg, fields...)
}

func (z *zLog) Error(msg string, fields ...Field) {
	z.l.Error(msg, fields...)
}

func (z *zLog) With(fields ...Field) ZLogger {
	return &zLog{z.l.With(fields...)}
}

func (z *zLog) Sync() error {
	return z.l.Sync()
}

func (z *zLog) Debug(msg string, fields ...Field) {
	z.l.Debug(msg, fields...)
}
func (z *zLog) Warn(msg string, fields ...Field) {
	z.l.Warn(msg, fields...)
}

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
