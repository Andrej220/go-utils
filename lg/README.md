# zlog

Minimal wrapper around Uber’s [`zap`](https://pkg.go.dev/go.uber.org/zap) that provides:
- Interface: `ZLogger`
- Dev/Prod presets (env-driven)
- JSON or console output
- Context helpers (`Attach`, `FromContext`)
- Safe fallback logger using stdlib `log` with flat `k=v` fields
- `Discard` no-op logger for tests

---

## Install

```bash
go get github.com/Andrej220/go-utils/zlog
```

```go
import "github.com/Andrej220/go-utils/zlog"
```

## Quick start

```go
lg := zlog.NewDefault("my-service")
defer lg.Sync()

lg.Info("started",
    zlog.String("env", "dev"),
    zlog.Int("port", 8080),
)
```

### Custom config

```go
cfg := &zlog.Config{
    ServiceName: "my-service",
    Debug:       false,                       // or true
    Format:      zlog.ZLoggerJsonFormat,      // or zlog.ZLoggerConsoleFormat
}
lg := zlog.New(cfg)
defer lg.Sync()
```

### Context usage

```go
ctx := zlog.Attach(context.Background(), lg)
zlog.FromContext(ctx).Info("request",
    zlog.String("path", "/health"),
)
```

## Fields

Typed helpers (recommended):
```go
zlog.String("user", "alice")
zlog.Int("age", 30)
zlog.Bool("ok", true)
zlog.Float64("score", 0.97)
zlog.Time("at", time.Now())
zlog.Error("err", err)
zlog.Any("raw", map[string]any{"k":"v"}) // may be JSON-ish in console output
```

## Formats

- `json` (default) — machine-friendly.
- `console` — human-friendly (set `LOG_FORMAT=console`).

> The fallback stdlib logger prints fields as `key=value key2=value2` via an internal `flatten` helper.

## Environment variables

- `APP_DEBUG`: `true` / `1` ⇒ dev mode (higher verbosity).
- `LOG_FORMAT`: `json` | `console`.

`NewDefault()` reads both.

## Testing

No-op logger:
```go
lg := zlog.Discard
lg.Info("no output", zlog.String("k","v"))
```

Use your logger in tests:
```go
func TestSomething(t *testing.T) {
    lg := zlog.NewDefault("test")
    defer lg.Sync()
    // ...
}
```

## Interface

```go
type ZLogger interface {
    Info(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    Debug(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    With(fields ...Field) ZLogger
    Sync() error
}
```

## Notes

- `With(...)` attaches base fields; the fallback logger preserves them too.


