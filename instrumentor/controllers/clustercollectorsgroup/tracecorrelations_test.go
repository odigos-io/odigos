package clustercollectorsgroup

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

func TestGetTraceCorrelationsSettings(t *testing.T) {
	enabled := true
	disabled := false

	tests := []struct {
		name string
		cfg  common.OdigosConfiguration
		want *odigosv1.CollectorsGroupTraceCorrelationsSettings
	}{
		{
			name: "nil when trace correlations unset",
			cfg:  common.OdigosConfiguration{},
			want: nil,
		},
		{
			name: "nil when serviceIO disabled",
			cfg: common.OdigosConfiguration{
				TraceCorrelations: &common.TraceCorrelationsConfiguration{
					ServiceIO: &common.TraceCorrelationsServiceIOConfiguration{
						Enabled: &disabled,
					},
				},
			},
			want: nil,
		},
		{
			name: "populated when serviceIO enabled",
			cfg: common.OdigosConfiguration{
				TraceCorrelations: &common.TraceCorrelationsConfiguration{
					ServiceIO: &common.TraceCorrelationsServiceIOConfiguration{
						Enabled: &enabled,
						InputSpanAttributes: []string{
							"http.route",
						},
						OutputSpanAttributes: []string{
							"db.system",
						},
					},
				},
			},
			want: &odigosv1.CollectorsGroupTraceCorrelationsSettings{
				ServiceIO: &common.TraceCorrelationsServiceIOConfiguration{
					Enabled: &enabled,
					InputSpanAttributes: []string{
						"http.route",
					},
					OutputSpanAttributes: []string{
						"db.system",
					},
					MetricsFlushInterval: "60s",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTraceCorrelationsSettings(&tt.cfg)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("getTraceCorrelationsSettings() = %#v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("getTraceCorrelationsSettings() = nil, want non-nil")
			}
			if got.ServiceIO == nil || tt.want.ServiceIO == nil {
				t.Fatalf("unexpected nil serviceIO: got=%#v want=%#v", got.ServiceIO, tt.want.ServiceIO)
			}
			if *got.ServiceIO.Enabled != *tt.want.ServiceIO.Enabled {
				t.Errorf("enabled = %v, want %v", *got.ServiceIO.Enabled, *tt.want.ServiceIO.Enabled)
			}
			if got.ServiceIO.MetricsFlushInterval != tt.want.ServiceIO.MetricsFlushInterval {
				t.Errorf("metricsFlushInterval = %q, want %q", got.ServiceIO.MetricsFlushInterval, tt.want.ServiceIO.MetricsFlushInterval)
			}
			if len(got.ServiceIO.InputSpanAttributes) != len(tt.want.ServiceIO.InputSpanAttributes) {
				t.Fatalf("inputSpanAttributes = %#v, want %#v", got.ServiceIO.InputSpanAttributes, tt.want.ServiceIO.InputSpanAttributes)
			}
			if len(got.ServiceIO.OutputSpanAttributes) != len(tt.want.ServiceIO.OutputSpanAttributes) {
				t.Fatalf("outputSpanAttributes = %#v, want %#v", got.ServiceIO.OutputSpanAttributes, tt.want.ServiceIO.OutputSpanAttributes)
			}
		})
	}
}
