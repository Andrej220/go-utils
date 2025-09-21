# srvx — tiny HTTP server kit (Go)

A small, focused helper for building HTTP services with:
- sane server defaults,
- graceful shutdown on SIGINT/SIGTERM,
- pluggable logging (integrates with your `zlog`),
- a generic JSON validation middleware,
- safe request-context helpers.

> Package name: **`srvx`**. 

```go
import srvx "github.com/Andrej220/go-utils/httpsrv" 
```

---

## Install

```bash
go get github.com/Andrej220/go-utils/httpsrv@latest
```

---

## Quick start

```go
package main

import (
	"net/http"

	srvx "github.com/Andrej220/go-utils/httpsrv"
	lg   "github.com/Andrej220/go-utils/zlog"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	cfg := srvx.DefaultServerConfig(lg.NewDefault("example-service"))
	// cfg.Addr = ""   // bind all interfaces (default)
	// cfg.Port = "8081" // or from env (see EnvPortKey)
	
	if err := srvx.RunServer(mux, cfg); err != nil {
		panic(err)
	}
}
```

- If you pass `handler == nil`, `srvx` uses a **new, empty mux** (`http.NewServeMux()`).
- Server stops **gracefully** on `SIGINT`/`SIGTERM` (waits up to `ShutdownTimeout`).

---

## Configuration

```go
type ServerConfig struct {
	Port            string
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	Logger          zlog.ZLogger
	EnvPortKey      string
}
```

**Defaults** (applied via `normalize` when zero values are provided):

| Field                | Default                | Notes |
|----------------------|------------------------|------|
| `Addr`               | `""`                   | Bind all interfaces (use e.g. `127.0.0.1` for localhost only). |
| `Port`               | `"8081"` or from env   | Uses `os.Getenv(EnvPortKey)` first if set. |
| `EnvPortKey`         | `"EXECUTORPORT"`       | Override if you prefer a different env var. |
| `ReadTimeout`        | `10s`                  | |
| `ReadHeaderTimeout`  | `5s`                   | Slowloris hardening (internal default in server construction). |
| `WriteTimeout`       | `10s`                  | |
| `IdleTimeout`        | `120s`                 | |
| `ShutdownTimeout`    | `30s`                  | |

---

## Logging

`srvx` expects your `zlog.ZLogger`. If `nil`, it falls back to `zlog.NewDefault("Default")`.

Internal `net/http` errors are routed into your logger by mapping `http.Server.ErrorLog` to a stdlib logger built from your `zlog`:

```go
errorLog := zlog.StdLoggerAt(logger, zapcore.ErrorLevel)
server := &http.Server{ ErrorLog: errorLog, /* ... */ }
```

> Works with both zap-backed and fallback `defaultLogger` backends.

---

## JSON validation middleware

Bind + validate request JSON and pass the parsed value down the chain via context.

```go
type CreateUser struct {
    Name string `json:"name"`
}

validate := func(c *CreateUser) error {
    if c.Name == "" { return errors.New("name required") }
    return nil
}

mux.Handle("/users",
    srvx.NewValidationHandler[CreateUser](
        http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            body, _ := srvx.GetRequest[CreateUser](r.Context())
            // use body.Name ...
            w.WriteHeader(http.StatusCreated)
        }),
        validate,
    ),
)
```

Features:
- Enforces `Content-Type: application/json` (if present).
- Limits body size (`defaultMaxBody` = 1 MiB).
- `DisallowUnknownFields()` enabled → unknown JSON keys return an error.
- Stores `*T` in context (`WithRequest` / `GetRequest[T]`) using **per-type** keys.

---

## Error model

The middleware returns JSON errors via an `APIError` shape and **stable codes** (you define these in your package). Example suggestion:

```go
const (
    ErrCodeEmptyBody            = "empty_body"
    ErrCodeInvalidJSON          = "invalid_json"
    ErrCodeUnsupportedMediaType = "unsupported_media_type"
    ErrValidationFailed         = "validation_failed"
)

type APIError struct {
    Code    string      `json:"error"`
    Message string      `json:"message"`
    Status  int         `json:"-"`
    Details interface{} `json:"details,omitempty"`
}

func WriteJSONError(w http.ResponseWriter, e APIError) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(e.Status)
    _ = json.NewEncoder(w).Encode(e)
}
```

Usage inside middleware (already wired in your code):

```go
WriteJSONError(w, APIError{Code: ErrCodeEmptyBody, Message: "Request body is required", Status: http.StatusBadRequest})
```

---

## Context helpers

Per-type keys avoid collisions and let you retrieve the exact DTO type:

```go
ctx := srvx.WithRequest(r.Context(), &dto)      // store
dtoPtr, ok := srvx.GetRequest[MyDTO](r.Context()) // load
```

Because the key is `reqKey[T]`, different `T` types won’t overwrite each other.

---

## Testing

Recommended flags:

```bash
go test ./... -v -race -cover
```

Typical unit tests (examples are included in discussion):
- **Graceful shutdown on SIGINT** (server exits cleanly).
- **Port already in use** (returns an error from `ListenAndServe` path).
- **Validation**: unsupported content-type, empty body, unknown fields, success + context retrieval.

---
