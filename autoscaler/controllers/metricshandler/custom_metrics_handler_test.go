package metricshandler

import "testing"

func TestParseGatewayRejectionMetric(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    float64
		wantErr bool
	}{
		{
			name: "new metric sums all counter series",
			body: `
# TYPE odigos_collector_memory_limiter_batch_rejections_total counter
odigos_collector_memory_limiter_batch_rejections_total{receiver="otlp"} 2
odigos_collector_memory_limiter_batch_rejections_total{receiver="otlp/2"} 3
`,
			want: 5,
		},
		{
			name: "legacy metric remains supported during rolling upgrade",
			body: `
# TYPE odigos_gateway_memory_limiter_rejections_total counter
odigos_gateway_memory_limiter_rejections_total{receiver="otlp"} 4
`,
			want: 4,
		},
		{
			name: "new metric wins if both names are present",
			body: `
# TYPE odigos_collector_memory_limiter_batch_rejections_total counter
odigos_collector_memory_limiter_batch_rejections_total 7
# TYPE odigos_gateway_memory_limiter_rejections_total counter
odigos_gateway_memory_limiter_rejections_total 11
`,
			want: 7,
		},
		{
			name: "missing metric is zero",
			body: `
# TYPE unrelated_total counter
unrelated_total 9
`,
			want: 0,
		},
		{
			name:    "invalid prometheus text returns error",
			body:    "not valid{",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGatewayRejectionMetric([]byte(tt.body))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("parseGatewayRejectionMetric() = %v, want %v", got, tt.want)
			}
		})
	}
}
