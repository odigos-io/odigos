package rust

import (
	"strings"
	"testing"
)

func Test_extractRustcCommitHash(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected string
	}{
		{
			name:     "panicking.rs path",
			data:     "/rustc/79e9716c980570bfd1f666e3b16ac583f0168962/library/std/src/panicking.rs",
			expected: "79e9716c980570bfd1f666e3b16ac583f0168962",
		},
		{
			name:     "hash embedded among other binary data",
			data:     "garbage\x00\x01/rustc/07dca489ac2d933c78d3c5158e3f43beefeb02ce/library/std/src/rt.rs\x00more garbage",
			expected: "07dca489ac2d933c78d3c5158e3f43beefeb02ce",
		},
		{
			name:     "no match",
			data:     "this binary has no rustc path in it",
			expected: "",
		},
		{
			name:     "hash too short is not matched",
			data:     "/rustc/deadbeef/library/std/src/panicking.rs",
			expected: "",
		},
		{
			name:     "empty input",
			data:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRustcCommitHash([]byte(tt.data))
			if result != tt.expected {
				t.Errorf("extractRustcCommitHash(%q) = %q; want %q", tt.data, result, tt.expected)
			}
		})
	}
}

func Test_scanForRustcCommitHashChunked(t *testing.T) {
	const hash = "79e9716c980570bfd1f666e3b16ac583f0168962"
	needle := "/rustc/" + hash + "/library/std/src/panicking.rs"

	tests := []struct {
		name      string
		data      string
		chunkSize int
		expected  string
	}{
		{
			name:      "match fits in a single chunk",
			data:      "garbage before " + needle + " garbage after",
			chunkSize: 4096,
			expected:  hash,
		},
		{
			name:      "match straddles a chunk boundary",
			data:      "garbage before " + needle + " garbage after",
			chunkSize: 8, // smaller than the pattern itself
			expected:  hash,
		},
		{
			name:      "match straddles a boundary with tiny chunks",
			data:      needle,
			chunkSize: 1,
			expected:  hash,
		},
		{
			name:      "no match, chunked",
			data:      strings.Repeat("no rustc path here, just filler bytes. ", 100),
			chunkSize: 16,
			expected:  "",
		},
		{
			name:      "empty input",
			data:      "",
			chunkSize: 16,
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.data)
			result := scanForRustcCommitHashChunked(r, tt.chunkSize)
			if result != tt.expected {
				t.Errorf("scanForRustcCommitHashChunked(chunkSize=%d) = %q; want %q", tt.chunkSize, result, tt.expected)
			}
		})
	}
}
