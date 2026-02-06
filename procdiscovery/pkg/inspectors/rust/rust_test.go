package rust

import (
	"testing"
)

func TestIsRustSymbol(t *testing.T) {
	tests := []struct {
		name     string
		symbol   string
		expected bool
	}{
		{
			name:     "rust alloc symbol",
			symbol:   "__rust_alloc",
			expected: true,
		},
		{
			name:     "rust dealloc symbol",
			symbol:   "__rust_dealloc",
			expected: true,
		},
		{
			name:     "rust begin unwind",
			symbol:   "rust_begin_unwind",
			expected: true,
		},
		{
			name:     "rust eh personality",
			symbol:   "rust_eh_personality",
			expected: true,
		},
		{
			name:     "mangled core symbol",
			symbol:   "_ZN4core3ptr13drop_in_place17h1234567890abcdefE",
			expected: true,
		},
		{
			name:     "mangled std symbol",
			symbol:   "_ZN3std2io5stdio6_print17h1234567890abcdefE",
			expected: true,
		},
		{
			name:     "mangled alloc symbol",
			symbol:   "_ZN5alloc3vec16Vec$LT$T$GT$4push17h1234567890abcdefE",
			expected: true,
		},
		{
			name:     "c++ symbol",
			symbol:   "_ZN3foo3barEv",
			expected: false,
		},
		{
			name:     "random symbol",
			symbol:   "main",
			expected: false,
		},
		{
			name:     "libc symbol",
			symbol:   "__libc_start_main",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRustSymbol(tt.symbol)
			if result != tt.expected {
				t.Errorf("isRustSymbol(%q) = %v, want %v", tt.symbol, result, tt.expected)
			}
		})
	}
}
