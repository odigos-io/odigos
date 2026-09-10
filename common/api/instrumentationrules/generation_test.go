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
