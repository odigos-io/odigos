package logger

import (
	"strings"

	"go.uber.org/zap/zapcore"
)

// DowngradeRule lowers the level of a log record whose message contains all of
// Contains and at least one of AnyOf (AnyOf empty means "no additional match").
// It exists to keep expected, fail-safe conditions from surfacing as errors —
// e.g. the eBPF profiler recognizing an interpreter whose runtime version it does
// not support, where native profiling still works and only interpreter frames are
// missing. Only records at error level or above are considered.
type DowngradeRule struct {
	Contains []string
	AnyOf    []string
	To       zapcore.Level
}

func (r DowngradeRule) matches(msg string) bool {
	for _, s := range r.Contains {
		if !strings.Contains(msg, s) {
			return false
		}
	}
	if len(r.AnyOf) == 0 {
		return true
	}
	for _, s := range r.AnyOf {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// DefaultDowngradeRules are applied to every logger built by Init and are shared
// with the collector. Kept intentionally narrow so real errors are never hidden.
func DefaultDowngradeRules() []DowngradeRule {
	return []DowngradeRule{
		// eBPF profiler: recognized interpreter, unsupported runtime version (CORE-1110).
		{Contains: []string{"Failed to load"}, AnyOf: []string{"unsupported", "not supported", "need >="}, To: zapcore.WarnLevel},
	}
}

// NewDowngradeCore wraps inner so records matching a rule are emitted at the rule's
// target level. Non-matching records and records below error level pass through
// unchanged. Reused by k8s components, vm-agent and the collector for one behavior.
func NewDowngradeCore(inner zapcore.Core, rules []DowngradeRule) zapcore.Core {
	if len(rules) == 0 {
		return inner
	}
	return &downgradeCore{Core: inner, rules: rules}
}

type downgradeCore struct {
	zapcore.Core
	rules []DowngradeRule
}

func (c *downgradeCore) target(ent zapcore.Entry) zapcore.Level {
	if ent.Level < zapcore.ErrorLevel {
		return ent.Level
	}
	for _, r := range c.rules {
		if r.matches(ent.Message) {
			return r.To
		}
	}
	return ent.Level
}

func (c *downgradeCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	ent.Level = c.target(ent)
	if c.Core.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *downgradeCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	return c.Core.Write(ent, fields)
}

func (c *downgradeCore) With(fields []zapcore.Field) zapcore.Core {
	return &downgradeCore{Core: c.Core.With(fields), rules: c.rules}
}
