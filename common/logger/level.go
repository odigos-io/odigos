package logger

import (
	"strings"

	"go.uber.org/zap/zapcore"
)

type OdigosLogLevel string

const (
	LevelError OdigosLogLevel = "error"
	LevelWarn  OdigosLogLevel = "warn"
	LevelInfo  OdigosLogLevel = "info"
	LevelDebug OdigosLogLevel = "debug"
)

// ParseLevel converts a string to zapcore.Level. Accepted: error, warn, info, debug.
func ParseLevel(lvl string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(lvl)) {
	case "error":
		return zapcore.ErrorLevel
	case "warn":
		return zapcore.WarnLevel
	case "info":
		return zapcore.InfoLevel
	case "debug":
		return zapcore.DebugLevel
	default:
		return zapcore.InfoLevel
	}
}
