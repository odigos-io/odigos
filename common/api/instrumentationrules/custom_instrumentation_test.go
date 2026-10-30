package instrumentationrules

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCustomProbeCategoryVerify(t *testing.T) {
	t.Parallel()

	if err := CustomProbeCategory("").Verify(); err != nil {
		t.Fatalf("empty category: %v", err)
	}
	if err := CustomProbeCategoryObservability.Verify(); err != nil {
		t.Fatalf("observability: %v", err)
	}
	if err := CustomProbeCategory("billing").Verify(); err == nil {
		t.Fatal("expected error for unknown category")
	}
}

func TestToConfigDropsMetadata(t *testing.T) {
	t.Parallel()

	rule := &CustomInstrumentationsWithMetadata{
		Java: []JavaCustomProbeWithMetadata{{
			JavaCustomProbe: JavaCustomProbe{ClassName: "com.acme.Pricing", MethodName: "calculate"},
			CustomProbeMetadata: CustomProbeMetadata{
				Notes:    "called from checkout",
				Category: CustomProbeCategoryObservability,
				AddedAt:  "2026-10-19T09:00:00Z",
			},
		}},
	}
	if err := rule.Verify(); err != nil {
		t.Fatalf("Verify() = %v", err)
	}

	raw, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("marshal rule: %v", err)
	}
	if !strings.Contains(string(raw), `"notes"`) || !strings.Contains(string(raw), `"className"`) {
		t.Fatalf("rule json = %s", raw)
	}

	cfg := rule.ToConfig()
	raw, err = json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if strings.Contains(string(raw), "notes") || strings.Contains(string(raw), "observability") || strings.Contains(string(raw), "addedAt") {
		t.Fatalf("config json includes metadata: %s", raw)
	}
	if cfg.Java[0].ClassName != "com.acme.Pricing" || cfg.Java[0].MethodName != "calculate" {
		t.Fatalf("probe = %+v", cfg.Java[0])
	}
}

func TestPhpCustomProbeVerify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		probe   PhpCustomProbe
		wantErr bool
	}{
		{
			name:    "class and function",
			probe:   PhpCustomProbe{ClassName: `App\Service\OrderService`, FunctionName: "processOrder"},
			wantErr: false,
		},
		{
			name:    "global function",
			probe:   PhpCustomProbe{FunctionName: "my_global_function"},
			wantErr: false,
		},
		{
			name:    "missing function name",
			probe:   PhpCustomProbe{ClassName: `App\Service\OrderService`},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.probe.Verify()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCustomInstrumentationsVerifyPhp(t *testing.T) {
	t.Parallel()

	ci := &CustomInstrumentations{
		Php: []PhpCustomProbe{
			{ClassName: "Foo", FunctionName: "bar"},
			{FunctionName: ""},
		},
	}
	if err := ci.Verify(); err == nil {
		t.Fatal("expected error for invalid php probe")
	}
}

func TestPhpCustomProbeString(t *testing.T) {
	t.Parallel()

	if got := (&PhpCustomProbe{ClassName: "Foo", FunctionName: "bar"}).String(); got != "Foo::bar" {
		t.Fatalf("String() = %q, want %q", got, "Foo::bar")
	}
	if got := (&PhpCustomProbe{FunctionName: "bar"}).String(); got != "bar" {
		t.Fatalf("String() = %q, want %q", got, "bar")
	}
}

func TestPythonCustomProbeVerify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		probe   PythonCustomProbe
		wantErr bool
	}{
		{
			name:    "method",
			probe:   PythonCustomProbe{Module: "app.billing", Qualname: "Invoice.calculate"},
			wantErr: false,
		},
		{
			name:    "with expiry",
			probe:   PythonCustomProbe{Module: "__main__", Qualname: "handler", ExpiresAt: "2026-10-30T12:00:00Z"},
			wantErr: false,
		},
		{
			name:    "missing module",
			probe:   PythonCustomProbe{Qualname: "handler"},
			wantErr: true,
		},
		{
			name:    "missing qualname",
			probe:   PythonCustomProbe{Module: "app.billing"},
			wantErr: true,
		},
		{
			name:    "bad expiry",
			probe:   PythonCustomProbe{Module: "app.billing", Qualname: "handler", ExpiresAt: "tomorrow"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.probe.Verify()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPythonToConfigDropsMetadata(t *testing.T) {
	t.Parallel()

	rule := &CustomInstrumentationsWithMetadata{
		Python: []PythonCustomProbeWithMetadata{{
			PythonCustomProbe:   PythonCustomProbe{Module: "app.billing", Qualname: "Invoice.calculate", ExpiresAt: "2026-10-30T12:00:00Z"},
			CustomProbeMetadata: CustomProbeMetadata{Notes: "slow checkout", AddedAt: "2026-10-30T11:00:00Z"},
		}},
	}
	if err := rule.Verify(); err != nil {
		t.Fatalf("Verify() = %v", err)
	}
	cfg := rule.ToConfig()
	if len(cfg.Python) != 1 || cfg.Python[0] != rule.Python[0].PythonCustomProbe {
		t.Fatalf("python probes = %+v", cfg.Python)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if strings.Contains(string(raw), "notes") || strings.Contains(string(raw), "addedAt") {
		t.Fatalf("config json includes metadata: %s", raw)
	}
	if !strings.Contains(string(raw), `"python":[{"module":"app.billing","qualname":"Invoice.calculate","expiresAt":"2026-10-30T12:00:00Z"}]`) {
		t.Fatalf("config json = %s", raw)
	}
}
