package tracesurge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
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

const (
	ns = "odigos-system"
	// the object size etcd accepts.
	etcdMaxBytes = 1536 << 10
)

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
	reads             int
	err               error
}

// paymentsErrorRate sets the percent of payments' 1000 calls in the window that fail.
func (f *fakeMetrics) paymentsErrorRate(percent float64) {
	f.services[payments] = &serviceMetrics{namespace: "shop", serviceName: "payments", redMetrics: redMetrics{calls: 1000, errors: 10 * percent, p95Ms: math.NaN()}}
}

type metricsAdapter struct{ f *fakeMetrics }

func (a metricsAdapter) read(_ context.Context, window time.Duration, minOperationCalls int) (map[k8sconsts.PodWorkload]*serviceMetrics, map[string][]string, error) {
	a.f.reads++
	a.f.minOperationCalls = minOperationCalls
	a.f.window = window
	if a.f.err != nil {
		return nil, nil, a.f.err
	}
	return a.f.services, a.f.callers, nil
}

func ptr[T any](v T) *T { return &v }

// effectiveConfig turns trace surges on, with the given lines of sampling.traceSurge.
func effectiveConfig(insights bool, traceSurge ...string) *corev1.ConfigMap {
	yaml := "insights:\n  enabled: " + strconv.FormatBool(insights) + "\nsampling:\n  traceSurge:\n    enabled: true\n"
	for _, line := range traceSurge {
		yaml += line
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: ns},
		Data:       map[string]string{consts.OdigosConfigurationFileName: yaml},
	}
}

func withLimits(t *testing.T, c client.Client, limits string) {
	t.Helper()
	require.NoError(t, c.Update(context.Background(), effectiveConfig(true, limits)))
}

func ic(pw k8sconsts.PodWorkload) *odigosv1.InstrumentationConfig {
	return &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "deployment-" + pw.Name, Namespace: pw.Namespace},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{{ContainerName: "app", Language: common.JavaProgrammingLanguage}},
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
		WithStatusSubresource(&odigosv1.Sampling{}).
		WithObjects(sampling, ic(frontend), ic(payments), ic(other), effectiveConfig(true)).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourcePatch: func(ctx context.Context, c client.Client, subResource string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
				if _, ok := obj.(*odigosv1.Sampling); ok {
					*writes++
				}
				if data, err := patch.Data(obj); err == nil {
					if len(data) > etcdMaxBytes {
						return errors.New("etcdserver: request is too large")
					}
					if check := patchChecks[t.Name()]; check != nil {
						if err := check(data); err != nil {
							return err
						}
					}
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
	e := &Evaluator{Client: c, Logger: logr.Discard(), Meter: sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)).Meter("test")}
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
	// how long each put takes.
	delay time.Duration
	// called on each put, before it.
	onPut func(surge *odigosv1.TraceSurge)
}

func (a *fakeArchive) put(ctx context.Context, surge *odigosv1.TraceSurge) error {
	a.puts++
	if a.onPut != nil {
		a.onPut(surge)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(a.delay):
	}
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
	// fails the status patches it returns an error for.
	patchChecks = map[string]func(patch []byte) error{}
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
	assert.Equal(t, []odigosv1.TraceSurgeTarget{{Workload: payments}, {Workload: frontend}}, surge.Spec.Targets,
		"the service itself, then the services in scope that call it")
	require.NotNil(t, surge.Status.Trigger)
	assert.Equal(t, 50.0, surge.Status.Trigger.Value)
	assert.Len(t, surge.Status.TriggerSamples, 3)

	tick(40 * time.Second)

	// the metric recovers, but the boost holds for minimumBoostSeconds (120s since boosting at 30s).
	m.paymentsErrorRate(1)
	tick(50 * time.Second)
	assert.Equal(t, odigosv1.TraceSurgePhaseRecovering, surges(t, c)[0].Status.Phase)
	// a fresh breach restarts recovery.
	m.paymentsErrorRate(9)
	tick(60 * time.Second)
	assert.Equal(t, odigosv1.TraceSurgePhaseBoosting, surges(t, c)[0].Status.Phase)
	m.paymentsErrorRate(0)
	for at := 70 * time.Second; at <= 130*time.Second; at += 10 * time.Second {
		tick(at)
	}
	tick(131 * time.Second)
	assert.Equal(t, odigosv1.TraceSurgePhaseRecovering, surges(t, c)[0].Status.Phase,
		"recovered for 61s >= recoverySeconds, but boosted for 101s < minimumBoostSeconds")
	tick(151 * time.Second)
	surge = surges(t, c)[0]
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, surge.Status.Phase)
	assert.NotEmpty(t, surge.Status.RestoreReason)

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

func TestObservationsAreOverTheWindowInsightsSummed(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	withLimits(t, c, "    evaluationWindow: 25s\n")
	t0 := time.Unix(1_800_000_000, 0)
	m.paymentsErrorRate(50)
	m.services[payments].window = 30 * time.Second
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	all := surges(t, c)
	require.Len(t, all, 1)
	assert.Equal(t, 25*time.Second, m.window, "the window asked for")
	assert.Equal(t, 30, all[0].Status.Trigger.WindowSeconds, "whole 10s slots of odigos insights")
	assert.Equal(t, "shop/payments: 50% over 1000 calls in the last 30s.", all[0].Status.Timeline[0].Description)
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

func TestSurgesEndWhenTraceSurgesAreTurnedOff(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	require.Len(t, surges(t, c), 1)

	off := effectiveConfig(true)
	off.Data[consts.OdigosConfigurationFileName] = "insights:\n  enabled: true\n"
	require.NoError(t, c.Update(ctx, off))
	reads := m.reads
	require.NoError(t, e.evaluate(ctx, t0.Add(40*time.Second)))
	surge := surges(t, c)[0]
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, surge.Status.Phase)
	assert.Equal(t, "Trace surges were turned off.", surge.Status.RestoreReason)

	require.NoError(t, e.evaluate(ctx, t0.Add(80*time.Second)))
	assert.Len(t, surges(t, c), 1, "no surge starts while trace surges are off")
	assert.Equal(t, reads, m.reads, "nothing is read from odigos insights while trace surges are off")
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
	restarted := &Evaluator{Client: c, Logger: e.Logger}
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
	again := &Evaluator{Client: c, Logger: e.Logger}
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
	for at := 50 * time.Second; at <= 170*time.Second; at += 10 * time.Second {
		tick(at)
	}
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, phases()["frontend"])
	tick(180 * time.Second)
	assert.Equal(t, odigosv1.TraceSurgePhaseBoosting, phases()["payments"])
}

func TestMaxBoostedWorkloads(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	withLimits(t, c, "    maxBoostedWorkloads: 1\n")
	t0 := time.Unix(1_800_000_000, 0)
	// frontend spikes first and raises itself.
	m.services[frontend] = &serviceMetrics{namespace: "shop", serviceName: "frontend", redMetrics: redMetrics{calls: 2000, errors: 1000, p95Ms: 12}}
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0.Add(40*time.Second)))
	require.NoError(t, e.evaluate(ctx, t0.Add(70*time.Second)))
	all := surges(t, c)
	require.Len(t, all, 2)
	surge := all[1]
	require.Equal(t, payments, surge.Spec.Service)
	assert.Equal(t, odigosv1.TraceSurgePhaseLimited, surge.Status.Phase, "raising payments would raise 2 workloads with frontend")
	assert.Equal(t, "Raising shop/payments would sample 2 workloads at a raised percentage, above the limit of 1 (sampling.traceSurge.maxBoostedWorkloads).",
		surge.Status.LimitReason)

	// the spike ends while limited: the surge ends without having raised anything.
	m.paymentsErrorRate(0)
	require.NoError(t, e.evaluate(ctx, t0.Add(80*time.Second)))
	surge = surges(t, c)[1]
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, surge.Status.Phase)
	assert.Nil(t, surge.Status.BoostedAt)
	assert.Equal(t, "Surge ended", surge.Status.Timeline[len(surge.Status.Timeline)-1].Title)
}

func TestASurgeRaisesAtMostMaxBoostedWorkloads(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	withLimits(t, c, "    maxBoostedWorkloads: 1\n")
	t0 := time.Unix(1_800_000_000, 0)
	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	surge := surges(t, c)[0]
	assert.Equal(t, odigosv1.TraceSurgePhaseBoosting, surge.Status.Phase)
	assert.Equal(t, []odigosv1.TraceSurgeTarget{{Workload: payments}}, surge.Spec.Targets, "the nearest target: the service itself")
	last := surge.Status.Timeline[len(surge.Status.Timeline)-1]
	assert.Equal(t, leftOutTitle, last.Title)
	assert.Equal(t, "shop/frontend also lead to shop/payments. Raising them would sample more than 1 workload at a raised percentage, the limit (sampling.traceSurge.maxBoostedWorkloads).",
		last.Description)

	require.NoError(t, e.evaluate(ctx, t0.Add(40*time.Second)))
	surge = surges(t, c)[0]
	assert.Equal(t, []odigosv1.TraceSurgeTarget{{Workload: payments}}, surge.Spec.Targets)
	assert.Equal(t, 1, countEvents(surge, leftOutTitle), "recorded once")
}

func TestNewCallersAreRaisedWithinMaxBoostedWorkloads(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	withLimits(t, c, "    maxBoostedWorkloads: 2\n")
	checkout := k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "checkout"}
	require.NoError(t, c.Create(ctx, ic(checkout)))
	t0 := time.Unix(1_800_000_000, 0)
	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	require.Equal(t, []odigosv1.TraceSurgeTarget{{Workload: payments}, {Workload: frontend}}, surges(t, c)[0].Spec.Targets)

	// checkout starts calling payments: raising it too would sample 3 workloads.
	m.callers["shop/payments"] = []string{"shop/frontend", "shop/checkout"}
	for at := 40 * time.Second; at <= 60*time.Second; at += 10 * time.Second {
		require.NoError(t, e.evaluate(ctx, t0.Add(at)))
	}
	surge := surges(t, c)[0]
	assert.Equal(t, []odigosv1.TraceSurgeTarget{{Workload: payments}, {Workload: frontend}}, surge.Spec.Targets)
	assert.Equal(t, 1, countEvents(surge, leftOutTitle))
	assert.Contains(t, surge.Status.Timeline[len(surge.Status.Timeline)-1].Description, "shop/checkout also lead to shop/payments")

	withLimits(t, c, "    maxBoostedWorkloads: 3\n")
	require.NoError(t, e.evaluate(ctx, t0.Add(70*time.Second)))
	surge = surges(t, c)[0]
	assert.Equal(t, []odigosv1.TraceSurgeTarget{{Workload: payments}, {Workload: frontend}, {Workload: checkout}}, surge.Spec.Targets)
	assert.Equal(t, "More services raised", surge.Status.Timeline[len(surge.Status.Timeline)-1].Title)
}

func TestLoweredMaxActiveSurgesEndsTheNewestSurges(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	m.services[frontend] = &serviceMetrics{namespace: "shop", serviceName: "frontend", redMetrics: redMetrics{calls: 2000, errors: 1000, p95Ms: 12}}
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0.Add(40*time.Second)))
	require.NoError(t, e.evaluate(ctx, t0.Add(70*time.Second)))
	phase := func(pw k8sconsts.PodWorkload) odigosv1.TraceSurgePhase {
		for _, s := range surges(t, c) {
			if s.Spec.Service == pw && s.Status.Phase != odigosv1.TraceSurgePhaseRestored {
				return s.Status.Phase
			}
		}
		return odigosv1.TraceSurgePhaseRestored
	}
	require.Equal(t, odigosv1.TraceSurgePhaseBoosting, phase(frontend))
	require.Equal(t, odigosv1.TraceSurgePhaseBoosting, phase(payments))

	withLimits(t, c, "    maxActiveSurges: 1\n")
	require.NoError(t, e.evaluate(ctx, t0.Add(80*time.Second)))
	assert.Equal(t, odigosv1.TraceSurgePhaseBoosting, phase(frontend), "the oldest keeps raising sampling")
	ended := surges(t, c)[1]
	require.Equal(t, payments, ended.Spec.Service)
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, ended.Status.Phase)
	assert.Equal(t, "sampling.traceSurge.maxActiveSurges was lowered to 1 while more surges raised sampling: the newest ones ended.", ended.Status.RestoreReason)

	withLimits(t, c, "    maxActiveSurges: 0\n")
	require.NoError(t, e.evaluate(ctx, t0.Add(90*time.Second)))
	assert.False(t, raised(surges(t, c), frontend))
	assert.False(t, raised(surges(t, c), payments))
	assert.Equal(t, "Trace surges are turned off: sampling.traceSurge.maxActiveSurges is 0.", surges(t, c)[0].Status.RestoreReason)
}

func countEvents(surge odigosv1.TraceSurge, title string) int {
	n := 0
	for _, event := range surge.Status.Timeline {
		if event.Title == title {
			n++
		}
	}
	return n
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
	e, c, m, _ := setup(t)
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
	for at := 40 * time.Second; at <= 140*time.Second; at += 10 * time.Second {
		tick(at)
	}

	// insights is down when the surge ends: it waits in the status.
	archive.err = errors.New("insights unavailable")
	tick(150 * time.Second)
	require.Len(t, openSurges(t, c), 1)
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, openSurges(t, c)[0].Status.Phase)
	assert.Equal(t, 1, archive.puts)

	archive.err = nil
	tick(160 * time.Second)
	assert.Len(t, openSurges(t, c), 1, "no put until the retry time, so insights being down doesn't slow each evaluation")
	assert.Equal(t, 1, archive.puts)
	tick(185 * time.Second)
	assert.Empty(t, openSurges(t, c), "it moved to insights")
	require.Contains(t, archive.surges, name)
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, archive.surges[name].Status.Phase)
	assert.Equal(t, "Sampling restored to 1%", archive.surges[name].Status.Timeline[len(archive.surges[name].Status.Timeline)-1].Title)
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
	assert.Len(t, openSurges(t, c), 1, "it holds the service back, long after it ended")
	assert.Empty(t, archives[t.Name()].surges)

	m.paymentsErrorRate(1)
	tick(31 * time.Minute)
	assert.Empty(t, openSurges(t, c), "the service recovered: it moved to insights")
	assert.Len(t, archives[t.Name()].surges, 1)
}

func TestOldEndedSurgesAreDroppedFromTheStatus(t *testing.T) {
	ctx := context.Background()
	e, c, m, ruleID := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	ended := func(name string, service k8sconsts.PodWorkload, startedAt time.Time, reason string) odigosv1.TraceSurge {
		restored := metav1.NewTime(startedAt.Add(time.Minute))
		return odigosv1.TraceSurge{
			Name:      name,
			StartedAt: metav1.NewTime(startedAt),
			Spec:      odigosv1.TraceSurgeSpec{SamplingName: "default", RuleID: ruleID, Service: service},
			Status:    odigosv1.TraceSurgeStatus{Phase: odigosv1.TraceSurgePhaseRestored, RestoredAt: &restored, RestoreReason: reason},
		}
	}
	names := func() []string {
		var out []string
		for _, s := range surges(t, c) {
			out = append(out, s.Name)
		}
		return out
	}

	// the oldest ended at the maximum duration, and frontend has not recovered since.
	sampling := &odigosv1.Sampling{}
	require.NoError(t, c.Get(ctx, client.ObjectKey{Namespace: ns, Name: "default"}, sampling))
	sampling.Status.TraceSurges = []odigosv1.TraceSurge{ended("held", frontend, t0.Add(-time.Hour), maxDurationReason+" 30m0s. shop/frontend must recover before it can surge again.")}
	for i := range 25 {
		sampling.Status.TraceSurges = append(sampling.Status.TraceSurges,
			ended("ended-"+strconv.Itoa(i), payments, t0.Add(time.Duration(i-30)*time.Minute), "Error rate stayed at or below 2% for 60s."))
	}
	require.NoError(t, c.Status().Update(ctx, sampling))
	m.services[frontend] = &serviceMetrics{namespace: "shop", serviceName: "frontend", redMetrics: redMetrics{calls: 2000, errors: 1000, p95Ms: 12}}
	// insights is down: the status falls back to keeping the newest ended surges.
	archives[t.Name()].err = errors.New("insights unavailable")

	require.NoError(t, e.evaluate(ctx, t0))
	kept := names()
	assert.Len(t, kept, maxEndedInStatus+1, "the newest 20, and the one that holds frontend back")
	assert.Contains(t, kept, "held")
	assert.Contains(t, kept, "ended-5")
	assert.NotContains(t, kept, "ended-4")

	m.services[frontend] = &serviceMetrics{namespace: "shop", serviceName: "frontend", redMetrics: redMetrics{calls: 2000, errors: 0, p95Ms: 12}}
	require.NoError(t, e.evaluate(ctx, t0.Add(10*time.Second)))
	kept = names()
	assert.Len(t, kept, maxEndedInStatus)
	assert.NotContains(t, kept, "held", "frontend recovered: nothing holds it back")
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

func TestHoldIsReleasedWhenNothingCanRaiseTheService(t *testing.T) {
	cases := map[string]func(t *testing.T, c client.Client){
		"rule removed": func(t *testing.T, c client.Client) {
			sampling := &odigosv1.Sampling{}
			require.NoError(t, c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "default"}, sampling))
			sampling.Spec.NoisyOperations[0].Surge = nil
			require.NoError(t, c.Update(context.Background(), sampling))
		},
		"service no longer instrumented": func(t *testing.T, c client.Client) {
			require.NoError(t, c.Delete(context.Background(), ic(payments)))
		},
		"trace surges turned off": func(t *testing.T, c client.Client) {
			off := effectiveConfig(true)
			off.Data[consts.OdigosConfigurationFileName] = "insights:\n  enabled: true\n"
			require.NoError(t, c.Update(context.Background(), off))
		},
		"held for a day": func(*testing.T, client.Client) {},
	}
	for name, release := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			e, c, m, _ := setup(t)
			withLimits(t, c, "    maxDuration: 2m\n")
			t0 := time.Unix(1_800_000_000, 0)
			m.paymentsErrorRate(50)
			require.NoError(t, e.evaluate(ctx, t0))
			require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
			require.NoError(t, e.evaluate(ctx, t0.Add(150*time.Second)))
			require.True(t, e.holdsBack(&surges(t, c)[0]), "ended at the maximum duration, still spiking")

			release(t, c)
			at := t0.Add(160 * time.Second)
			if name == "held for a day" {
				require.NoError(t, e.evaluate(ctx, t0.Add(150*time.Second+maxHoldBack-time.Second)))
				require.True(t, e.holdsBack(&surges(t, c)[0]))
				at = t0.Add(150*time.Second + maxHoldBack)
			}
			require.NoError(t, e.evaluate(ctx, at))
			ended := surges(t, c)[0]
			assert.False(t, e.holdsBack(&ended))
			assert.Equal(t, "No longer held back", ended.Status.Timeline[len(ended.Status.Timeline)-1].Title)
			assert.Empty(t, openSurges(t, c), "it moved to insights")
		})
	}
}

func TestReleasedServiceCanSurgeAgain(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	withLimits(t, c, "    maxDuration: 2m\n")
	t0 := time.Unix(1_800_000_000, 0)
	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	require.NoError(t, e.evaluate(ctx, t0.Add(150*time.Second)))
	ended := t0.Add(150 * time.Second)

	// still spiking a day later: the hold ends, and so does the wait for a recovery.
	require.NoError(t, e.evaluate(ctx, ended.Add(maxHoldBack)))
	require.NoError(t, e.evaluate(ctx, ended.Add(maxHoldBack+10*time.Second)))
	require.NoError(t, e.evaluate(ctx, ended.Add(maxHoldBack+40*time.Second)))
	all := surges(t, c)
	require.Len(t, all, 2)
	assert.Equal(t, odigosv1.TraceSurgePhaseBoosting, all[1].Status.Phase)
}

func TestMalformedStatusDoesNotStopTheEvaluator(t *testing.T) {
	ctx := context.Background()
	e, c, m, ruleID := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	sampling := &odigosv1.Sampling{}
	require.NoError(t, c.Get(ctx, client.ObjectKey{Namespace: ns, Name: "default"}, sampling))
	settings := *sampling.Spec.NoisyOperations[0].Surge
	sampling.Status.TraceSurges = []odigosv1.TraceSurge{
		{Name: "boosting", StartedAt: metav1.NewTime(t0), Spec: odigosv1.TraceSurgeSpec{SamplingName: "default", RuleID: ruleID, Settings: settings, Service: payments},
			Status: odigosv1.TraceSurgeStatus{Phase: odigosv1.TraceSurgePhaseBoosting}},
		{Name: "recovering", StartedAt: metav1.NewTime(t0), Spec: odigosv1.TraceSurgeSpec{SamplingName: "default", RuleID: ruleID, Settings: settings, Service: frontend},
			Status: odigosv1.TraceSurgeStatus{Phase: odigosv1.TraceSurgePhaseRecovering}},
	}
	require.NoError(t, c.Status().Update(ctx, sampling))
	m.paymentsErrorRate(1)

	require.NoError(t, e.evaluate(ctx, t0.Add(10*time.Second)))
	for _, surge := range surges(t, c) {
		require.NotNil(t, surge.Status.BoostedAt, surge.Name)
		assert.Equal(t, t0.Add(10*time.Second), surge.Status.BoostedAt.Time, "boosting as of now")
		require.NotNil(t, surge.Status.RecoveryStartedAt, surge.Name)
		assert.Equal(t, odigosv1.TraceSurgePhaseRecovering, surge.Status.Phase, surge.Name)
	}
}

type panickingMetrics struct{}

func (panickingMetrics) read(context.Context, time.Duration, int) (map[k8sconsts.PodWorkload]*serviceMetrics, map[string][]string, error) {
	panic("boom")
}

func TestEvaluationPanicIsLogged(t *testing.T) {
	e, _, _, _ := setup(t)
	var logged []string
	e.Logger = funcr.New(func(_, args string) { logged = append(logged, args) }, funcr.Options{})
	e.metrics = panickingMetrics{}

	assert.NotPanics(t, func() { e.evaluateSafely(context.Background(), time.Unix(1_800_000_000, 0)) })
	require.Len(t, logged, 1)
	assert.Contains(t, logged[0], "trace surge evaluation panicked")
	assert.Contains(t, logged[0], "boom")
}

func TestRecoveryWindowRestartsAfterAMetricsGap(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	tick := func(at time.Duration) {
		t.Helper()
		require.NoError(t, e.evaluate(ctx, t0.Add(at)))
	}
	m.paymentsErrorRate(50)
	tick(0)
	tick(30 * time.Second)
	m.paymentsErrorRate(0)
	for at := 40 * time.Second; at <= 140*time.Second; at += 10 * time.Second {
		tick(at)
	}
	require.Equal(t, odigosv1.TraceSurgePhaseRecovering, surges(t, c)[0].Status.Phase, "boosted for 110s < minimumBoostSeconds")

	// odigos insights is down for a minute: one healthy observation after it is not 60s of recovery.
	m.err = errors.New("insights unavailable")
	for at := 150 * time.Second; at <= 200*time.Second; at += 10 * time.Second {
		tick(at)
	}
	m.err = nil
	tick(210 * time.Second)
	surge := surges(t, c)[0]
	assert.Equal(t, odigosv1.TraceSurgePhaseRecovering, surge.Status.Phase)
	assert.Equal(t, t0.Add(210*time.Second), surge.Status.RecoveryStartedAt.Time)
	assert.Equal(t, "Recovery window restarted", surge.Status.Timeline[len(surge.Status.Timeline)-1].Title)

	for at := 220 * time.Second; at <= 260*time.Second; at += 10 * time.Second {
		tick(at)
	}
	assert.Equal(t, odigosv1.TraceSurgePhaseRecovering, surges(t, c)[0].Status.Phase, "50s since the window restarted")
	tick(270 * time.Second)
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, surges(t, c)[0].Status.Phase)
}

func TestIncompleteMetricsNeverCountAsRecovery(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))

	m.paymentsErrorRate(0)
	m.services[payments].incomplete = true
	require.NoError(t, e.evaluate(ctx, t0.Add(40*time.Second)))
	assert.Equal(t, odigosv1.TraceSurgePhaseBoosting, surges(t, c)[0].Status.Phase, "insights dropped some of payments' series")
}

func TestInsightsWithTraceSurgesOffIsLoggedOnce(t *testing.T) {
	ctx := context.Background()
	e, _, m, _ := setup(t)
	var logged []string
	e.Logger = funcr.New(func(_, args string) { logged = append(logged, args) }, funcr.Options{})
	m.err = errInsightsTraceSurgesOff
	for i := range 3 {
		require.NoError(t, e.evaluate(ctx, time.Unix(1_800_000_000+int64(i)*10, 0)))
	}
	require.Len(t, logged, 1)
	assert.Contains(t, logged[0], "odigos insights has trace surges turned off while sampling.traceSurge.enabled is on")
}

func TestSurgeNamesTellApartServicesOfTheSameName(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	shopPayments := surgeName(surgeKey{"default", "44136fa3c0ffee00", payments}, now)
	otherPayments := surgeName(surgeKey{"default", "44136fa3c0ffee00", k8sconsts.PodWorkload{Namespace: "other", Kind: k8sconsts.WorkloadKindDeployment, Name: "payments"}}, now)
	assert.Regexp(t, `^44136fa3-payments-tro8w0-[0-9a-f]{16}$`, shopPayments, "a 64-bit hash")
	assert.NotEqual(t, shopPayments, otherPayments, "same rule, workload name and second")
	assert.Equal(t, shopPayments, surgeName(surgeKey{"default", "44136fa3c0ffee00", payments}, now))
}

// A failing dependency makes every service that calls it breach at once.
func TestManyBreachingServicesKeepTheStatusSmall(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	const n = 300
	failed := 0
	e.Logger = funcr.New(func(_, args string) {
		if strings.Contains(args, "failed to write the trace surges") {
			failed++
		}
	}, funcr.Options{})
	for i := range n {
		pw := k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: fmt.Sprintf("service-%03d", i)}
		require.NoError(t, c.Create(ctx, ic(pw)))
		m.services[pw] = &serviceMetrics{namespace: "shop", serviceName: pw.Name, redMetrics: redMetrics{calls: 1000, errors: 500, p95Ms: math.NaN()}}
		if i > 0 {
			m.callers[fmt.Sprintf("shop/service-%03d", i-1)] = []string{"shop/" + pw.Name}
		}
	}
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))

	status := openSurges(t, c)
	data, err := json.Marshal(status)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(data), maxStatusBytes)
	phases := map[odigosv1.TraceSurgePhase]int{}
	for _, surge := range status {
		phases[surge.Status.Phase]++
		assert.LessOrEqual(t, len(surge.Spec.Targets), defaultMaxBoostedWorkloads)
	}
	assert.Equal(t, maxLimitedInStatus, phases[odigosv1.TraceSurgePhaseLimited])
	assert.Positive(t, phases[odigosv1.TraceSurgePhaseBoosting])
	assert.Zero(t, failed, "every status write succeeds")
	open := 0
	for _, state := range e.samplings {
		open += len(state.surges)
	}
	assert.Equal(t, n, open, "a surge for each service: the ones not in the status are kept in memory")

	// the dependency recovers: every surge ends, and the restorations are written.
	for i := range n {
		m.services[k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: fmt.Sprintf("service-%03d", i)}].errors = 0
	}
	for at := 40 * time.Second; at <= 160*time.Second; at += 10 * time.Second {
		require.NoError(t, e.evaluate(ctx, t0.Add(at)))
	}
	for _, surge := range openSurges(t, c) {
		assert.False(t, surge.Active(), surge.Name)
	}
	assert.Zero(t, failed)
}

func TestStatusKeepsWhatDecidesSamplingFirst(t *testing.T) {
	t0 := time.Unix(1_800_000_000, 0)
	surge := func(name string, phase odigosv1.TraceSurgePhase, startedAt time.Duration, kib int) *odigosv1.TraceSurge {
		return &odigosv1.TraceSurge{Name: name, StartedAt: metav1.NewTime(t0.Add(startedAt)),
			Spec:   odigosv1.TraceSurgeSpec{SamplingName: "default", RuleID: "rule", Service: k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: name}},
			Status: odigosv1.TraceSurgeStatus{Phase: phase, Timeline: []odigosv1.TraceSurgeEvent{{Description: strings.Repeat("x", kib<<10)}}}}
	}
	names := func(surges []odigosv1.TraceSurge) []string {
		var out []string
		for _, s := range surges {
			out = append(out, s.Name)
		}
		return out
	}
	held := surge("held", odigosv1.TraceSurgePhaseRestored, -time.Hour, 60)
	held.Status.RestoreReason = maxDurationReason + " 30m0s."
	e := &Evaluator{latched: map[surgeKey]bool{keyOf(held): true}}
	all := []*odigosv1.TraceSurge{surge("boosting", odigosv1.TraceSurgePhaseBoosting, 0, 100), surge("restoring", odigosv1.TraceSurgePhaseRestored, 0, 100), held}
	for i := range 6 {
		all = append(all, surge(fmt.Sprintf("ended-%d", i), odigosv1.TraceSurgePhaseRestored, time.Duration(i)*time.Minute, 60))
	}
	for i := range 12 {
		all = append(all, surge(fmt.Sprintf("limited-%02d", i), odigosv1.TraceSurgePhaseLimited, time.Duration(i)*time.Minute, 1))
	}
	written := []odigosv1.TraceSurge{{Name: "restoring", Status: odigosv1.TraceSurgeStatus{Phase: odigosv1.TraceSurgePhaseBoosting}}}

	kept := e.statusSurges(all, written)
	assert.Equal(t, []string{"boosting", "restoring", "held", "ended-2", "ended-3", "ended-4", "ended-5"}, names(kept),
		"what the agents sample by, then the ended surges, the one holding its service back first, while they fit; limited surges are dropped first")
	data, err := json.Marshal(kept)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(data), maxStatusBytes)

	kept = e.statusSurges(all[3:], nil)
	assert.Equal(t, []string{"ended-0", "ended-1", "ended-2", "ended-3", "ended-4", "ended-5",
		"limited-02", "limited-03", "limited-04", "limited-05", "limited-06", "limited-07", "limited-08", "limited-09", "limited-10", "limited-11"}, names(kept),
		"the newest limited surges")
}

func TestLimitedSurgeKeepsTheTargetsALoweredLimitAllows(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	withLimits(t, c, "    maxActiveSurges: 1\n    maxBoostedWorkloads: 2\n")
	t0 := time.Unix(1_800_000_000, 0)
	tick := func(at time.Duration) {
		t.Helper()
		require.NoError(t, e.evaluate(ctx, t0.Add(at)))
	}
	m.services[frontend] = &serviceMetrics{namespace: "shop", serviceName: "frontend", redMetrics: redMetrics{calls: 2000, errors: 1000, p95Ms: 12}}
	tick(0)
	tick(30 * time.Second)
	m.paymentsErrorRate(50)
	tick(40 * time.Second)
	tick(70 * time.Second)
	limited := surges(t, c)[1]
	require.Equal(t, odigosv1.TraceSurgePhaseLimited, limited.Status.Phase)
	require.Equal(t, []odigosv1.TraceSurgeTarget{{Workload: payments}, {Workload: frontend}}, limited.Spec.Targets)

	withLimits(t, c, "    maxActiveSurges: 1\n    maxBoostedWorkloads: 1\n")
	tick(80 * time.Second)
	assert.Equal(t, []odigosv1.TraceSurgeTarget{{Workload: payments}}, surges(t, c)[1].Spec.Targets, "the nearest one")

	// once frontend's surge ends, payments' fits the lowered limit.
	m.services[frontend] = &serviceMetrics{namespace: "shop", serviceName: "frontend", redMetrics: redMetrics{calls: 2000, errors: 0, p95Ms: 12}}
	for at := 90 * time.Second; at <= 160*time.Second; at += 10 * time.Second {
		tick(at)
	}
	all := surges(t, c)
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, all[0].Status.Phase)
	assert.Equal(t, odigosv1.TraceSurgePhaseBoosting, all[1].Status.Phase)
}

func TestHoldReleaseSurvivesARestart(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	withLimits(t, c, "    maxDuration: 2m\n")
	t0 := time.Unix(1_800_000_000, 0)
	setSurge := func(surge *odigosv1.TraceSurgeSettings) {
		t.Helper()
		sampling := &odigosv1.Sampling{}
		require.NoError(t, c.Get(ctx, client.ObjectKey{Namespace: ns, Name: "default"}, sampling))
		sampling.Spec.NoisyOperations[0].Surge = surge
		require.NoError(t, c.Update(ctx, sampling))
	}
	sampling := &odigosv1.Sampling{}
	require.NoError(t, c.Get(ctx, client.ObjectKey{Namespace: ns, Name: "default"}, sampling))
	settings := sampling.Spec.NoisyOperations[0].Surge
	// insights is down: the surge that held payments back stays in the status.
	archives[t.Name()].err = errors.New("insights unavailable")

	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	require.NoError(t, e.evaluate(ctx, t0.Add(150*time.Second)))
	setSurge(nil)
	require.NoError(t, e.evaluate(ctx, t0.Add(160*time.Second)))
	require.NotNil(t, surges(t, c)[0].Status.HoldReleasedAt, "the release is in the status")

	// the rule is back, and a restarted evaluator does not hold the still spiking service back again.
	setSurge(settings)
	restarted := &Evaluator{Client: c, Logger: e.Logger}
	restarted.init(ns, e.metrics, e.archive)
	require.NoError(t, restarted.evaluate(ctx, t0.Add(170*time.Second)))
	require.NoError(t, restarted.evaluate(ctx, t0.Add(200*time.Second)))
	all := surges(t, c)
	require.Len(t, all, 2)
	assert.Equal(t, odigosv1.TraceSurgePhaseBoosting, all[1].Status.Phase)
}

func TestLoweredMaxBoostedWorkloadsEndsTheNewestSurges(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	m.services[frontend] = &serviceMetrics{namespace: "shop", serviceName: "frontend", redMetrics: redMetrics{calls: 2000, errors: 1000, p95Ms: 12}}
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0.Add(40*time.Second)))
	require.NoError(t, e.evaluate(ctx, t0.Add(70*time.Second)))
	require.True(t, raised(surges(t, c), payments))

	// frontend's surge raises frontend, payments' raises payments and frontend.
	withLimits(t, c, "    maxBoostedWorkloads: 1\n")
	require.NoError(t, e.evaluate(ctx, t0.Add(80*time.Second)))
	all := surges(t, c)
	assert.Equal(t, odigosv1.TraceSurgePhaseBoosting, all[0].Status.Phase, "the oldest fits the lowered limit")
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, all[1].Status.Phase)
	assert.Equal(t, "sampling.traceSurge.maxBoostedWorkloads was lowered to 1 while the surges raised more workloads: the newest ones ended.", all[1].Status.RestoreReason)
	assert.False(t, raised(all, payments))
}

func TestSurgesThatWouldOverflowTheStatusStayLimited(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	withLimits(t, c, "    maxActiveSurges: 100\n    maxBoostedWorkloads: 1000\n")
	t0 := time.Unix(1_800_000_000, 0)
	var callers []string
	for i := range 600 {
		pw := k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: fmt.Sprintf("caller-%03d", i)}
		require.NoError(t, c.Create(ctx, ic(pw)))
		callers = append(callers, "shop/"+pw.Name)
	}
	for i := range 15 {
		pw := k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: fmt.Sprintf("service-%02d", i)}
		require.NoError(t, c.Create(ctx, ic(pw)))
		m.services[pw] = &serviceMetrics{namespace: "shop", serviceName: pw.Name, redMetrics: redMetrics{calls: 1000, errors: 500, p95Ms: math.NaN()}}
		m.callers["shop/"+pw.Name] = callers
	}
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))

	data, err := json.Marshal(openSurges(t, c))
	require.NoError(t, err)
	assert.LessOrEqual(t, len(data), maxStatusBytes)
	phases := map[odigosv1.TraceSurgePhase]int{}
	for _, surge := range e.samplings["default"].surges {
		phases[surge.Status.Phase]++
		if surge.Status.Phase == odigosv1.TraceSurgePhaseLimited {
			assert.Contains(t, surge.Status.LimitReason, "The Sampling status has no room for another surge that raises sampling")
		}
	}
	assert.Positive(t, phases[odigosv1.TraceSurgePhaseBoosting])
	assert.Positive(t, phases[odigosv1.TraceSurgePhaseLimited], "the surges the status has no room for")
	assert.Equal(t, 15, phases[odigosv1.TraceSurgePhaseBoosting]+phases[odigosv1.TraceSurgePhaseLimited])
}

func TestSurgePayloadIsBounded(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	for i := range 15 {
		pw := k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: fmt.Sprintf("caller-%02d", i)}
		require.NoError(t, c.Create(ctx, ic(pw)))
		m.callers["shop/payments"] = append(m.callers["shop/payments"], "shop/"+pw.Name)
	}
	operation := "POST /" + strings.Repeat("x", 1000)
	m.services[payments] = &serviceMetrics{namespace: "shop", serviceName: "payments",
		redMetrics: redMetrics{calls: 1000, errors: 10, p95Ms: math.NaN()},
		operations: []operationMetrics{{name: operation, redMetrics: redMetrics{calls: 500, errors: 250, p95Ms: math.NaN()}}},
	}
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	surge := surges(t, c)[0]
	assert.Len(t, surge.Spec.Targets, 17)
	assert.LessOrEqual(t, len(surge.Status.Trigger.Operation), maxOperation)
	for _, sample := range surge.Status.TriggerSamples {
		assert.LessOrEqual(t, len(sample.Operation), maxOperation)
	}
	raisedEvent := surge.Status.Timeline[1]
	require.Equal(t, "Sampling raised to 80%", raisedEvent.Title)
	assert.Contains(t, raisedEvent.Description, " and 7 more are sampled at 80%", "10 of the 17 targets are named")

	addEvent(&surge, t0, "long", strings.Repeat("é", 1000))
	description := surge.Status.Timeline[len(surge.Status.Timeline)-1].Description
	assert.LessOrEqual(t, len(description), maxDescription)
	assert.True(t, utf8.ValidString(description))
	assert.True(t, strings.HasSuffix(description, "…"))
}

func TestCompactStatusIsWrittenWhenTheFullOneIsRejected(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))
	require.NotEmpty(t, openSurges(t, c)[0].Status.TriggerSamples)

	// the full status is rejected, e.g. for its size: the restoration is written anyway.
	patchChecks[t.Name()] = func(patch []byte) error {
		if strings.Contains(string(patch), "triggerSamples") {
			return errors.New("request entity too large")
		}
		return nil
	}
	withLimits(t, c, "    maxActiveSurges: 0\n")
	archives[t.Name()].err = errors.New("insights unavailable")
	require.NoError(t, e.evaluate(ctx, t0.Add(40*time.Second)))
	surge := openSurges(t, c)[0]
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, surge.Status.Phase)
	assert.Empty(t, surge.Status.TriggerSamples)
	assert.LessOrEqual(t, len(surge.Status.Timeline), compactTimeline)
	assert.Equal(t, "Sampling restored to 1%", surge.Status.Timeline[len(surge.Status.Timeline)-1].Title)
}

func TestTraceSurgeLimitsAreClamped(t *testing.T) {
	huge := 1_000_000
	l := limitsFrom(&common.SamplingConfiguration{TraceSurge: &common.TraceSurgeConfiguration{MaxActiveSurges: &huge, MaxBoostedWorkloads: &huge}})
	assert.Equal(t, maxMaxActiveSurges, l.maxActive)
	assert.Equal(t, maxMaxBoostedWorkloads, l.maxBoosted)
}

func TestRestorationIsWrittenBeforeArchiving(t *testing.T) {
	ctx := context.Background()
	e, c, m, _ := setup(t)
	t0 := time.Unix(1_800_000_000, 0)
	m.paymentsErrorRate(50)
	require.NoError(t, e.evaluate(ctx, t0))
	require.NoError(t, e.evaluate(ctx, t0.Add(30*time.Second)))

	var phaseWhenArchived odigosv1.TraceSurgePhase
	archives[t.Name()].onPut = func(*odigosv1.TraceSurge) { phaseWhenArchived = openSurges(t, c)[0].Status.Phase }
	withLimits(t, c, "    maxActiveSurges: 0\n")
	writes := *statusWrites[t.Name()]
	require.NoError(t, e.evaluate(ctx, t0.Add(40*time.Second)))
	assert.Equal(t, odigosv1.TraceSurgePhaseRestored, phaseWhenArchived, "the agents' sampling is restored first")
	assert.Empty(t, openSurges(t, c), "then it moved to insights")
	assert.Equal(t, writes+2, *statusWrites[t.Name()], "the restoration, then the move")
}

func TestArchivingTakesAtMostItsBudget(t *testing.T) {
	ctx := context.Background()
	e, c, _, ruleID := setup(t)
	archive := archives[t.Name()]
	t0 := time.Unix(1_800_000_000, 0)
	sampling := &odigosv1.Sampling{}
	require.NoError(t, c.Get(ctx, client.ObjectKey{Namespace: ns, Name: "default"}, sampling))
	restored := metav1.NewTime(t0)
	for i := range 10 {
		sampling.Status.TraceSurges = append(sampling.Status.TraceSurges, odigosv1.TraceSurge{Name: fmt.Sprintf("ended-%d", i), StartedAt: restored,
			Spec:   odigosv1.TraceSurgeSpec{SamplingName: "default", RuleID: ruleID, Service: payments},
			Status: odigosv1.TraceSurgeStatus{Phase: odigosv1.TraceSurgePhaseRestored, RestoredAt: &restored}})
	}
	require.NoError(t, c.Status().Update(ctx, sampling))
	// the fourth put would end past the budget.
	archive.delay = archiveBudget * 3 / 10

	began := time.Now()
	require.NoError(t, e.evaluate(ctx, t0.Add(10*time.Second)))
	assert.Less(t, time.Since(began), archiveBudget+time.Second)
	moved := len(archive.surges)
	assert.Equal(t, 3, moved, "a put cut short by the budget does not count")
	assert.Len(t, openSurges(t, c), 10-moved)

	require.NoError(t, e.evaluate(ctx, t0.Add(20*time.Second)))
	assert.Len(t, archive.surges, 2*moved, "out of time is not a failure: no back-off")
}
