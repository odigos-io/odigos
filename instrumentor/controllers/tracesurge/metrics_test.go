package tracesurge

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
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
				 "calls": 10, "error_calls": 1, "duration_bounds_ms": [1000], "duration_bucket_counts": [10, 0]},
				{"namespace": "shop", "workload_kind": "Deployment", "workload_name": "orders", "service_name": "orders",
				 "calls": 500, "error_calls": 0, "duration_bounds_ms": [10], "duration_bucket_counts": [500, 0], "incomplete": true}
			],
			"operations": [
				{"namespace": "shop", "workload_kind": "Deployment", "workload_name": "payments", "span_name": "POST /pay",
				 "calls": 40, "error_calls": 9, "duration_bounds_ms": [10, 100], "duration_bucket_counts": [0, 40, 0]},
				{"namespace": "shop", "workload_kind": "Deployment", "workload_name": "payments", "span_name": "POST /pay",
				 "calls": 2, "error_calls": 1, "duration_bounds_ms": [5], "duration_bucket_counts": [2, 0]},
				{"namespace": "shop", "workload_kind": "Deployment", "workload_name": "payments", "span_name": "GET /receipt",
				 "calls": 200, "error_calls": 200, "duration_bounds_ms": [5], "duration_bucket_counts": [200, 0], "incomplete": true},
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

	require.Len(t, services, 2)
	m := services[k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "payments"}]
	require.NotNil(t, m)
	assert.Equal(t, 100.0, m.calls)
	assert.Equal(t, 10.0, m.errors)
	assert.False(t, m.incomplete)
	assert.InDelta(t, 950.0, m.p95Ms, 1e-9, "the highest p95 of the bounds: the [1000] entry's 1000*9.5/10, though it has fewer calls")
	require.Len(t, m.operations, 2, "an operation of a workload without totals is dropped")
	op := m.operations[1]
	assert.Equal(t, "POST /pay", op.name)
	assert.Equal(t, 42.0, op.calls, "the operation's entries of each bounds add up")
	assert.Equal(t, 10.0, op.errors)
	assert.InDelta(t, 95.5, op.p95Ms, 1e-9, "the highest p95 of the bounds: the [10, 100] entry's 10 + 90*38/40")

	value, operation, _, ok := m.observe("error_rate", time.Minute, 2)
	require.True(t, ok)
	assert.Equal(t, "POST /pay", operation, "GET /receipt is incomplete: it is not evaluated on its own")
	assert.InDelta(t, 100*10.0/42, value, 1e-9)

	orders := services[k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "orders"}]
	require.NotNil(t, orders, "an incomplete workload stays in the service graph")
	assert.True(t, orders.incomplete)
	_, _, _, ok = orders.observe("request_rate", time.Minute, 1)
	assert.False(t, ok, "insights dropped some of its series: no observation, neither a breach nor a recovery")

	assert.Equal(t, map[string][]string{"shop/payments-svc": {"shop/frontend"}}, callers)
}

func TestInsightsMetricsReadWhileInsightsHasTraceSurgesOff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	s := &insightsMetrics{baseURL: srv.URL, client: srv.Client()}
	_, _, err := s.read(context.Background(), time.Minute, 100)
	assert.True(t, errors.Is(err, errInsightsTraceSurgesOff), "got %v", err)
}

func TestRequestRateIsOverTheWindowInsightsSummed(t *testing.T) {
	var query string
	windowSeconds := `"window_seconds": 30,`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = w.Write([]byte(`{` + windowSeconds + ` "workloads": [
			{"namespace": "shop", "workload_kind": "Deployment", "workload_name": "payments", "service_name": "payments",
			 "calls": 600, "error_calls": 0, "duration_bounds_ms": [10], "duration_bucket_counts": [600, 0]}
		]}`))
	}))
	defer srv.Close()
	s := &insightsMetrics{baseURL: srv.URL, client: srv.Client()}
	payments := k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "payments"}

	services, _, err := s.read(context.Background(), 25*time.Second, 1)
	require.NoError(t, err)
	assert.Contains(t, query, "window_seconds=25")
	assert.Equal(t, 30*time.Second, services[payments].windowOr(25*time.Second))
	value, _, _, ok := services[payments].observe("request_rate", 25*time.Second, 1)
	require.True(t, ok)
	assert.Equal(t, 20.0, value, "600 calls over the 30s insights summed: whole 10s slots")

	windowSeconds = ""
	services, _, err = s.read(context.Background(), 25*time.Second, 1)
	require.NoError(t, err)
	value, _, _, _ = services[payments].observe("request_rate", 25*time.Second, 1)
	assert.Equal(t, 24.0, value, "without window_seconds, the window asked for")
}
