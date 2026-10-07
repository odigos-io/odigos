package tracesurge

import (
	"context"
	"sync"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

// The result of one evaluation, and of one read of the span metrics.
const (
	resultOK = "ok"
	// the configuration or the Sampling objects could not be read: nothing was evaluated.
	resultError = "error"
	// odigos insights did not answer: no service was evaluated, and open surges hold until stale.
	resultMetricsError = "metrics_error"
	// odigos insights is off: no surge can start, and open ones were restored.
	resultInsightsOff = "insights_off"
)

// What happened to a surge.
const (
	eventStarted    = "started"
	eventLimited    = "limited"
	eventPromoted   = "promoted"
	eventRecovering = "recovering"
	eventRebounded  = "rebounded"
	eventRestored   = "restored"
)

// Why a surge was restored.
type restoreCause string

const (
	causeRecovered    restoreCause = "recovered"
	causeMaxDuration  restoreCause = "max_duration"
	causeStaleMetrics restoreCause = "stale_metrics"
	causeRuleRemoved  restoreCause = "rule_removed"
	causeInsightsOff  restoreCause = "insights_off"
)

var openPhases = []odigosv1.TraceSurgePhase{odigosv1.TraceSurgePhaseLimited, odigosv1.TraceSurgePhaseBoosting, odigosv1.TraceSurgePhaseRecovering}

var evaluationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30}

type surgeCount struct {
	phase odigosv1.TraceSurgePhase
	rule  string
}

// evaluatorMetrics are the evaluator's own metrics, served on the instrumentor's metrics port.
// Only the leader evaluates, so only the leader reports them, and only once trace surge rules or
// surges exist. The gauges show the state after the latest evaluation.
type evaluatorMetrics struct {
	evaluations metric.Float64Histogram
	queries     metric.Float64Histogram
	transitions metric.Int64Counter

	mu sync.Mutex
	// trace surge rules or surges exist: the metrics are reported.
	inUse       bool
	lastSuccess time.Time
	services    int64
	surges      map[surgeCount]int64
	boosted     int64
	limits      limits
}

func newEvaluatorMetrics(meter metric.Meter) (*evaluatorMetrics, error) {
	if meter == nil {
		meter = noop.NewMeterProvider().Meter("")
	}
	m := &evaluatorMetrics{}
	var err error
	if m.evaluations, err = meter.Float64Histogram("odigos.trace_surge.evaluation.duration",
		metric.WithDescription("How long each trace surge evaluation took, by result: ok, metrics_error (odigos insights did not answer), insights_off, or error (nothing was evaluated)."),
		metric.WithUnit("s"), metric.WithExplicitBucketBoundaries(evaluationBuckets...)); err != nil {
		return nil, err
	}
	if m.queries, err = meter.Float64Histogram("odigos.trace_surge.metrics_query.duration",
		metric.WithDescription("How long reading the span metrics from odigos insights took, by result: ok or error."),
		metric.WithUnit("s"), metric.WithExplicitBucketBoundaries(evaluationBuckets...)); err != nil {
		return nil, err
	}
	if m.transitions, err = meter.Int64Counter("odigos.trace_surge.transitions",
		metric.WithDescription("Trace surge transitions by rule and event: started, limited, promoted, recovering, rebounded, or restored (with the reason: recovered, max_duration, stale_metrics, rule_removed, insights_off)."),
		metric.WithUnit("{transition}")); err != nil {
		return nil, err
	}

	lastSuccess, err := meter.Float64ObservableGauge("odigos.trace_surge.evaluation.last_success",
		metric.WithDescription("Unix time of the latest evaluation that read the span metrics, or found odigos insights off."),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	services, err := meter.Int64ObservableGauge("odigos.trace_surge.services",
		metric.WithDescription("Workloads with span metrics in the latest evaluation's window; 0 when it read none."),
		metric.WithUnit("{workload}"))
	if err != nil {
		return nil, err
	}
	surges, err := meter.Int64ObservableGauge("odigos.trace_surge.surges",
		metric.WithDescription("Open trace surges by phase (Limited, Boosting, Recovering) and rule. Every rule with a surge has a series per phase."),
		metric.WithUnit("{surge}"))
	if err != nil {
		return nil, err
	}
	boosted, err := meter.Int64ObservableGauge("odigos.trace_surge.boosted_workloads",
		metric.WithDescription("Workloads whose sampling a trace surge currently raises."),
		metric.WithUnit("{workload}"))
	if err != nil {
		return nil, err
	}
	maxActive, err := meter.Int64ObservableGauge("odigos.trace_surge.limit.max_active_surges",
		metric.WithDescription("The configured sampling.traceSurge.maxActiveSurges."),
		metric.WithUnit("{surge}"))
	if err != nil {
		return nil, err
	}
	maxBoosted, err := meter.Int64ObservableGauge("odigos.trace_surge.limit.max_boosted_workloads",
		metric.WithDescription("The configured sampling.traceSurge.maxBoostedWorkloads."),
		metric.WithUnit("{workload}"))
	if err != nil {
		return nil, err
	}

	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		m.mu.Lock()
		defer m.mu.Unlock()
		if !m.inUse {
			return nil
		}
		if !m.lastSuccess.IsZero() {
			o.ObserveFloat64(lastSuccess, float64(m.lastSuccess.UnixNano())/1e9)
		}
		o.ObserveInt64(services, m.services)
		for c, n := range m.surges {
			o.ObserveInt64(surges, n, metric.WithAttributes(attribute.String("phase", string(c.phase)), attribute.String("rule", c.rule)))
		}
		o.ObserveInt64(boosted, m.boosted)
		o.ObserveInt64(maxActive, int64(m.limits.maxActive))
		o.ObserveInt64(maxBoosted, int64(m.limits.maxBoosted))
		return nil
	}, lastSuccess, services, surges, boosted, maxActive, maxBoosted)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (m *evaluatorMetrics) evaluated(ctx context.Context, result string, took time.Duration, at time.Time) {
	m.mu.Lock()
	inUse := m.inUse
	m.mu.Unlock()
	if !inUse {
		return
	}
	m.evaluations.Record(ctx, took.Seconds(), metric.WithAttributes(attribute.String("result", result)))
	if result == resultOK || result == resultInsightsOff {
		m.mu.Lock()
		m.lastSuccess = at
		m.mu.Unlock()
	}
}

func (m *evaluatorMetrics) queried(ctx context.Context, took time.Duration, err error) {
	result := resultOK
	if err != nil {
		result = resultError
	}
	m.queries.Record(ctx, took.Seconds(), metric.WithAttributes(attribute.String("result", result)))
}

func (m *evaluatorMetrics) transition(ctx context.Context, surge *odigosv1.TraceSurge, event string, cause restoreCause) {
	attrs := []attribute.KeyValue{attribute.String("rule", ruleLabel(surge.Spec.RuleName, surge.Spec.RuleID)), attribute.String("event", event)}
	if cause != "" {
		attrs = append(attrs, attribute.String("reason", string(cause)))
	}
	m.transitions.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// snapshot records the state the evaluation left: the open surges of every rule with a surge, and
// the workloads they raise.
func (m *evaluatorMetrics) snapshot(rules map[[2]string]surgeRule, samplings map[string]*samplingState, services int, l limits) {
	surges := make(map[surgeCount]int64, len(rules)*len(openPhases))
	for _, rule := range rules {
		for _, phase := range openPhases {
			surges[surgeCount{phase, ruleLabel(rule.op.Name, rule.id)}] += 0
		}
	}
	boosted := map[k8sconsts.PodWorkload]bool{}
	for _, state := range samplings {
		for _, surge := range state.surges {
			if surge.Status.Phase == "" || surge.Status.Phase == odigosv1.TraceSurgePhaseRestored {
				continue
			}
			surges[surgeCount{surge.Status.Phase, ruleLabel(surge.Spec.RuleName, surge.Spec.RuleID)}]++
			if surge.Active() {
				for _, t := range surge.Spec.Targets {
					boosted[t.Workload] = true
				}
			}
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inUse = m.inUse || len(rules) > 0 || len(boosted) > 0 || anySurge(samplings)
	m.surges = surges
	m.boosted = int64(len(boosted))
	m.services = int64(services)
	m.limits = l
}

// ruleLabel names a rule in the metrics: by its name, or its id when it has none.
func ruleLabel(name, id string) string {
	if name != "" {
		return name
	}
	return id
}

func anySurge(samplings map[string]*samplingState) bool {
	for _, state := range samplings {
		if len(state.surges) > 0 {
			return true
		}
	}
	return false
}
