package logger

import (
	"context"
	"log/slog"
)

type contextLoggerKey struct{}

// IntoContext stores a slog logger in the context for downstream usage
func IntoContext(ctx context.Context, l *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if l == nil {
		l = FromContext(ctx)
	}
	return context.WithValue(ctx, contextLoggerKey{}, l)
}

// FromContext returns the slog logger stored in the context
func FromContext(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if l, ok := ctx.Value(contextLoggerKey{}).(*slog.Logger); ok && l != nil {
			return l
		}
	}
	if l := Logger(); l != nil {
		return l
	}
	return slog.Default()
}
