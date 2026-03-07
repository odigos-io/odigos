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
	LevelTrace OdigosLogLevel = "trace"
)

// ParseLevel converts a string to zapcore.Level.
// "trace" => -5 (captures ctrl-runtime V(4)/V(5)).
func ParseLevel(lvl string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(lvl)) {
	case "error":
		return zapcore.ErrorLevel // 2
	case "warn":
		return zapcore.WarnLevel // 1
	case "info":
		return zapcore.InfoLevel // 0
	case "debug":
		return zapcore.DebugLevel // -1
	case "trace":
		return zapcore.Level(-5) // -5
	default:
		return zapcore.InfoLevel
	}
}
