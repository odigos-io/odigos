package tracesurge

import (
	"context"
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	instance "github.com/odigos-io/odigos/k8sutils/pkg/instrumentation_instance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const ns = "odigos-system"

var (
	frontend = k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "frontend"}
	payments = k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "payments"}
	other    = k8sconsts.PodWorkload{Namespace: "other", Kind: k8sconsts.WorkloadKindDeployment, Name: "other"}
)

type fakeMetrics struct {
	services          map[k8sconsts.PodWorkload]*serviceMetrics
	callers           map[string][]string
	minOperationCalls int
	window            time.Duration
	err               error
}

// paymentsErrorRate sets the percent of payments' 1000 calls in the window that fail.
func (f *fakeMetrics) paymentsErrorRate(percent float64) {
	f.services[payments] = &serviceMetrics{namespace: "shop", serviceName: "payments", redMetrics: redMetrics{calls: 1000, errors: 10 * percent, p95Ms: math.NaN()}}
}

type metricsAdapter struct{ f *fakeMetrics }

func (a metricsAdapter) read(_ context.Context, window time.Duration, minOperationCalls int) (map[k8sconsts.PodWorkload]*serviceMetrics, map[string][]string, error) {
	a.f.minOperationCalls = minOperationCalls
	a.f.window = window
	if a.f.err != nil {
		return nil, nil, a.f.err
	}
	return a.f.services, a.f.callers, nil
}

func ptr[T any](v T) *T { return &v }

func effectiveConfig(insights bool, more ...string) *corev1.ConfigMap {
	yaml := "insights:\n  enabled: " + strconv.FormatBool(insights) + "\n"
	for _, m := range more {
		yaml += m
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: ns},
		Data:       map[string]string{consts.OdigosConfigurationFileName: yaml},
	}
}

func withLimits(t *testing.T, c client.Client, limits string) {
	t.Helper()
	require.NoError(t, c.Update(context.Background(), effectiveConfig(true, "sampling:\n  traceSurge:\n"+limits)))
}

func ic(pw k8sconsts.PodWorkload) *odigosv1.InstrumentationConfig {
	return &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "deployment-" + pw.Name, Namespace: pw.Namespace},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{{ContainerName: "app", Language: common.JavaProgrammingLanguage}},
		},
	}
}

func instrumentationInstance(pw k8sconsts.PodWorkload, pod string, applied string) *odigosv1.InstrumentationInstance {
	return &odigosv1.InstrumentationInstance{
		ObjectMeta: metav1.ObjectMeta{Name: pod + "-1", Namespace: pw.Namespace, Labels: map[string]string{
			consts.InstrumentedAppNameLabel: "deployment-" + pw.Name,
			odigosv1.OwnerPodNameLabel:      pod,
		}},
		Status: odigosv1.InstrumentationInstanceStatus{
			NonIdentifyingAttributes: []odigosv1.Attribute{{Key: instance.HeadSamplingAppliedAttribute, Value: applied}},
		},
	}
}

func setup(t *testing.T) (*Evaluator, client.Client, *fakeMetrics, string) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, odigosv1.AddToScheme(scheme))
	rule := odigosv1.NoisyOperation{
		Name:             "shop",
		SourceScopes:     &k8sconsts.SourcesScopes{Namespaces: []string{"shop"}},
		PercentageAtMost: ptr(1.0),
		Surge: &odigosv1.TraceSurgeSettings{
			Version: "1", Metric: odigosv1.TraceSurgeMetricErrorRate, Threshold: 5, SustainedSeconds: 30, MinimumRequests: 100,
			BoostPercent: 80, RecoveryThreshold: 2, RecoverySeconds: 60, MinimumBoostSeconds: 120,
		},
	}
	sampling := &odigosv1.Sampling{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: ns},
		Spec:       odigosv1.SamplingSpec{NoisyOperations: []odigosv1.NoisyOperation{rule}},
	}
	ruleID := odigosv1.ComputeNoisyOperationHash(&rule)
	require.NoError(t, corev1.AddToScheme(scheme))
	writes := new(int)
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&odigosv1.Sampling{}, &odigosv1.InstrumentationInstance{}).
		WithObjects(sampling, ic(frontend), ic(payments), ic(other), effectiveConfig(true)).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourcePatch: func(ctx context.Context, c client.Client, subResource string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
				if _, ok := obj.(*odigosv1.Sampling); ok {
					*writes++
				}
				return c.SubResource(subResource).Patch(ctx, obj, patch, opts...)
			},
		}).
		Build()
	statusWrites[t.Name()] = writes
	m := &fakeMetrics{
		services: map[k8sconsts.PodWorkload]*serviceMetrics{
			frontend: {namespace: "shop", serviceName: "frontend", redMetrics: redMetrics{calls: 2000, p95Ms: 12}},
			other:    {namespace: "other", serviceName: "other", redMetrics: redMetrics{calls: 2000, errors: 1000, p95Ms: 12}},
		},
		callers: map[string][]string{"shop/payments": {"shop/frontend"}},
	}
	reader := sdkmetric.NewManualReader()
	readers[t.Name()] = reader
	e := &Evaluator{Client: c, APIReader: c, Logger: logr.Discard(), Meter: sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)).Meter("test")}
	archive := &fakeArchive{surges: map[string]odigosv1.TraceSurge{}}
	archives[t.Name()] = archive
	e.init(ns, metricsAdapter{m}, archive)
	return e, c, m, ruleID
}

// fakeArchive keeps the surges the evaluator moves out of the Sampling status.
type fakeArchive struct {
	surges map[string]odigosv1.TraceSurge
	err    error
	puts   int
}

func (a *fakeArchive) put(_ context.Context, surge *odigosv1.TraceSurge) error {
	a.puts++
	if a.err != nil {
		return a.err
	}
	a.surges[surge.Name] = *surge.DeepCopy()
	return nil
}

var (
	// by test name: the fake client with interceptors cannot be a map key.
	archives     = map[string]*fakeArchive{}
	statusWrites = map[string]*int{}
	readers      = map[string]*sdkmetric.ManualReader{}
)

// openSurges returns the surges in the status of the Sampling.
func openSurges(t *testing.T, c client.Client) []odigosv1.TraceSurge {
	t.Helper()
	sampling := &odigosv1.Sampling{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "default"}, sampling))
	return sampling.Status.TraceSurges
}

// surges returns every surge recorded, in the Sampling status or moved to the archive, oldest first.
func surges(t *testing.T, c client.Client) []odigosv1.TraceSurge {
	t.Helper()
	all := openSurges(t, c)
	for name, surge := range archives[t.Name()].surges {
		if !slices.ContainsFunc(all, func(s odigosv1.TraceSurge) bool { return s.Name == name }) {
			all = append(all, surge)
		}
	}
	slices.SortFunc(all, func(a, b odigosv1.TraceSurge) int {
		if c := a.StartedAt.Compare(b.StartedAt.Time); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
	})
	return all
}

func TestSurgeLifecycle(t *testing.T) {
	ctx := context.Background()
	e, c, m, ruleID := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	tick := func(at time.Duration) {
		t.Helper()
		require.NoError(t, e.evaluate(ctx, t0.Add(at)))
	}

	// payments errors at 50%: a breach, but not sustained for 30s yet.
	m.paymentsErrorRate(50)
	tick(0)
	tick(20 * time.Second)
	assert.Empty(t, surges(t, c), "the breach has not lasted sustainedSeconds")

	// "other" is outside the rule's scope and nothing in scope calls it: never evaluated.
	tick(30 * time.Second)
	all := surges(t, c)
	require.Len(t, all, 1)
	surge := all[0]
	assert.Equal(t, odigosv1.TraceSurgePhaseBoosting, surge.Status.Phase)
	assert.Equal(t, payments, surge.Spec.Service)
	assert.Equal(t, ruleID, surge.Spec.RuleID)
	assert.Equal(t, 1.0, surge.Spec.NormalPercent)
	assert.Equal(t, []odigosv1.TraceSurgeTarget{{Workload: frontend}, {Workload: payments}}, surge.Spec.Targets,
		"the service itself and the services in scope that call it")
	require.NotNil(t, surge.Status.Trigger)
	assert.Equal(t, 50.0, surge.Status.Trigger.Value)
	assert.Len(t, surge.Status.TriggerSamples, 3)
	assert.Equal(t, 0, surge.Status.Targets[0].Total, "no processes reported yet")

	// the processes of frontend report the boost; payments' process still the normal percentage.
	require.NoError(t, c.Create(ctx, instrumentationInstance(frontend, "frontend-a", ruleID+"=80")))
	require.NoError(t, c.Create(ctx, instrumentationInstance(frontend, "frontend-b", ruleID+"=80")))
	require.NoError(t, c.Create(ctx, instrumentationInstance(payments, "payments-a", ruleID+"=1")))
	tick(40 * time.Second)
	surge = surges(t, c)[0]
	assert.Equal(t, 2, surge.Status.Targets[0].Confirmed)
	assert.Equal(t, 0, surge.Status.Targets[1].Confirmed)
	assert.Equal(t, 1, surge.Status.Targets[1].Total)

	// the metric recovers, but the boost holds for minimumBoostSeconds (120s since boosting at 30s).
	m.paymentsErrorRate(1)
	tick(50 * time.Second)
	assert.Equal(t, odigosv1.TraceSurgePhaseRecovering, surges(t, c)[0].Status.Phase)
	// a fresh breach restarts recovery.
	m.paymentsErrorRate(9)
	tick(60 * time.Second)
	assert.Equal(t, odigosv1.TraceSurgePhaseBoosting, surges(t, c)[0].Status.Phase)
	m.paymentsErrorRate(0)
	tick(70 * time.Second)
	tick(131 * time.Second)
	assert.Equal(t, odigosv1.TraceSurgePhaseRecovering, surges(t, c)[0].Status.Phase,
		"recovered for 61s >= recoverySeconds, but boosted for 101s < minimumBoostSeconds")
	tick(151 * time.Second)
	surge = surges(t, c)[0]
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, surge.Status.Phase)
	assert.NotEmpty(t, surge.Status.RestoreReason)

	// after restoring, the targets are confirmed against the normal percentage.
	assert.Equal(t, 0, surge.Status.Targets[0].Confirmed, "frontend still reports 80%")
	assert.Equal(t, 1, surge.Status.Targets[1].Confirmed, "payments reports 1%")

	// a restored surge no longer counts: a new breach starts a new surge.
	m.paymentsErrorRate(50)
	tick(160 * time.Second)
	tick(190 * time.Second)
	assert.Len(t, surges(t, c), 2)
}

func TestOperationSpikeIsNotDiluted(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)

	// 4.5% of payments' calls fail, under the 5% threshold, but they are 45% of POST /pay.
	m.services[payments] = &serviceMetrics{namespace: "shop", serviceName: "payments",
		redMetrics: redMetrics{calls: 1000, errors: 45, p95Ms: math.NaN()},
		operations: []operationMetrics{{name: "POST /pay", redMetrics: redMetrics{calls: 100, errors: 45, p95Ms: math.NaN()}}},
	}
	require.NoError(t, e.evaluate(ctx, t0))
	assert.Equal(t, 100, m.minOperationCalls, "operations with fewer calls than the rule requires are not read")
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))

	all := surges(t, c)
	require.Len(t, all, 1)
	trigger := all[0].Status.Trigger
	require.NotNil(t, trigger)
	assert.Equal(t, 45.0, trigger.Value)
	assert.Equal(t, "POST /pay", trigger.Operation)
	assert.Equal(t, int64(100), trigger.Requests)
	assert.Equal(t, "shop/payments POST /pay: 45% over 100 calls in the last minute.", all[0].Status.Timeline[0].Description)
}

func TestZeroMaxActiveSurgesTurnsSurgesOff(t *testing.T) {
	ctx := context.Background()
	e, c, m, ruleID := setup(t)
	withLimits(t, c, "    maxActiveSurges: 0\n")
	t0 := time.Unix(1_800_000_000, 0)

	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	all := surges(t, c)
	require.Len(t, all, 1)
	assert.Equal(t, odigosv1.TraceSurgePhaseLimited, all[0].Status.Phase)
	assert.Equal(t, "Trace surges are turned off: sampling.traceSurge.maxActiveSurges is 0.", all[0].Status.LimitReason)
	assert.Nil(t, all[0].Status.BoostedAt, "nothing is raised")
	assert.False(t, all[0].Active())
	assert.Equal(t, ruleID, all[0].Spec.RuleID)
}

func TestConfiguredEvaluationWindow(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	withLimits(t, c, "    evaluationWindow: 20s\n    evaluationInterval: 5s\n")
	sampling := &odigosv1.Sampling{}
	require.NoError(t, c.Get(ctx, client.ObjectKey{Namespace: ns, Name: "default"}, sampling))
	sampling.Spec.NoisyOperations[0].Surge.SustainedSeconds = 20
	require.NoError(t, c.Update(ctx, sampling))
	t0 := time.Unix(1_800_000_000, 0)

	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0))
	assert.Equal(t, 20*time.Second, m.window, "the metrics are read over the configured window")
	assert.Equal(t, 5*time.Second, e.interval)
	require.NoError(t, e.evaluate(ctx, t0.Add(15*time.Second)))
	assert.Empty(t, surges(t, c), "the breach has not lasted the rule's 20s")
	require.NoError(t, e.evaluate(ctx, t0.Add(20*time.Second)))
	all := surges(t, c)
	require.Len(t, all, 1)
	assert.Equal(t, "shop/payments: 50% over 1000 calls in the last 20s.", all[0].Status.Timeline[0].Description)
}

func TestMinimumRequests(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)

	m.services[payments] = &serviceMetrics{namespace: "shop", serviceName: "payments", redMetrics: redMetrics{calls: 50, errors: 50, p95Ms: math.NaN()}}
	for at := time.Duration(0); at <= time.Minute; at += 10 * time.Second {
		require.NoError(t, e.evaluate(ctx, t0.Add(at)))
	}
	assert.Empty(t, surges(t, c), "50 calls is below minimumRequests")
}

func TestEndedSurgeConfirmsWhatTheWorkloadsShouldApplyNow(t *testing.T) {
	ctx := context.Background()
	e, c, m, ruleID := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	tick := func(at time.Duration) {
		t.Helper()
		require.NoError(t, e.evaluate(ctx, t0.Add(at)))
	}
	byService := func(service string) odigosv1.TraceSurge {
		t.Helper()
		for _, s := range surges(t, c) {
			if s.Spec.Service.Name == service {
				return s
			}
		}
		t.Fatalf("no surge for %s", service)
		return odigosv1.TraceSurge{}
	}

	// payments, then frontend, spike: two surges of the same rule.
	m.paymentsErrorRate(50)
	m.services[frontend] = &serviceMetrics{namespace: "shop", serviceName: "frontend", redMetrics: redMetrics{calls: 2000, errors: 1000, p95Ms: 12}}
	tick(0)
	tick(30 * time.Second)
	require.Len(t, surges(t, c), 2)
	assert.Equal(t, []odigosv1.TraceSurgeTarget{{Workload: frontend}}, byService("frontend").Spec.Targets, "nothing in scope calls frontend")

	require.NoError(t, c.Create(ctx, instrumentationInstance(frontend, "frontend-a", ruleID+"=80")))
	require.NoError(t, c.Create(ctx, instrumentationInstance(payments, "payments-a", ruleID+"=80")))

	// payments recovers and its surge ends while frontend's continues.
	m.paymentsErrorRate(0)
	tick(40 * time.Second)
	tick(160 * time.Second)
	payments := byService("payments")
	require.Equal(t, odigosv1.TraceSurgePhaseRestored, payments.Status.Phase)
	require.Equal(t, odigosv1.TraceSurgePhaseBoosting, byService("frontend").Status.Phase)

	// frontend stays at 80% for the active surge, payments goes back to 1%.
	ii := &odigosv1.InstrumentationInstance{}
	require.NoError(t, c.Get(ctx, client.ObjectKey{Namespace: "shop", Name: "payments-a-1"}, ii))
	ii.Status.NonIdentifyingAttributes = []odigosv1.Attribute{{Key: instance.HeadSamplingAppliedAttribute, Value: ruleID + "=1"}}
	require.NoError(t, c.Status().Update(ctx, ii))
	tick(170 * time.Second)
	payments = byService("payments")
	for _, target := range payments.Status.Targets {
		assert.Equal(t, target.Total, target.Confirmed, "%s applies what it should now", target.Workload.Name)
	}
	last := payments.Status.Timeline[len(payments.Status.Timeline)-1]
	assert.Equal(t, "All 2 processes apply the rule's current percentage", last.Title)

	// once confirmed, the ended surge's record is final.
	require.NoError(t, c.Get(ctx, client.ObjectKey{Namespace: "shop", Name: "frontend-a-1"}, ii))
	ii.Status.NonIdentifyingAttributes = []odigosv1.Attribute{{Key: instance.HeadSamplingAppliedAttribute, Value: ruleID + "=1"}}
	require.NoError(t, c.Status().Update(ctx, ii))
	tick(180 * time.Second)
	assert.Equal(t, payments.Status.Targets, byService("payments").Status.Targets)
}

func TestSurgesEndWhenInsightsIsTurnedOff(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	require.Len(t, surges(t, c), 1)

	require.NoError(t, c.Update(ctx, effectiveConfig(false)))
	require.NoError(t, e.evaluate(ctx, t0.Add(40*time.Second)))
	surge := surges(t, c)[0]
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, surge.Status.Phase)
	assert.Contains(t, surge.Status.RestoreReason, "Insights")

	// no new surge starts without insights.
	require.NoError(t, e.evaluate(ctx, t0.Add(80*time.Second)))
	require.NoError(t, e.evaluate(ctx, t0.Add(120*time.Second)))
	assert.Len(t, surges(t, c), 1)
}

func TestHistogramQuantile(t *testing.T) {
	bounds := []float64{10, 100, 1000}
	assert.InDelta(t, 55.0, histogramQuantile(0.95, bounds, []int64{90, 10, 0, 0}), 1e-9, "halfway into the (10, 100] bucket")
	assert.InDelta(t, 9.5, histogramQuantile(0.95, bounds, []int64{100, 0, 0, 0}), 1e-9)
	assert.Equal(t, 1000.0, histogramQuantile(0.95, bounds, []int64{1, 0, 0, 99}), "above the last bound")
	assert.True(t, math.IsNaN(histogramQuantile(0.95, bounds, []int64{0, 0, 0, 0})))
	assert.True(t, math.IsNaN(histogramQuantile(0.95, bounds, []int64{1, 2})))
}

func TestMaxDurationEndsSurgeAndWaitsForRecovery(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	withLimits(t, c, "    maxDuration: 2m\n")
	t0 := time.Unix(1_800_000_000, 0)
	tick := func(at time.Duration) {
		t.Helper()
		require.NoError(t, e.evaluate(ctx, t0.Add(at)))
	}

	m.paymentsErrorRate(50)
	tick(0)
	tick(30 * time.Second)
	require.Len(t, surges(t, c), 1)
	tick(149 * time.Second)
	assert.Equal(t, odigosv1.TraceSurgePhaseBoosting, surges(t, c)[0].Status.Phase, "boosted for 119s")
	tick(150 * time.Second)
	surge := surges(t, c)[0]
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, surge.Status.Phase, "boosted for the maximum duration, still spiking")
	assert.Contains(t, surge.Status.RestoreReason, "maximum surge duration of 2m0s")

	// still spiking: no new surge until the service recovers.
	for at := 160 * time.Second; at <= 400*time.Second; at += 10 * time.Second {
		tick(at)
	}
	assert.Len(t, surges(t, c), 1)

	// a restarted evaluator keeps waiting for the recovery too.
	restarted := &Evaluator{Client: c, APIReader: c, Logger: e.Logger}
	restarted.init(ns, e.metrics, e.archive)
	require.NoError(t, restarted.evaluate(ctx, t0.Add(410*time.Second)))
	require.NoError(t, restarted.evaluate(ctx, t0.Add(450*time.Second)))
	assert.Len(t, surges(t, c), 1)

	m.paymentsErrorRate(1)
	require.NoError(t, restarted.evaluate(ctx, t0.Add(460*time.Second)))
	surge = surges(t, c)[0]
	require.NotNil(t, surge.Status.RecoveredAt, "the recovery is recorded on the surge that ended")
	assert.Equal(t, "Error rate back at 1%", surge.Status.Timeline[len(surge.Status.Timeline)-1].Title)

	// an evaluator restarted after the recovery does not hold the service back again.
	again := &Evaluator{Client: c, APIReader: c, Logger: e.Logger}
	again.init(ns, e.metrics, e.archive)
	m.paymentsErrorRate(50)
	require.NoError(t, again.evaluate(ctx, t0.Add(470*time.Second)))
	require.NoError(t, again.evaluate(ctx, t0.Add(500*time.Second)))
	assert.Len(t, surges(t, c), 2, "after recovering, a new spike starts a new surge")
}

func TestMaxDurationWhileRecoveringDoesNotHoldTheServiceBack(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	withLimits(t, c, "    maxDuration: 2m\n")
	t0 := time.Unix(1_800_000_000, 0)
	tick := func(at time.Duration) {
		t.Helper()
		require.NoError(t, e.evaluate(ctx, t0.Add(at)))
	}

	m.paymentsErrorRate(50)
	tick(0)
	tick(30 * time.Second)
	// recovered at 120s: the 60s recovery window would end at 180s, after the maximum duration.
	m.paymentsErrorRate(0)
	tick(120 * time.Second)
	assert.Equal(t, odigosv1.TraceSurgePhaseRecovering, surges(t, c)[0].Status.Phase)
	tick(150 * time.Second)
	surge := surges(t, c)[0]
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, surge.Status.Phase)
	assert.Equal(t, "Error rate was at or below 2% when the maximum surge duration of 2m0s was reached.", surge.Status.RestoreReason)

	// not held back: a new spike starts a new surge.
	m.paymentsErrorRate(50)
	tick(160 * time.Second)
	tick(190 * time.Second)
	assert.Len(t, surges(t, c), 2)
}

func TestLimitsKeepSurgesFromRaisingSampling(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	withLimits(t, c, "    maxActiveSurges: 1\n")
	t0 := time.Unix(1_800_000_000, 0)
	tick := func(at time.Duration) {
		t.Helper()
		require.NoError(t, e.evaluate(ctx, t0.Add(at)))
	}
	phases := func() map[string]odigosv1.TraceSurgePhase {
		out := map[string]odigosv1.TraceSurgePhase{}
		for _, s := range surges(t, c) {
			out[s.Spec.Service.Name] = s.Status.Phase
		}
		return out
	}

	// frontend spikes first and is raised. payments spikes after it, and the limit keeps it from raising.
	m.services[frontend] = &serviceMetrics{namespace: "shop", serviceName: "frontend", redMetrics: redMetrics{calls: 2000, errors: 1000, p95Ms: 12}}
	tick(0)
	m.paymentsErrorRate(50)
	tick(10 * time.Second)
	tick(30 * time.Second)
	tick(40 * time.Second)
	assert.Equal(t, map[string]odigosv1.TraceSurgePhase{"frontend": odigosv1.TraceSurgePhaseBoosting, "payments": odigosv1.TraceSurgePhaseLimited}, phases())
	for _, s := range surges(t, c) {
		if s.Spec.Service == payments {
			assert.Nil(t, s.Status.BoostedAt)
			assert.Equal(t, "1 trace surge already raising sampling, the limit (sampling.traceSurge.maxActiveSurges).", s.Status.LimitReason)
			assert.Equal(t, "Sampling not raised", s.Status.Timeline[len(s.Status.Timeline)-1].Title)
		}
	}
	// a limited surge raises nothing.
	boosted := raised(surges(t, c), payments)
	assert.False(t, boosted, "payments is not raised")

	// once frontend's surge ends, payments - still spiking - is raised.
	m.services[frontend] = &serviceMetrics{namespace: "shop", serviceName: "frontend", redMetrics: redMetrics{calls: 2000, errors: 0, p95Ms: 12}}
	tick(50 * time.Second)
	tick(170 * time.Second)
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, phases()["frontend"])
	tick(180 * time.Second)
	assert.Equal(t, odigosv1.TraceSurgePhaseBoosting, phases()["payments"])
}

func TestMaxBoostedWorkloads(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	withLimits(t, c, "    maxBoostedWorkloads: 1\n")
	t0 := time.Unix(1_800_000_000, 0)
	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	surge := surges(t, c)[0]
	assert.Equal(t, odigosv1.TraceSurgePhaseLimited, surge.Status.Phase, "payments' surge would raise frontend and payments")
	assert.Contains(t, surge.Status.LimitReason, "maxBoostedWorkloads")

	// the spike ends while limited: the surge ends without having raised anything.
	m.paymentsErrorRate(0)
	require.NoError(t, e.evaluate(ctx, t0.Add(40*time.Second)))
	surge = surges(t, c)[0]
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, surge.Status.Phase)
	assert.Nil(t, surge.Status.BoostedAt)
	assert.Equal(t, "Surge ended", surge.Status.Timeline[len(surge.Status.Timeline)-1].Title)
	assert.Empty(t, surge.Status.Targets, "nothing to confirm")
}

// raised reports whether any surge raises the workload's sampling.
func raised(surges []odigosv1.TraceSurge, pw k8sconsts.PodWorkload) bool {
	for _, s := range surges {
		if !s.Active() {
			continue
		}
		for _, t := range s.Spec.Targets {
			if t.Workload == pw {
				return true
			}
		}
	}
	return false
}

func TestEndedSurgesMoveToInsights(t *testing.T) {
	ctx := context.Background()
	e, c, m, ruleID := setup(t)
	archive := archives[t.Name()]
	t0 := time.Unix(1_800_000_000, 0)
	tick := func(at time.Duration) {
		t.Helper()
		require.NoError(t, e.evaluate(ctx, t0.Add(at)))
	}

	m.paymentsErrorRate(50)
	tick(0)
	tick(30 * time.Second)
	require.Len(t, openSurges(t, c), 1)
	name := openSurges(t, c)[0].Name

	m.paymentsErrorRate(0)
	tick(40 * time.Second)
	tick(100 * time.Second)
	tick(150 * time.Second)
	require.Equal(t, odigosv1.TraceSurgePhaseRestored, openSurges(t, c)[0].Status.Phase, "ended, waiting for its targets to be confirmed back")
	assert.Empty(t, archive.surges)

	// insights is down: the surge waits in the status.
	archive.err = errors.New("insights unavailable")
	require.NoError(t, c.Create(ctx, instrumentationInstance(frontend, "frontend-a", ruleID+"=1")))
	require.NoError(t, c.Create(ctx, instrumentationInstance(payments, "payments-a", ruleID+"=1")))
	tick(160 * time.Second)
	assert.Len(t, openSurges(t, c), 1)
	assert.Equal(t, 1, archive.puts)

	archive.err = nil
	tick(170 * time.Second)
	assert.Len(t, openSurges(t, c), 1, "no put until the retry time, so insights being down doesn't slow each evaluation")
	assert.Equal(t, 1, archive.puts)
	tick(190 * time.Second)
	assert.Empty(t, openSurges(t, c), "confirmed back: it moved to insights")
	require.Contains(t, archive.surges, name)
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, archive.surges[name].Status.Phase)
	assert.Equal(t, "All 2 processes back at 1%", archive.surges[name].Status.Timeline[len(archive.surges[name].Status.Timeline)-1].Title)
}

func TestMaxDurationSurgeStaysUntilItsServiceRecovers(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	withLimits(t, c, "    maxDuration: 2m\n")
	t0 := time.Unix(1_800_000_000, 0)
	tick := func(at time.Duration) {
		t.Helper()
		require.NoError(t, e.evaluate(ctx, t0.Add(at)))
	}

	m.paymentsErrorRate(50)
	tick(0)
	tick(30 * time.Second)
	tick(150 * time.Second)
	require.Equal(t, odigosv1.TraceSurgePhaseRestored, openSurges(t, c)[0].Status.Phase)
	tick(30 * time.Minute)
	assert.Len(t, openSurges(t, c), 1, "it holds the service back, long after its targets could be confirmed")
	assert.Empty(t, archives[t.Name()].surges)

	m.paymentsErrorRate(1)
	tick(31 * time.Minute)
	assert.Empty(t, openSurges(t, c), "the service recovered: it moved to insights")
	assert.Len(t, archives[t.Name()].surges, 1)
}

func TestStatusWrittenOncePerEvaluation(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)

	m.paymentsErrorRate(50)
	m.services[frontend] = &serviceMetrics{namespace: "shop", serviceName: "frontend", redMetrics: redMetrics{calls: 2000, errors: 1000, p95Ms: math.NaN()}}
	require.NoError(t, e.evaluate(ctx, t0))
	assert.Equal(t, 0, *statusWrites[t.Name()], "nothing changed in the status")
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	assert.Len(t, openSurges(t, c), 2, "a surge for each service")
	assert.Equal(t, 1, *statusWrites[t.Name()], "both surges, in one write")
}
