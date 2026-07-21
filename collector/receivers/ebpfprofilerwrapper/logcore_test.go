package ebpfprofilerwrapper

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestInterpreterLogCoreDowngrade(t *testing.T) {
	obs, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(wrapInterpreterLogCore(obs))

	cases := []struct {
		msg  string
		want zapcore.Level
	}{
		{"Failed to load /usr/lib64/libperl.so.5.26.3 (0x034ab7a43f14d990): unsupported Perl 5.26.3 (need >= 5.28 and < 5.43)", zapcore.WarnLevel},
		{"Failed to load x (0x1): PHP version 7.2.0 (need >= 7.3 and < 8.5)", zapcore.WarnLevel},
		{"Failed to load x (0x2): dotnet version 5.0.0 not supported", zapcore.WarnLevel},
		{"Failed to load x (0x3): permission denied", zapcore.ErrorLevel},
		{"reporter connection failed", zapcore.ErrorLevel},
	}
	for _, c := range cases {
		logger.Error(c.msg)
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
