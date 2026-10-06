package utils

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

type memProcessFile struct {
	*bytes.Reader
}

func (m memProcessFile) Close() error { return nil }

func buildSyntheticMaps(lines int, hitBinary string, hitAt int) []byte {
	var b strings.Builder
	for i := 0; i < lines; i++ {
		if i == hitAt && hitBinary != "" {
			b.WriteString("7f8a1c000000-7f8a1c200000 r-xp 00000000 08:01 123456 /usr/lib/")
			b.WriteString(hitBinary)
			b.WriteString("\n")
			continue
		}
		b.WriteString("7f8a1c000000-7f8a1c200000 r-xp 00000000 08:01 123456 /usr/lib/x86_64-linux-gnu/libc.so.6\n")
	}
	return []byte(b.String())
}

func BenchmarkIsMapsFileContainsBinary(b *testing.B) {
	cases := []struct {
		name     string
		lines    int
		hitAt    int
		binaries []string
	}{
		{"miss_200", 200, -1, []string{"libpython", "libjvm.so", "libruby.so"}},
		{"hit_early_200", 200, 5, []string{"libpython", "libjvm.so", "libruby.so"}},
		{"miss_2000", 2000, -1, []string{"libpython", "libjvm.so", "libruby.so"}},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			data := buildSyntheticMaps(tc.lines, "libpython3.11.so.1.0", tc.hitAt)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				f := memProcessFile{bytes.NewReader(data)}
				_ = IsMapsFileContainsBinary(f, tc.binaries)
			}
		})
	}
}

// Ensure memProcessFile satisfies the small interface used by the helper.
var _ io.ReadSeekCloser = memProcessFile{}
var _ io.ReaderAt = memProcessFile{}
