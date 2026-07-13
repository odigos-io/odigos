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
