package ebpfprofilerwrapper

import (
	"strings"

	"go.uber.org/zap/zapcore"
)

// wrapInterpreterLogCore wraps a zapcore.Core so that the eBPF profiler's
// "unsupported interpreter version" load errors are emitted as warnings rather
// than errors. Genuine load failures keep their error level.
func wrapInterpreterLogCore(inner zapcore.Core) zapcore.Core {
	return &interpreterLogCore{Core: inner}
}

type interpreterLogCore struct {
	zapcore.Core
}

func (c *interpreterLogCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if ent.Level == zapcore.ErrorLevel && isUnsupportedInterpreter(ent.Message) {
		ent.Level = zapcore.WarnLevel
	}
	if c.Core.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *interpreterLogCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	return c.Core.Write(ent, fields)
}

func (c *interpreterLogCore) With(fields []zapcore.Field) zapcore.Core {
	return &interpreterLogCore{Core: c.Core.With(fields)}
}

func isUnsupportedInterpreter(msg string) bool {
	if !strings.Contains(msg, "Failed to load") {
		return false
	}
	return strings.Contains(msg, "unsupported") ||
		strings.Contains(msg, "not supported") ||
		strings.Contains(msg, "need >=")
}
