package insights

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GetAnomaly backfills the anomaly/baseline comparison traces the detail
// payload may omit. The fallbacks decide what the compare view shows, and each
// of them fails softly, so a broken one is invisible unless it is asserted.
func TestGetAnomalyCompareTraceFallbacks(t *testing.T) {
	const anomalyDetail = `{
		"transaction_id": 9,
		"signature": "sig",
		"status": "open",
		"last_trace_id": "trace-last"
	}`

	tests := []struct {
		name              string
		routes            map[string]string
		wantAnomalyTrace  string
		wantBaselineTrace string
	}{
		{
			name: "falls back to last_trace_id when no evidence sample is retained",
			routes: map[string]string{
				"/api/v1/transactions/9/observations":            `{"items":[]}`,
				"/api/v1/transactions/9/observations/trace-last": `{"trace_id":"trace-last"}`,
			},
			wantAnomalyTrace: "trace-last",
		},
		{
			name: "gives up when the last trace is gone",
			routes: map[string]string{
				"/api/v1/transactions/9/observations": `{"items":[]}`,
			},
		},
		{
			name: "gives up when the evidence sample cannot be fetched",
			routes: map[string]string{
				"/api/v1/transactions/9/observations": `{"items":[{"trace_id":"trace-evidence"}]}`,
			},
		},
		{
			name: "prefers the evidence sample over last_trace_id",
			routes: map[string]string{
				"/api/v1/transactions/9/observations":                `{"items":[{"trace_id":"trace-evidence"}]}`,
				"/api/v1/transactions/9/observations/trace-evidence": `{"trace_id":"trace-evidence"}`,
				"/api/v1/transactions/9/observations/trace-last":     `{"trace_id":"trace-last"}`,
			},
			wantAnomalyTrace:  "trace-evidence",
			wantBaselineTrace: "trace-evidence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/anomalies/9/sig") {
					_, _ = w.Write([]byte(anomalyDetail))
					return
				}
				body, ok := tt.routes[r.URL.Path]
				if !ok {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()

			client, err := NewClient(server.URL)
			require.NoError(t, err)

			// The enrichment must never turn a readable anomaly into an error.
			got, err := client.GetAnomaly(context.Background(), 9, "sig")
			require.NoError(t, err)

			if tt.wantAnomalyTrace == "" {
				assert.Nil(t, got.AnomalyTrace)
			} else {
				require.NotNil(t, got.AnomalyTrace)
				assert.Equal(t, tt.wantAnomalyTrace, got.AnomalyTrace.TraceID)
			}
			if tt.wantBaselineTrace == "" {
				assert.Nil(t, got.BaselineTrace)
			} else {
				require.NotNil(t, got.BaselineTrace)
				assert.Equal(t, tt.wantBaselineTrace, got.BaselineTrace.TraceID)
			}
		})
	}
}

// A detail payload that already carries both samples must not trigger extra
// requests to the observations API.
func TestGetAnomalyKeepsEmbeddedCompareTraces(t *testing.T) {
	var observationRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/observations") {
			observationRequests++
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{
			"transaction_id": 9,
			"signature": "sig",
			"anomaly_trace": {"trace_id":"embedded-anomaly"},
			"baseline_trace": {"trace_id":"embedded-baseline"}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	got, err := client.GetAnomaly(context.Background(), 9, "sig")
	require.NoError(t, err)
	require.NotNil(t, got.AnomalyTrace)
	assert.Equal(t, "embedded-anomaly", got.AnomalyTrace.TraceID)
	require.NotNil(t, got.BaselineTrace)
	assert.Equal(t, "embedded-baseline", got.BaselineTrace.TraceID)
	assert.Zero(t, observationRequests)
}

func TestEnrichAnomalyCompareTracesToleratesAMissingIssue(t *testing.T) {
	client, err := NewClient("http://insights:8080")
	require.NoError(t, err)
	assert.NotPanics(t, func() { client.enrichAnomalyCompareTraces(context.Background(), nil) })
}

// The evidence sample is looked up by the anomaly:<signature> storage tag, so a
// sample belonging to a different signature must not be shown as the evidence.
func TestGetAnomalyLooksUpEvidenceBySignature(t *testing.T) {
	var sampleReasons []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/transactions/9/observations" {
			sampleReasons = append(sampleReasons, r.URL.Query().Get("sample_reason"))
			_, _ = w.Write([]byte(`{"items":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"transaction_id":9,"signature":"D2_egress|db:5432"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	_, err = client.GetAnomaly(context.Background(), 9, "D2_egress|db:5432")
	require.NoError(t, err)
	assert.Equal(t, []string{"anomaly:D2_egress|db:5432", "example"}, sampleReasons)
}
