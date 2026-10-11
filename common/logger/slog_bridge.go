package logger

import (
	"log/slog"

	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

// NewSlogHandler returns a slog.Handler that writes to the Odigos zap logger with the given subsystem.
// Use it to pass a *slog.Logger to libraries that require slog (e.g. runtime-detector) so logs
// include component and subsystem and respect ODIGOS_LOG_LEVEL.
func NewSlogHandler(subsystem string) slog.Handler {
	zl := Logger()
	if zl == nil {
		zl = zap.NewNop()
	}
	if subsystem != "" {
		zl = zl.With(zap.String("subsystem", subsystem))
	}
	return zapslog.NewHandler(zl.Core(), zapslog.WithCaller(true))
}
