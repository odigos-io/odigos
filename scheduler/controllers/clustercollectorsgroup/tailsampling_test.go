package clustercollectorsgroup

import (
	"context"
	"errors"
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func tsScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := odigosv1.AddToScheme(scheme); err != nil {
		t.Fatalf("adding odigos types to scheme: %v", err)
	}
	return scheme
}

func tsSampling(namespace, name string) *odigosv1.Sampling {
	return &odigosv1.Sampling{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
	}
}

func tsBoolPtr(b bool) *bool { return &b }

func tsStringPtr(s string) *string { return &s }

// TestIsTailSamplingEnabled covers the gate that decides whether the cluster collector is
// configured with tail sampling at all. It is now "any Sampling CR exists", after the
// object-level spec.disabled flag was removed in favour of the per-rule disabled flags, which
// the samplers themselves honour while still emitting metrics for disabled rules.
//
// Both directions are expensive to get wrong: leaving it off drops the sampling rules a user
// configured, and turning it on unconditionally adds the trace aggregation wait to the pipeline
// (latency and collector memory) for clusters that never asked for sampling.
func TestIsTailSamplingEnabled(t *testing.T) {
	odigosNamespace := env.GetCurrentNamespace()

	tests := []struct {
		name     string
		config   common.OdigosConfiguration
		existing []client.Object
		want     bool
	}{
		{
			name:   "no sampling CRs",
			config: common.OdigosConfiguration{},
			want:   false,
		},
		{
			name:     "one sampling CR",
			config:   common.OdigosConfiguration{},
			existing: []client.Object{tsSampling(odigosNamespace, "noisy-ops")},
			want:     true,
		},
		{
			name:   "several sampling CRs",
			config: common.OdigosConfiguration{},
			existing: []client.Object{
				tsSampling(odigosNamespace, "noisy-ops"),
				tsSampling(odigosNamespace, "cost-reduction"),
			},
			want: true,
		},
		{
			name: "globally disabled with a sampling CR present",
			config: common.OdigosConfiguration{
				Sampling: &common.SamplingConfiguration{
					TailSampling: &sampling.TailSamplingConfiguration{Disabled: tsBoolPtr(true)},
				},
			},
			existing: []client.Object{tsSampling(odigosNamespace, "noisy-ops")},
			want:     false,
		},
		{
			name: "globally enabled explicitly with a sampling CR present",
			config: common.OdigosConfiguration{
				Sampling: &common.SamplingConfiguration{
					TailSampling: &sampling.TailSamplingConfiguration{Disabled: tsBoolPtr(false)},
				},
			},
			existing: []client.Object{tsSampling(odigosNamespace, "noisy-ops")},
			want:     true,
		},
		{
			name: "globally enabled explicitly with no sampling CR",
			config: common.OdigosConfiguration{
				Sampling: &common.SamplingConfiguration{
					TailSampling: &sampling.TailSamplingConfiguration{Disabled: tsBoolPtr(false)},
				},
			},
			want: false,
		},
		{
			// The global switch is three-state: a nil Disabled must not be read as disabled.
			name: "tail sampling configured without the disabled flag",
			config: common.OdigosConfiguration{
				Sampling: &common.SamplingConfiguration{
					TailSampling: &sampling.TailSamplingConfiguration{TraceAggregationWaitDuration: tsStringPtr("5s")},
				},
			},
			existing: []client.Object{tsSampling(odigosNamespace, "noisy-ops")},
			want:     true,
		},
		{
			name: "sampling configured but tail sampling absent",
			config: common.OdigosConfiguration{
				Sampling: &common.SamplingConfiguration{DryRun: tsBoolPtr(true)},
			},
			existing: []client.Object{tsSampling(odigosNamespace, "noisy-ops")},
			want:     true,
		},
		{
			// Sampling CRs are read from the odigos namespace only, so a CR a user created
			// elsewhere must not switch tail sampling on cluster wide.
			name:     "sampling CR in another namespace only",
			config:   common.OdigosConfiguration{},
			existing: []client.Object{tsSampling("some-other-namespace", "noisy-ops")},
			want:     false,
		},
		{
			name:   "sampling CR in the odigos namespace alongside one elsewhere",
			config: common.OdigosConfiguration{},
			existing: []client.Object{
				tsSampling("some-other-namespace", "noisy-ops"),
				tsSampling(odigosNamespace, "noisy-ops"),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(tsScheme(t)).
				WithObjects(tt.existing...).
				Build()

			if got := isTailSamplingEnabled(context.Background(), c, &tt.config); got != tt.want {
				t.Errorf("isTailSamplingEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsTailSamplingEnabledFailsClosedOnListError pins that a failure to read the Sampling CRs
// is not treated as "no rules configured but keep going". The gate has to pick one, and it picks
// off: turning the processor on with an unknown rule set would let the collector aggregate and
// then drop traces according to whatever it last knew.
func TestIsTailSamplingEnabledFailsClosedOnListError(t *testing.T) {
	odigosNamespace := env.GetCurrentNamespace()
	wantErr := errors.New("sampling list is unavailable")

	c := fake.NewClientBuilder().
		WithScheme(tsScheme(t)).
		// A CR the list would have found, so the false can only come from the error path.
		WithObjects(tsSampling(odigosNamespace, "noisy-ops")).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, cl client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if _, ok := list.(*odigosv1.SamplingList); ok {
					return wantErr
				}
				return cl.List(ctx, list, opts...)
			},
		}).
		Build()

	if isTailSamplingEnabled(context.Background(), c, &common.OdigosConfiguration{}) {
		t.Error("isTailSamplingEnabled() = true when listing Sampling CRs failed, want false")
	}
}

// TestResolveTailSamplingConfig covers how the resolved gate decision and the aggregation wait
// duration are written into the collectors group. Disabled is the inverse of enabled, and an
// inverted or dropped negation would hand the collector the opposite of the gate's decision.
func TestResolveTailSamplingConfig(t *testing.T) {
	defaultDuration := resolveTailSamplingConfig(context.Background(), &common.OdigosConfiguration{}, true)
	if defaultDuration.TraceAggregationWaitDuration == nil {
		t.Fatal("TraceAggregationWaitDuration is nil for an empty configuration")
	}
	fallback := *defaultDuration.TraceAggregationWaitDuration
	if fallback == "" {
		t.Fatal("the default TraceAggregationWaitDuration is empty")
	}

	tests := []struct {
		name         string
		configured   *string
		enabled      bool
		wantDuration string
	}{
		{name: "unset duration", configured: nil, enabled: true, wantDuration: fallback},
		{name: "valid duration", configured: tsStringPtr("15s"), enabled: true, wantDuration: "15s"},
		{name: "valid sub-second duration", configured: tsStringPtr("500ms"), enabled: true, wantDuration: "500ms"},
		{name: "unparsable duration falls back", configured: tsStringPtr("15 seconds"), enabled: true, wantDuration: fallback},
		{name: "empty duration falls back", configured: tsStringPtr(""), enabled: true, wantDuration: fallback},
		{name: "zero duration falls back", configured: tsStringPtr("0s"), enabled: true, wantDuration: fallback},
		{name: "negative duration falls back", configured: tsStringPtr("-5s"), enabled: true, wantDuration: fallback},
		{name: "duration is resolved even when disabled", configured: tsStringPtr("15s"), enabled: false, wantDuration: "15s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := common.OdigosConfiguration{}
			if tt.configured != nil {
				config.Sampling = &common.SamplingConfiguration{
					TailSampling: &sampling.TailSamplingConfiguration{TraceAggregationWaitDuration: tt.configured},
				}
			}

			got := resolveTailSamplingConfig(context.Background(), &config, tt.enabled)

			if got.TraceAggregationWaitDuration == nil {
				t.Fatal("TraceAggregationWaitDuration is nil")
			}
			if *got.TraceAggregationWaitDuration != tt.wantDuration {
				t.Errorf("TraceAggregationWaitDuration = %q, want %q", *got.TraceAggregationWaitDuration, tt.wantDuration)
			}
			if got.Disabled == nil {
				t.Fatal("Disabled is nil; the collector reads it to decide whether to run the processor")
			}
			if *got.Disabled != !tt.enabled {
				t.Errorf("Disabled = %v for enabled = %v, want %v", *got.Disabled, tt.enabled, !tt.enabled)
			}
		})
	}
}
