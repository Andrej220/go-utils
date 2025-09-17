package zlog

import (
    "context"
    "log"
    "os"
    "strings"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "time"
    "slices"
    "fmt"
)

const(
    ZLoggerJsonFormat = "json"
    ZLoggerConsoleFormat = "console"
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
}

// Config holds logging configuration options.
type Config struct {
    ServiceName string 
    Debug       bool   
    Format      string // "json" or "console"
}

func DebugFromEnv() bool {
    return os.Getenv("APP_DEBUG") == "true" || 
           os.Getenv("APP_DEBUG") == "1" ||
           strings.EqualFold(os.Getenv("APP_DEBUG"), "true")
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
    baseCfg.Sampling = &zap.SamplingConfig{Initial: 100, Thereafter: 100}

    logger, err := baseCfg.Build(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
    if err != nil {
        // Fall back to standard log if zap fails
        log.Printf("[FATAL] cannot initialize zap logger: %v", err)
        return defaultLogger{}
    }

    return &zapLogger{l: logger}
}

// NewDefault creates a logger with default configuration
func NewDefault(serviceName string) ZLogger {
    return New(&Config{
        ServiceName: serviceName,
        Debug:       DebugFromEnv(),
        Format:      FormatFromEnv(ZLoggerJsonFormat),
    })
}

// zapLogger wraps a *zap.ZLogger to implement ZLogger.
type zapLogger struct{ l *zap.Logger }

func (z *zapLogger) Info(msg string, fields ...Field) {
    z.l.Info(msg, fields...)
}

func (z *zapLogger) Error(msg string, fields ...Field) {
    z.l.Error(msg, fields...)
}

func (z *zapLogger) With(fields ...Field) ZLogger {
    return &zapLogger{z.l.With(fields...)}
}

func (z *zapLogger) Sync() error {
    return z.l.Sync()
}

func (z *zapLogger) Debug(msg string, fields ...Field){
     z.l.Debug(msg, fields...)
}
func (z *zapLogger) Warn(msg string, fields ...Field){
    z.l.Warn(msg, fields...)
}


// defaultLogger falls back to the standard log package.
type defaultLogger struct{
    base []Field
}

func (d defaultLogger) Info(msg string, fields ...Field) {
    all := append(slices.Clone(d.base), fields...)
    log.Println("INFO:", msg, flatten(all...))
}

func (d defaultLogger) Error(msg string, fields ...Field) {
    all := append(slices.Clone(d.base), fields...)
    log.Println("ERROR:", msg, flatten(all...))
}

func (d defaultLogger) With(fields ...Field) ZLogger {  
    all := append(slices.Clone(d.base), fields...)
    return defaultLogger{base: all}
}

func (d defaultLogger) Sync() error { return nil }
func (d defaultLogger) Debug(msg string, fields ...Field){
    all := append(slices.Clone(d.base), fields...)
    log.Println("DEBUG:", msg, flatten(all...))
}
func (d defaultLogger) Warn(msg string, fields ...Field){
    all := append(slices.Clone(d.base), fields...)
    log.Println("WARN:", msg, flatten(all...))
}


func flatten(fields ...zapcore.Field) string {
    // assert types
    enc := zapcore.NewMapObjectEncoder()
	for _, f := range fields { f.AddTo(enc) }

    if len(enc.Fields) == 0 { return "" }

    pairs := make([]string, 0, len(fields))
    for k, v := range enc.Fields { 
        pairs = append( pairs, fmt.Sprintf("%s=%v", k, v) )
    }
    return strings.Join(pairs, ", ")
}

// context key type for carrying ZLogger
type ctxKey struct{}

// Attach returns a new context with the provided ZLogger.
func Attach(ctx context.Context, lg ZLogger) context.Context {
    return context.WithValue(ctx, ctxKey{}, lg)
}

// FromContext retrieves the ZLogger from ctx, or falls back to defaultLogger.
func FromContext(ctx context.Context) ZLogger {
    if lg, ok := ctx.Value(ctxKey{}).(ZLogger); ok && lg != nil {
        return lg
    }
    return defaultLogger{}
}

// noopLogger does absolutely nothing. For test only
type noopLogger struct{}
func (noopLogger) Info(msg string, _ ...Field) {}
func (noopLogger) Debug(msg string, _ ...Field) {}
func (noopLogger) Error(msg string, _ ...Field) {}
func (noopLogger) Warn(msg string, _ ...Field) {}
func (noopLogger) With(_ ...Field) ZLogger { return noopLogger{} }
func (noopLogger) Sync() error { return nil }
var Discard ZLogger = noopLogger{}

//func _flatten(fields ...zapcore.Field) string {
//	enc := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{ConsoleSeparator: " "})
//	buf, _ := enc.EncodeEntry(zapcore.Entry{}, fields)
//	defer buf.Free()
//	return strings.TrimSpace(buf.String())
//}