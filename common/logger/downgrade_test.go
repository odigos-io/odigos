package logger

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewDowngradeCore(t *testing.T) {
	obs, logs := observer.New(zapcore.DebugLevel)
	log := zap.New(NewDowngradeCore(obs, DefaultDowngradeRules()))

	cases := []struct {
		emit zapcore.Level
		msg  string
		want zapcore.Level
	}{
		{zapcore.ErrorLevel, "Failed to load /usr/lib64/libperl.so.5.26.3 (0x0): unsupported Perl 5.26.3 (need >= 5.28 and < 5.43)", zapcore.WarnLevel},
		{zapcore.ErrorLevel, "Failed to load x (0x1): dotnet version 5.0.0 not supported", zapcore.WarnLevel},
		{zapcore.ErrorLevel, "Failed to load x (0x2): permission denied", zapcore.ErrorLevel},
		{zapcore.ErrorLevel, "reporter connection refused", zapcore.ErrorLevel},
		{zapcore.WarnLevel, "Failed to load x (0x3): unsupported Perl", zapcore.WarnLevel},
		{zapcore.InfoLevel, "starting", zapcore.InfoLevel},
	}
	for _, c := range cases {
		switch c.emit {
		case zapcore.ErrorLevel:
			log.Error(c.msg)
		case zapcore.WarnLevel:
			log.Warn(c.msg)
		default:
			log.Info(c.msg)
		}
	}

	got := map[string]zapcore.Level{}
	for _, e := range logs.All() {
		got[e.Message] = e.Level
	}
	for _, c := range cases {
		if got[c.msg] != c.want {
			t.Errorf("msg %q: level=%v want %v", c.msg, got[c.msg], c.want)
		}
	}
}

func TestNewDowngradeCoreNoRules(t *testing.T) {
	obs, _ := observer.New(zapcore.DebugLevel)
	if NewDowngradeCore(obs, nil) != obs {
		t.Fatal("empty rules should return the inner core unchanged")
	}
}
