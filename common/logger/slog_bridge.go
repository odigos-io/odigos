package logger

import (
	"context"
	"log/slog"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// slogHandler forwards slog records to the shared zap logger with an optional subsystem.
type slogHandler struct {
	sugared *zap.SugaredLogger
	attrs   []slog.Attr
}

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
	return &slogHandler{sugared: zl.Sugar(), attrs: nil}
}

func (h *slogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return atom.Enabled(slogLevelToZap(level))
}

//nolint:gocritic // slog.Handler interface requires value receiver for record
func (h *slogHandler) Handle(_ context.Context, record slog.Record) error {
	kvs := make([]interface{}, 0, len(h.attrs)*2+record.NumAttrs()*2)
	for _, a := range h.attrs {
		kvs = append(kvs, a.Key, a.Value.Any())
	}
	record.Attrs(func(a slog.Attr) bool {
		kvs = append(kvs, a.Key, a.Value.Any())
		return true
	})
	zl := slogLevelToZap(record.Level)
	switch zl {
	case zapcore.DebugLevel:
		h.sugared.Debugw(record.Message, kvs...)
	case zapcore.InfoLevel:
		h.sugared.Infow(record.Message, kvs...)
	case zapcore.WarnLevel:
		h.sugared.Warnw(record.Message, kvs...)
	default:
		h.sugared.Errorw(record.Message, kvs...)
	}
	return nil
}

func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	newAttrs = append(newAttrs, h.attrs...)
	newAttrs = append(newAttrs, attrs...)
	return &slogHandler{sugared: h.sugared, attrs: newAttrs}
}

func (h *slogHandler) WithGroup(name string) slog.Handler {
	// Keep same handler; group name could be used to prefix keys but we don't need it for detector.
	return h
}

func slogLevelToZap(l slog.Level) zapcore.Level {
	switch {
	case l < 0:
		return zapcore.DebugLevel
	case l <= 0:
		return zapcore.InfoLevel
	case l <= 4:
		return zapcore.WarnLevel
	default:
		return zapcore.ErrorLevel
	}
}
