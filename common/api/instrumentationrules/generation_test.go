package instrumentationrules

import (
	"strings"
	"testing"
)

func TestGolangGenerationValidation(t *testing.T) {
	for _, value := range []string{"", "0123456789abcdef0123456789abcdef", "0", strings.Repeat("0", 32), strings.Repeat("a", 33), strings.Repeat("g", 32), strings.Repeat("A", 32)} {
		p := GolangCustomProbe{PackageName: "main", FunctionName: "work", Generation: value}
		valid := value == "" || value == "0123456789abcdef0123456789abcdef"
		if err := p.Verify(); (err == nil) != valid {
			t.Errorf("generation %q: valid=%v, error=%v", value, valid, err)
		}
	}
}

func TestGolangExpirationValidation(t *testing.T) {
	for _, tc := range []struct {
		generation, expiry string
		valid              bool
	}{
		{"", "", true},
		{"", "2026-09-10T18:30:00Z", false},
		{"0123456789abcdef0123456789abcdef", "2026-09-10T18:30:00Z", true},
		{"0123456789abcdef0123456789abcdef", "yesterday", false},
		{"0123456789abcdef0123456789abcdef", "0001-01-01T00:00:00Z", false},
	} {
		p := GolangCustomProbe{PackageName: "main", FunctionName: "work", Generation: tc.generation, ExpiresAt: tc.expiry}
		if err := p.Verify(); (err == nil) != tc.valid {
			t.Errorf("expiry %q: valid=%v, error=%v", tc.expiry, tc.valid, err)
		}
	}
}
