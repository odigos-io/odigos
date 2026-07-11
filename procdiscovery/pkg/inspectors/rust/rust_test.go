package rust

import (
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
