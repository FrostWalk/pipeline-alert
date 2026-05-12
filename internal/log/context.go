package log

import (
	"context"

	"go.uber.org/zap"
)

type loggerContextKey struct{}

// ContextWithLogger stores logger in request context.
func ContextWithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// LoggerFromContext returns logger from request context, or no-op logger when missing.
func LoggerFromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(loggerContextKey{}).(*zap.Logger); ok && logger != nil {
		return logger
	}
	return zap.NewNop()
}
