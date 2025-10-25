# zlog

Lightweight, production-ready logging library built on top of [Uber’s zap](https://pkg.go.dev/go.uber.org/zap).

Provides:
- Unified interface `ZLogger` for structured logs
- Dev / Prod presets (`APP_DEBUG`, `LOG_FORMAT`)
- JSON or console output
- Context helpers (`Attach`, `FromContext`)
- Safe stdlib fallback (`defaultLogger`)
- No-op logger (`Discard`)
- Stdlog redirection (`RedirectStdLog`, `RedirectOutput`)

---

## Install

```bash
go get github.com/azargarov/go-utils/zlog
```

```go
import "github.com/azargarov/go-utils/zlog"
```

---

## Quick start

```go
lg := zlog.NewDefault("my-service")
defer lg.Sync()

lg.Info("started",
    zlog.String("env", "dev"),
    zlog.Int("port", 8080),
)
```

---

## Config

```go
cfg := &zlog.Config{
    ServiceName: "my-service",
    Debug:       false,
    Format:      zlog.ZLoggerJsonFormat,
    ForceStderr: false,
}
lg := zlog.New(cfg)
```

Environment variables:

| Variable | Values | Description |
|-----------|---------|-------------|
| `APP_DEBUG` | `true` / `1` | Enables dev mode |
| `LOG_FORMAT` | `json` / `console` | Output format |

---

## Context helpers

```go
ctx := zlog.Attach(context.Background(), lg)
zlog.FromContext(ctx).Info("req", zlog.String("path", "/health"))
```

`FromContextDiscard()` returns a silent no-op logger.

---

## Fields

```go
zlog.String("user", "alice")
zlog.Int("age", 30)
zlog.Bool("ok", true)
zlog.Float64("score", 0.97)
zlog.Error("err", err)
zlog.Time("at", time.Now())
```

---

## Stdlog redirection

Redirect global `log` output:

```go
restore := lg.RedirectStdLog(zapcore.WarnLevel)
defer restore()
log.Println("stdlib message") // -> WARN in zlog
```

Redirect output to a custom writer:

```go
f, _ := os.Create("/tmp/app.log")
restore := lg.RedirectOutput(f, zapcore.InfoLevel)
defer restore()
lg.Info("written to /tmp/app.log")
```

---

## Interface

```go
type ZLogger interface {
    Info(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    Debug(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    With(fields ...Field) ZLogger
    Sync() error
    RedirectStdLog(level zapcore.Level) (restore func())
    RedirectOutput(w io.Writer, level zapcore.Level) (restore func())
}
```

---

## Fallback safety

If zap initialization fails, zlog falls back to a stdlib logger that prints
flat `key=value` pairs and never panics.

---

## Example output

**JSON**
```json
{"level":"info","timestamp":"2025-10-25T12:34:56Z","msg":"started","service":"my-service","port":8080}
```

**Console**
```
2025-10-25T12:34:56Z INFO started service=my-service port=8080
```

---

## License

MIT © Andrey Zargarov