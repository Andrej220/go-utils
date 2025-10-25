package zlog

import (
	"context"
)

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
	return newDefaultLogger()
}

// FromContext retrieves the ZLogger from ctx, or falls back to noop logger.
func FromContextDiscard(ctx context.Context) ZLogger {
	if lg, ok := ctx.Value(ctxKey{}).(ZLogger); ok && lg != nil {
		return lg
	}
	return NewDiscard()
}
