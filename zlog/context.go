package zlog

import (
	"context"
)

// context key type for carrying ZLogger
type ctxKey struct{}

// Attach returns a new context with the provided ZLogger stored inside.
func Attach(ctx context.Context, lg ZLogger) context.Context {
	return context.WithValue(ctx, ctxKey{}, lg)
}

// FromContext retrieves a ZLogger from ctx or returns a stdlib fallback logger
// if none is present. The fallback never panics and prints key=value fields.
func FromContext(ctx context.Context) ZLogger {
	if lg, ok := ctx.Value(ctxKey{}).(ZLogger); ok && lg != nil {
		return lg
	}
	return newDefaultLogger()
}

// FromContextDiscard retrieves a ZLogger from ctx or returns a no-op logger
// (Discard) that drops all output. Useful in tests and benchmarks.
func FromContextDiscard(ctx context.Context) ZLogger {
	if lg, ok := ctx.Value(ctxKey{}).(ZLogger); ok && lg != nil {
		return lg
	}
	return NewDiscard()
}
