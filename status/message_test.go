package status

import (
	"testing"
)

func TestRenderMessage(t *testing.T) {
	t.Parallel()

	type params struct {
		WorkloadKind string
	}

	tests := []struct {
		name    string
		message string
		params  params
		want    string
		wantErr bool
	}{
		{
			name:    "no template",
			message: "All pods have the instrumentation agent applied.",
			params:  params{WorkloadKind: "Deployment"},
			want:    "All pods have the instrumentation agent applied.",
		},
		{
			name:    "workload kind",
			message: "Automatic {{ .WorkloadKind }} rollout should be triggered soon",
			params:  params{WorkloadKind: "Deployment"},
			want:    "Automatic Deployment rollout should be triggered soon",
		},
		{
			name:    "missing key",
			message: "Waiting for {{ .Missing }}",
			params:  params{WorkloadKind: "Deployment"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reason := WithMessageTemplate(Reason{Name: tt.name, Message: tt.message})
			got, err := RenderMessage(reason, tt.params)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWithMessageTemplatePanicsOnInvalidTemplate(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = WithMessageTemplate(Reason{Name: "Bad", Message: "broken {{ .Foo"})
}
