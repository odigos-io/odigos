package process

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func buildSyntheticEnviron(numVars int) []byte {
	var b strings.Builder
	for i := 0; i < numVars; i++ {
		switch i % 20 {
		case 0:
			fmt.Fprintf(&b, "NODE_VERSION=v%d.0.0\x00", i)
		case 1:
			fmt.Fprintf(&b, "PYTHON_VERSION=%d.11\x00", i)
		case 2:
			fmt.Fprintf(&b, "PATH=/usr/local/bin:/usr/bin:/bin%d\x00", i)
		case 3:
			fmt.Fprintf(&b, "HOME=/home/user%d\x00", i)
		default:
			fmt.Fprintf(&b, "ENV_VAR_%d=value_%d_with_some_payload_data\x00", i, i)
		}
	}
	return []byte(b.String())
}

func setupFakeProcEnviron(b *testing.B, pid int, environ []byte) (restore func()) {
	b.Helper()
	tmp := b.TempDir()
	pidDir := filepath.Join(tmp, fmt.Sprintf("%d", pid))
	if err := os.MkdirAll(pidDir, 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pidDir, "environ"), environ, 0o644); err != nil {
		b.Fatal(err)
	}
	old := procDir
	procDir = tmp
	return func() { procDir = old }
}

func BenchmarkGetRelevantEnvVars(b *testing.B) {
	cases := []struct {
		name    string
		numVars int
	}{
		{"50vars", 50},
		{"200vars", 200},
		{"500vars", 500},
	}
	runtimeDetectionEnvs := map[string]struct{}{
		"ODIGOS_SERVICE_NAME": {},
		"OTEL_SERVICE_NAME":   {},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			environ := buildSyntheticEnviron(tc.numVars)
			restore := setupFakeProcEnviron(b, 42, environ)
			defer restore()

			b.ReportAllocs()
			b.ResetTimer()
			var sink ProcessEnvs
			for i := 0; i < b.N; i++ {
				sink = getRelevantEnvVars(42, runtimeDetectionEnvs)
			}
			if sink.DetailedEnvs == nil && sink.OverwriteEnvs == nil && len(environ) == 0 {
				b.Fatal("unexpected empty result for empty environ")
			}
		})
	}
}
