package tracesurge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

// serviceMetrics are the RED metrics of a workload's server spans over the evaluation window,
// and of its operations that had enough calls to be evaluated on their own.
type serviceMetrics struct {
	namespace   string
	serviceName string
	// the window the metrics were summed over: the window asked for, rounded up to whole slots of
	// odigos insights. 0 when not known.
	window time.Duration
	redMetrics
	operations []operationMetrics
}

type redMetrics struct {
	calls  float64
	errors float64
	p95Ms  float64
	// odigos insights dropped some of the series in the window: the values are not to be trusted.
	incomplete bool
}

// operationMetrics are the RED metrics of the server spans of one name, e.g. "GET /checkout/{id}".
type operationMetrics struct {
	name string
	redMetrics
}

// serviceKey identifies a service in the service graph.
func serviceKey(namespace, service string) string {
	return namespace + "/" + service
}

func (m serviceMetrics) key() string {
	return serviceKey(m.namespace, m.serviceName)
}

func (m redMetrics) value(metric string, window time.Duration) (float64, bool) {
	switch metric {
	case "error_rate":
		if m.calls == 0 {
			return 0, false
		}
		return 100 * m.errors / m.calls, true
	case "latency_p95":
		return m.p95Ms, !math.IsNaN(m.p95Ms)
	case "request_rate":
		return m.calls / window.Seconds(), true
	}
	return 0, false
}

// windowOr returns the window the metrics were summed over, or requested when not known.
func (m *serviceMetrics) windowOr(requested time.Duration) time.Duration {
	if m.window > 0 {
		return m.window
	}
	return requested
}

// observe returns the metric's value over the window the metrics were summed over (window when not
// known), or !ok under minimumRequests calls or when
// the metrics are incomplete. Error rate and p95 take the highest of the service and of each
// operation with minimumRequests calls, so a spike in one operation is not diluted; operation names
// it, or is empty for the service.
func (m *serviceMetrics) observe(metric string, window time.Duration, minimumRequests int) (value float64, operation string, requests int64, ok bool) {
	if m.incomplete || m.calls < float64(minimumRequests) {
		return 0, "", 0, false
	}
	window = m.windowOr(window)
	if value, ok = m.value(metric, window); !ok {
		return 0, "", 0, false
	}
	requests = int64(m.calls)
	if metric != "error_rate" && metric != "latency_p95" {
		return value, "", requests, true
	}
	for _, op := range m.operations {
		if op.incomplete || op.calls < float64(minimumRequests) {
			continue
		}
		if v, vok := op.value(metric, window); vok && v > value {
			value, operation, requests = v, op.name, int64(op.calls)
		}
	}
	return value, operation, requests, true
}

// insightsMetrics reads the agents' span metrics and the service graph from odigos-insights.
type insightsMetrics struct {
	baseURL string
	client  *http.Client
}

func newInsightsMetrics(odigosNamespace string) *insightsMetrics {
	return &insightsMetrics{
		baseURL: k8sconsts.InsightsHTTPEndpoint(odigosNamespace),
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

type spanMetricsHistogram struct {
	Calls                int64     `json:"calls"`
	ErrorCalls           int64     `json:"error_calls"`
	DurationBoundsMs     []float64 `json:"duration_bounds_ms"`
	DurationBucketCounts []int64   `json:"duration_bucket_counts"`
	// insights' series budget dropped some of the series in the window.
	Incomplete bool `json:"incomplete"`
}

// errInsightsTraceSurgesOff is returned while odigos insights has trace surges turned off: it
// stores no span metrics then.
var errInsightsTraceSurgesOff = errors.New("odigos insights has trace surges turned off")

// spanMetricsWindow is the response of insights' GET /api/v1/span-metrics.
type spanMetricsWindow struct {
	// the window summed: window_seconds rounded up to whole 10s slots.
	WindowSeconds int `json:"window_seconds"`
	Workloads     []struct {
		Namespace    string `json:"namespace"`
		WorkloadKind string `json:"workload_kind"`
		WorkloadName string `json:"workload_name"`
		ServiceName  string `json:"service_name"`
		spanMetricsHistogram
	} `json:"workloads"`
	Operations []struct {
		Namespace    string `json:"namespace"`
		WorkloadKind string `json:"workload_kind"`
		WorkloadName string `json:"workload_name"`
		SpanName     string `json:"span_name"`
		spanMetricsHistogram
	} `json:"operations"`
	ServiceCalls []struct {
		ClientNamespace string `json:"client_namespace"`
		ClientService   string `json:"client_service"`
		ServerNamespace string `json:"server_namespace"`
		ServerService   string `json:"server_service"`
	} `json:"service_calls"`
}

// read returns the RED metrics of every workload and of its operations with at least
// minOperationCalls calls, and the callers of each service (by serviceKey).
func (s *insightsMetrics) read(ctx context.Context, window time.Duration, minOperationCalls int) (map[k8sconsts.PodWorkload]*serviceMetrics, map[string][]string, error) {
	u := s.baseURL + "/api/v1/span-metrics?" + url.Values{
		"window_seconds":      {strconv.Itoa(int(window.Seconds()))},
		"min_operation_calls": {strconv.Itoa(minOperationCalls)},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("read span metrics from insights: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, nil, errInsightsTraceSurgesOff
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("read span metrics from insights: %s", resp.Status)
	}
	var body spanMetricsWindow
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, nil, fmt.Errorf("decode span metrics from insights: %w", err)
	}

	services := map[k8sconsts.PodWorkload]*serviceMetrics{}
	// a workload or operation has an entry per histogram bounds, e.g. while its agents roll out with
	// new ones. its p95 is the highest of theirs, so a slow tail in one of them is not hidden.
	add := func(m *redMetrics, h *spanMetricsHistogram) {
		m.calls += float64(h.Calls)
		m.errors += float64(h.ErrorCalls)
		m.incomplete = m.incomplete || h.Incomplete
		if h.Calls == 0 {
			return
		}
		if p95 := histogramQuantile(0.95, h.DurationBoundsMs, h.DurationBucketCounts); math.IsNaN(m.p95Ms) || p95 > m.p95Ms {
			m.p95Ms = p95
		}
	}
	for i := range body.Workloads {
		w := &body.Workloads[i]
		pw := k8sconsts.PodWorkload{Namespace: w.Namespace, Kind: k8sconsts.WorkloadKind(w.WorkloadKind), Name: w.WorkloadName}
		m := services[pw]
		if m == nil {
			m = &serviceMetrics{namespace: w.Namespace, serviceName: w.ServiceName, window: time.Duration(body.WindowSeconds) * time.Second,
				redMetrics: redMetrics{p95Ms: math.NaN()}}
			services[pw] = m
		}
		add(&m.redMetrics, &w.spanMetricsHistogram)
	}
	operations := map[k8sconsts.PodWorkload]map[string]*redMetrics{}
	for i := range body.Operations {
		o := &body.Operations[i]
		pw := k8sconsts.PodWorkload{Namespace: o.Namespace, Kind: k8sconsts.WorkloadKind(o.WorkloadKind), Name: o.WorkloadName}
		if services[pw] == nil {
			continue
		}
		if operations[pw] == nil {
			operations[pw] = map[string]*redMetrics{}
		}
		m := operations[pw][o.SpanName]
		if m == nil {
			m = &redMetrics{p95Ms: math.NaN()}
			operations[pw][o.SpanName] = m
		}
		add(m, &o.spanMetricsHistogram)
	}
	for pw, ops := range operations {
		m := services[pw]
		m.operations = make([]operationMetrics, 0, len(ops))
		for name, op := range ops {
			m.operations = append(m.operations, operationMetrics{name: name, redMetrics: *op})
		}
		slices.SortFunc(m.operations, func(a, b operationMetrics) int { return strings.Compare(a.name, b.name) })
	}
	callers := map[string][]string{}
	for _, c := range body.ServiceCalls {
		server := serviceKey(c.ServerNamespace, c.ServerService)
		callers[server] = append(callers[server], serviceKey(c.ClientNamespace, c.ClientService))
	}
	return services, callers, nil
}

// histogramQuantile estimates the q quantile of an explicit-bucket histogram by linear
// interpolation within the bucket that holds it. counts has one more entry than bounds, for the
// values above the last bound, where the quantile is taken to be the last bound.
func histogramQuantile(q float64, bounds []float64, counts []int64) float64 {
	if len(counts) != len(bounds)+1 || len(bounds) == 0 {
		return math.NaN()
	}
	var total int64
	for _, c := range counts {
		total += c
	}
	if total == 0 {
		return math.NaN()
	}
	rank := q * float64(total)
	var seen int64
	for i, c := range counts {
		if c == 0 || float64(seen+c) < rank {
			seen += c
			continue
		}
		if i == len(bounds) {
			return bounds[len(bounds)-1]
		}
		lower := 0.0
		if i > 0 {
			lower = bounds[i-1]
		}
		return lower + (bounds[i]-lower)*(rank-float64(seen))/float64(c)
	}
	return bounds[len(bounds)-1]
}
