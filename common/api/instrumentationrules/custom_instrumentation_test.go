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

func TestPhpCustomProbeString(t *testing.T) {
	t.Parallel()

	if got := (&PhpCustomProbe{ClassName: "Foo", FunctionName: "bar"}).String(); got != "Foo::bar" {
		t.Fatalf("String() = %q, want %q", got, "Foo::bar")
	}
	if got := (&PhpCustomProbe{FunctionName: "bar"}).String(); got != "bar" {
		t.Fatalf("String() = %q, want %q", got, "bar")
	}
}
package instrumentationrules

import "testing"

func Test_CustomInstrumentations_Verify(t *testing.T) {
	tests := []struct {
		name    string
		ci      *CustomInstrumentations
		wantErr bool
	}{
		{
			name:    "nil receiver",
			ci:      nil,
			wantErr: false,
		},
		{
			name:    "empty",
			ci:      &CustomInstrumentations{},
			wantErr: false,
		},
		{
			name: "valid cpp probe",
			ci: &CustomInstrumentations{
				Cpp: []CppCustomProbe{{Signature: "std::vector::push_back"}},
			},
			wantErr: false,
		},
		{
			name: "invalid cpp probe: empty signature",
			ci: &CustomInstrumentations{
				Cpp: []CppCustomProbe{{Signature: ""}},
			},
			wantErr: true,
		},
		{
			name: "valid golang and java, invalid cpp still caught",
			ci: &CustomInstrumentations{
				Golang: []GolangCustomProbe{{PackageName: "net/http", FunctionName: "ListenAndServe"}},
				Java:   []JavaCustomProbe{{ClassName: "com.foo.Bar", MethodName: "baz"}},
				Cpp:    []CppCustomProbe{{Signature: ""}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ci.Verify()
			if (err != nil) != tt.wantErr {
				t.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
