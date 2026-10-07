package tracesurge

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObserveTakesTheWorstOperation(t *testing.T) {
	// 4.5% of payments' calls fail, all of them in POST /pay: diluted below a 5% threshold.
	m := &serviceMetrics{namespace: "shop", serviceName: "payments",
		redMetrics: redMetrics{calls: 1000, errors: 45, p95Ms: 20},
		operations: []operationMetrics{
			{name: "GET /browse", redMetrics: redMetrics{calls: 880, p95Ms: 10}},
			{name: "GET /rare", redMetrics: redMetrics{calls: 20, errors: 20, p95Ms: 900}},
			{name: "POST /pay", redMetrics: redMetrics{calls: 100, errors: 45, p95Ms: 300}},
		},
	}

	value, operation, requests, ok := m.observe("error_rate", time.Minute, 100)
	require.True(t, ok)
	assert.Equal(t, 45.0, value)
	assert.Equal(t, "POST /pay", operation)
	assert.Equal(t, int64(100), requests, "the calls of the operation the value is of")

	value, operation, _, ok = m.observe("latency_p95", time.Minute, 100)
	require.True(t, ok)
	assert.Equal(t, 300.0, value, "GET /rare is slower but has fewer calls than the rule requires")
	assert.Equal(t, "POST /pay", operation)

	value, operation, requests, ok = m.observe("error_rate", time.Minute, 200)
	require.True(t, ok)
	assert.Equal(t, 4.5, value, "no operation has the calls the rule requires: the whole service")
	assert.Empty(t, operation)
	assert.Equal(t, int64(1000), requests)

	value, operation, _, ok = m.observe("request_rate", time.Minute, 100)
	require.True(t, ok)
	assert.InDelta(t, 1000.0/60, value, 1e-9, "an operation's rate never exceeds the service's")
	assert.Empty(t, operation)

	_, _, _, ok = m.observe("error_rate", time.Minute, 1001)
	assert.False(t, ok)
}

func TestInsightsMetricsRead(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = w.Write([]byte(`{
			"workloads": [
				{"namespace": "shop", "workload_kind": "Deployment", "workload_name": "payments", "service_name": "payments-svc",
				 "calls": 90, "error_calls": 9, "duration_bounds_ms": [10, 100], "duration_bucket_counts": [80, 10, 0]},
				{"namespace": "shop", "workload_kind": "Deployment", "workload_name": "payments", "service_name": "payments-svc",
				 "calls": 10, "error_calls": 1, "duration_bounds_ms": [5], "duration_bucket_counts": [10, 0]}
			],
			"operations": [
				{"namespace": "shop", "workload_kind": "Deployment", "workload_name": "payments", "span_name": "POST /pay",
				 "calls": 40, "error_calls": 9, "duration_bounds_ms": [10, 100], "duration_bucket_counts": [0, 40, 0]},
				{"namespace": "shop", "workload_kind": "Deployment", "workload_name": "payments", "span_name": "POST /pay",
				 "calls": 2, "error_calls": 1, "duration_bounds_ms": [5], "duration_bucket_counts": [2, 0]},
				{"namespace": "shop", "workload_kind": "Deployment", "workload_name": "gone", "span_name": "GET /",
				 "calls": 5, "error_calls": 0, "duration_bounds_ms": [5], "duration_bucket_counts": [5, 0]}
			],
			"service_calls": [
				{"client_namespace": "shop", "client_service": "frontend", "server_namespace": "shop", "server_service": "payments-svc"}
			]
		}`))
	}))
	defer srv.Close()

	s := &insightsMetrics{baseURL: srv.URL, client: srv.Client()}
	services, callers, err := s.read(context.Background(), time.Minute, 100)
	require.NoError(t, err)
	assert.Equal(t, "min_operation_calls=100&window_seconds=60", query)

	require.Len(t, services, 1)
	for _, m := range services {
		assert.Equal(t, 100.0, m.calls)
		assert.Equal(t, 10.0, m.errors)
		assert.False(t, math.IsNaN(m.p95Ms), "p95 from the bounds with most calls")
		require.Len(t, m.operations, 1, "an operation of a workload without totals is dropped")
		op := m.operations[0]
		assert.Equal(t, "POST /pay", op.name)
		assert.Equal(t, 42.0, op.calls, "the operation's entries of each bounds add up")
		assert.Equal(t, 10.0, op.errors)
		assert.InDelta(t, 95.5, op.p95Ms, 1e-9, "p95 from the [10, 100] entry, which has most calls: 10 + 90*38/40")
	}
	assert.Equal(t, map[string][]string{"shop/payments-svc": {"shop/frontend"}}, callers)
}
