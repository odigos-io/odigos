package logger

import (
	"context"
)

type contextLoggerKey struct{}

// IntoContext stores an OdigosLogger in the context for downstream usage.
func IntoContext(ctx context.Context, l *OdigosLogger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if l == nil {
		l = FromContext(ctx)
	}
	return context.WithValue(ctx, contextLoggerKey{}, l)
}

// FromContext returns the OdigosLogger stored in the context, or the global LoggerCompat() if none.
func FromContext(ctx context.Context) *OdigosLogger {
	if ctx != nil {
		if l, ok := ctx.Value(contextLoggerKey{}).(*OdigosLogger); ok && l != nil {
			return l
		}
	}
	return LoggerCompat()
}
