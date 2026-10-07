package tracesurge

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// collected reads the metrics the evaluator of the test recorded so far.
type collected struct {
	t       *testing.T
	metrics map[string]metricdata.Aggregation
}

func collect(t *testing.T) collected {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, readers[t.Name()].Collect(context.Background(), &rm))
	c := collected{t: t, metrics: map[string]metricdata.Aggregation{}}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			c.metrics[m.Name] = m.Data
		}
	}
	return c
}

func matches(set attribute.Set, attrs map[string]string) bool {
	if set.Len() != len(attrs) {
		return false
	}
	for k, v := range attrs {
		if got, ok := set.Value(attribute.Key(k)); !ok || got.AsString() != v {
			return false
		}
	}
	return true
}

// value is the gauge's or counter's value for exactly attrs; ok is false when there is none.
func (c collected) value(name string, attrs map[string]string) (float64, bool) {
	switch data := c.metrics[name].(type) {
	case metricdata.Gauge[int64]:
		for _, dp := range data.DataPoints {
			if matches(dp.Attributes, attrs) {
				return float64(dp.Value), true
			}
		}
	case metricdata.Gauge[float64]:
		for _, dp := range data.DataPoints {
			if matches(dp.Attributes, attrs) {
				return dp.Value, true
			}
		}
	case metricdata.Sum[int64]:
		for _, dp := range data.DataPoints {
			if matches(dp.Attributes, attrs) {
				return float64(dp.Value), true
			}
		}
	}
	return 0, false
}

func (c collected) number(name string, attrs map[string]string) float64 {
	c.t.Helper()
	v, ok := c.value(name, attrs)
	require.True(c.t, ok, "%s%v not recorded", name, attrs)
	return v
}

// count is how many values the histogram recorded for exactly attrs.
func (c collected) count(name string, attrs map[string]string) uint64 {
	if data, ok := c.metrics[name].(metricdata.Histogram[float64]); ok {
		for _, dp := range data.DataPoints {
			if matches(dp.Attributes, attrs) {
				return dp.Count
			}
		}
	}
	return 0
}

func shop(phase odigosv1.TraceSurgePhase) map[string]string {
	return map[string]string{"phase": string(phase), "rule": "shop"}
}

func TestEvaluatorMetricsFollowASurge(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	tick := func(at time.Duration) {
		t.Helper()
		require.NoError(t, e.evaluate(ctx, t0.Add(at)))
	}

	m.paymentsErrorRate(50)
	tick(0)
	got := collect(t)
	for _, phase := range openPhases {
		assert.Equal(t, 0.0, got.number("odigos.trace_surge.surges", shop(phase)), "every rule with a surge has a series per phase")
	}
	assert.Equal(t, float64(len(m.services)), got.number("odigos.trace_surge.services", nil))
	assert.Equal(t, 10.0, got.number("odigos.trace_surge.limit.max_active_surges", nil))
	assert.Equal(t, 50.0, got.number("odigos.trace_surge.limit.max_boosted_workloads", nil))
	assert.Equal(t, float64(t0.Unix()), got.number("odigos.trace_surge.evaluation.last_success", nil))

	tick(30 * time.Second)
	got = collect(t)
	assert.Equal(t, 1.0, got.number("odigos.trace_surge.surges", shop(odigosv1.TraceSurgePhaseBoosting)))
	assert.Equal(t, 2.0, got.number("odigos.trace_surge.boosted_workloads", nil), "frontend and payments")
	assert.Equal(t, 1.0, got.number("odigos.trace_surge.transitions", map[string]string{"rule": "shop", "event": eventStarted}))
	assert.Equal(t, uint64(2), got.count("odigos.trace_surge.evaluation.duration", map[string]string{"result": resultOK}))
	assert.Equal(t, uint64(2), got.count("odigos.trace_surge.metrics_query.duration", map[string]string{"result": resultOK}))

	m.paymentsErrorRate(0)
	tick(40 * time.Second)
	got = collect(t)
	assert.Equal(t, 1.0, got.number("odigos.trace_surge.surges", shop(odigosv1.TraceSurgePhaseRecovering)))
	assert.Equal(t, 0.0, got.number("odigos.trace_surge.surges", shop(odigosv1.TraceSurgePhaseBoosting)))
	assert.Equal(t, 2.0, got.number("odigos.trace_surge.boosted_workloads", nil), "recovering still raises sampling")

	tick(110 * time.Second)
	tick(160 * time.Second)
	got = collect(t)
	require.Empty(t, openSurges(t, c))
	require.Len(t, archives[t.Name()].surges, 1, "restored and moved to insights")
	assert.Equal(t, 0.0, got.number("odigos.trace_surge.surges", shop(odigosv1.TraceSurgePhaseRecovering)))
	assert.Equal(t, 0.0, got.number("odigos.trace_surge.boosted_workloads", nil))
	assert.Equal(t, 1.0, got.number("odigos.trace_surge.transitions", map[string]string{"rule": "shop", "event": eventRecovering}))
	assert.Equal(t, 1.0, got.number("odigos.trace_surge.transitions", map[string]string{"rule": "shop", "event": eventRestored, "reason": string(causeRecovered)}))
	assert.Equal(t, uint64(5), got.count("odigos.trace_surge.evaluation.duration", map[string]string{"result": resultOK}))
}

func TestEvaluatorMetricsWhenInsightsFails(t *testing.T) {
	ctx := context.Background()
	e, _, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	require.NoError(t, e.evaluate(ctx, t0))

	m.err = errors.New("insights unavailable")
	require.NoError(t, e.evaluate(ctx, t0.Add(10*time.Second)))
	got := collect(t)
	assert.Equal(t, uint64(1), got.count("odigos.trace_surge.evaluation.duration", map[string]string{"result": resultMetricsError}))
	assert.Equal(t, uint64(1), got.count("odigos.trace_surge.metrics_query.duration", map[string]string{"result": resultError}))
	assert.Equal(t, float64(t0.Unix()), got.number("odigos.trace_surge.evaluation.last_success", nil),
		"an evaluation that read no metrics is not a success: alert when this stops moving")
	assert.Equal(t, 0.0, got.number("odigos.trace_surge.services", nil))

	m.err = nil
	require.NoError(t, e.evaluate(ctx, t0.Add(20*time.Second)))
	assert.Equal(t, float64(t0.Add(20*time.Second).Unix()), collect(t).number("odigos.trace_surge.evaluation.last_success", nil))
}

func TestEvaluatorMetricsWhenInsightsIsOff(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))

	require.NoError(t, c.Update(ctx, effectiveConfig(false)))
	require.NoError(t, e.evaluate(ctx, t0.Add(40*time.Second)))
	got := collect(t)
	assert.Equal(t, uint64(1), got.count("odigos.trace_surge.evaluation.duration", map[string]string{"result": resultInsightsOff}))
	assert.Equal(t, 1.0, got.number("odigos.trace_surge.transitions", map[string]string{"rule": "shop", "event": eventRestored, "reason": string(causeInsightsOff)}))
	assert.Equal(t, 0.0, got.number("odigos.trace_surge.boosted_workloads", nil))
}

func TestEvaluatorMetricsOfLimitedSurges(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	withLimits(t, c, "    maxBoostedWorkloads: 1\n")
	t0 := time.Unix(1_800_000_000, 0)
	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	got := collect(t)
	assert.Equal(t, 1.0, got.number("odigos.trace_surge.surges", shop(odigosv1.TraceSurgePhaseLimited)))
	assert.Equal(t, 0.0, got.number("odigos.trace_surge.boosted_workloads", nil), "a limited surge raises nothing")
	assert.Equal(t, 1.0, got.number("odigos.trace_surge.limit.max_boosted_workloads", nil))
	assert.Equal(t, 1.0, got.number("odigos.trace_surge.transitions", map[string]string{"rule": "shop", "event": eventLimited}))

	withLimits(t, c, "    maxBoostedWorkloads: 5\n")
	require.NoError(t, e.evaluate(ctx, t0.Add(40*time.Second)))
	got = collect(t)
	assert.Equal(t, 1.0, got.number("odigos.trace_surge.transitions", map[string]string{"rule": "shop", "event": eventPromoted}))
	assert.Equal(t, 1.0, got.number("odigos.trace_surge.surges", shop(odigosv1.TraceSurgePhaseBoosting)))
}

// The names a Prometheus scrape of the instrumentor shows, through the same exporter the
// instrumentor registers on its metrics endpoint.
func TestEvaluatorMetricNamesInPrometheus(t *testing.T) {
	registry := prometheus.NewRegistry()
	exporter, err := otelprometheus.New(otelprometheus.WithRegisterer(registry), otelprometheus.WithoutTargetInfo())
	require.NoError(t, err)
	m, err := newEvaluatorMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter)).Meter("test"))
	require.NoError(t, err)
	ctx := context.Background()
	families, err := registry.Gather()
	require.NoError(t, err)
	assert.Empty(t, families, "no trace surge rule: nothing is reported, not even target_info")

	m.snapshot(map[[2]string]surgeRule{{"default", "abc"}: {sampling: "default", id: "abc", op: odigosv1.NoisyOperation{}}}, nil, 3, limits{maxActive: 10, maxBoosted: 50})
	m.evaluated(ctx, resultOK, 30*time.Millisecond, time.Unix(1_800_000_000, 0))
	m.queried(ctx, 10*time.Millisecond, nil)
	m.transition(ctx, &odigosv1.TraceSurge{Spec: odigosv1.TraceSurgeSpec{RuleID: "abc"}}, eventStarted, "")

	families, err = registry.Gather()
	require.NoError(t, err)
	var names []string
	for _, f := range families {
		names = append(names, f.GetName())
	}
	sort.Strings(names)
	for _, want := range []string{
		"odigos_trace_surge_boosted_workloads",
		"odigos_trace_surge_evaluation_duration_seconds",
		"odigos_trace_surge_evaluation_last_success_seconds",
		"odigos_trace_surge_limit_max_active_surges",
		"odigos_trace_surge_limit_max_boosted_workloads",
		"odigos_trace_surge_metrics_query_duration_seconds",
		"odigos_trace_surge_services",
		"odigos_trace_surge_surges",
		"odigos_trace_surge_transitions_total",
	} {
		assert.Contains(t, names, want)
	}
}

func TestEvaluatorReportsNothingWithoutTraceSurgeRules(t *testing.T) {
	ctx := context.Background()
	e, c, _, _ := setup(t)
	sampling := &odigosv1.Sampling{}
	require.NoError(t, c.Get(ctx, client.ObjectKey{Namespace: ns, Name: "default"}, sampling))
	sampling.Spec.NoisyOperations[0].Surge = nil
	require.NoError(t, c.Update(ctx, sampling))

	for i := range 3 {
		require.NoError(t, e.evaluate(ctx, time.Unix(1_800_000_000+int64(i)*10, 0)))
	}
	var rm metricdata.ResourceMetrics
	require.NoError(t, readers[t.Name()].Collect(ctx, &rm))
	for _, sm := range rm.ScopeMetrics {
		assert.Empty(t, sm.Metrics, "a cluster that uses no trace surge sees no trace surge metrics")
	}
}
