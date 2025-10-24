package srvx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	lg "github.com/azargarov/go-utils/zlog"
	"go.uber.org/zap/zapcore"
)

// ServerConfig holds configuration of HTTP server.
type ServerConfig struct {
	Port            string
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	Logger          lg.ZLogger
	EnvPortKey      string
}

const (
	defaultPort              = "8081"
	defaultAddr              = ""
	defaultReadTimeout       = 10 * time.Second
	defaultReadHeaderTimeout = 5 * time.Second
	defaultWriteTimeout      = 10 * time.Second
	defaultIdleTimeout       = 120 * time.Second
	defaultShutdownTimeout   = 30 * time.Second
	defaultMaxBody           = 1 << 20
	defaultEnvPortKey        = "EXECUTORPORT"
)

// Per-type key variant
type reqKey[T any] struct{}

func WithRequest[T any](ctx context.Context, v *T) context.Context {
	return context.WithValue(ctx, reqKey[T]{}, v)
}

func GetRequest[T any](ctx context.Context) (*T, bool) {
	v, ok := ctx.Value(reqKey[T]{}).(*T)
	return v, ok
}

func normalize(c ServerConfig) ServerConfig {
	if c.EnvPortKey == "" {
		c.EnvPortKey = defaultEnvPortKey
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = defaultReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = defaultWriteTimeout
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = defaultIdleTimeout
	}
	if c.ShutdownTimeout == 0 {
		c.ShutdownTimeout = defaultShutdownTimeout
	}
	if c.Port == "" {
		if p := os.Getenv(c.EnvPortKey); p != "" {
			c.Port = p
		} else {
			c.Port = defaultPort
		}
	}
	if c.Addr == "" {
		c.Addr = defaultAddr
	}
	return c
}

func DefaultServerConfig(l lg.ZLogger) ServerConfig {
	return ServerConfig{
		Port:            defaultPort,
		ReadTimeout:     defaultReadTimeout,
		WriteTimeout:    defaultWriteTimeout,
		IdleTimeout:     defaultIdleTimeout,
		ShutdownTimeout: defaultShutdownTimeout,
		Logger:          l,
		EnvPortKey:      defaultEnvPortKey,
	}
}

func RunServer(handler http.Handler, config ServerConfig) error {
	// DONE: pass listening port with environment variable, for different services...

	//if handler is nil, the server uses default one
	if handler == nil {
		handler = http.NewServeMux()
	}

	config = normalize(config)
	var logger lg.ZLogger
	if config.Logger != nil {
		logger = config.Logger
	} else {
		logger = lg.NewDefault("Default")
	}

	srvAddr := net.JoinHostPort(config.Addr, config.Port)
	errorLog := lg.StdLoggerAt(logger, zapcore.ErrorLevel)

	server := &http.Server{
		Addr:              srvAddr,
		Handler:           handler,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ErrorLog:          errorLog,
	}
	// Channel to listen interrupt signals
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigc)

	serveErr := make(chan error, 1)

	go func() {
		logger.Info("Server starting", lg.String("addr", srvAddr))
		serveErr <- server.ListenAndServe()
	}()

	select {
	case sig := <-sigc:
		logger.Info("shutdown signal", lg.String("signal", sig.String()))
	case err := <-serveErr:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	defer cancel()

	// Attempt gracefully shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown failed", lg.Any("error", err))
		return err
	}

	// drain serveErr
	select {
	case err := <-serveErr:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server listener error", lg.Any("error", err))
			return err
		}
	default:
	}

	logger.Info("Server stopped gracefully")
	return nil
}

type ValidationHandler[T any] struct {
	next      http.Handler
	validator func(*T) error
}

func NewValidationHandler[T any](next http.Handler, validator ...func(*T) error) http.Handler {
	// DONE: implement a default validator
	var validateFunc func(*T) error
	if len(validator) > 0 {
		validateFunc = validator[0]
	} else {
		validateFunc = defaultValidator[T]
	}

	return &ValidationHandler[T]{
		next:      next,
		validator: validateFunc,
	}
}

func (h *ValidationHandler[T]) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	var request T

	if ct := r.Header.Get("Content-Type"); ct != "" {
		mt, _, _ := mime.ParseMediaType(ct)
		if mt != "application/json" {
			WriteJSONError(rw, APIError{Code: ErrCodeUnsupportedMediaType, Message: "unsupported_media_type", Status: http.StatusUnsupportedMediaType})
			return
		}
	}
	if r.Body == nil {
		WriteJSONError(rw, APIError{Code: ErrCodeEmptyBody, Message: "Request body is required", Status: http.StatusBadRequest})

		return
	}
	defer r.Body.Close()

	r.Body = http.MaxBytesReader(rw, r.Body, defaultMaxBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&request)
	if err != nil {
		if errors.Is(err, io.EOF) {
			WriteJSONError(rw, APIError{Code: ErrCodeEmptyBody, Message: "Request body is required", Status: http.StatusBadRequest})
			return
		}
		WriteJSONError(rw, APIError{Code: ErrCodeInvalidJSON, Message: "invalid_json", Status: http.StatusBadRequest})
		return
	}

	if err := h.validator(&request); err != nil {
		WriteJSONError(rw, APIError{Code: ErrValidationFailed, Message: "validation_failed", Status: http.StatusBadRequest})
		return
	}
	// Pass the decoded request to the next handler via context
	ctx := WithRequest(r.Context(), &request)
	h.next.ServeHTTP(rw, r.WithContext(ctx))
}

func defaultValidator[T any](req *T) error {
	return nil
}
