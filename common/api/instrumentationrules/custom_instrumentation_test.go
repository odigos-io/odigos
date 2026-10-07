package instrumentationrules

import (
	"testing"
)

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

func TestCustomInstrumentationsVerifyGolangSignatures(t *testing.T) {
	t.Parallel()

	probe := func(sig string) GolangCustomProbe {
		return GolangCustomProbe{PackageName: "main", FunctionName: "computeScore", Signature: sig}
	}

	ci := &CustomInstrumentations{Golang: []GolangCustomProbe{probe("func(n int32) uint64"), probe("func(n int32) uint64")}}
	if err := ci.Verify(); err != nil {
		t.Fatalf("Verify() error = %v, want nil", err)
	}

	ci = &CustomInstrumentations{Golang: []GolangCustomProbe{probe("func(n int32) uint64"), probe("func(n int64) uint64")}}
	err := ci.Verify()
	want := `invalid configuration for golang custom instrumentation: {main computeScore   } declares signatures "func(n int32) uint64" and "func(n int64) uint64"`
	if err == nil || err.Error() != want {
		t.Fatalf("Verify() error = %v, want %s", err, want)
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
