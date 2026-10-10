package clustercollectorsgroup

import (
	"context"
	"errors"
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func hpScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, odigosv1.AddToScheme(scheme))
	return scheme
}

func hpBoolPtr(b bool) *bool { return &b }

func hpSamplingCR(namespace string) *odigosv1.Sampling {
	return &odigosv1.Sampling{
		ObjectMeta: metav1.ObjectMeta{Name: "noisy-ops", Namespace: namespace},
	}
}

func hpConfig(healthProbesEnabled *bool, tailSamplingDisabled *bool) *common.OdigosConfiguration {
	config := &common.OdigosConfiguration{Sampling: &common.SamplingConfiguration{}}
	if healthProbesEnabled != nil {
		config.Sampling.K8sHealthProbesSampling = &common.K8sHealthProbesSamplingConfiguration{
			Enabled: healthProbesEnabled,
		}
	}
	if tailSamplingDisabled != nil {
		config.Sampling.TailSampling = &commonapisampling.TailSamplingConfiguration{
			Disabled: tailSamplingDisabled,
		}
	}
	return config
}

// CORE-1719 made k8s health-probes sampling enough on its own to turn the gateway tail
// sampling processor on, because health-probe noisy operations are applied there for
// sources whose agent cannot head-sample. Every level of the nil chain in front of that
// check must keep the previous "a Sampling CR is required" behaviour.
func TestIsTailSamplingEnabledWithoutSamplingCRsFollowsK8sHealthProbesSampling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config *common.OdigosConfiguration
		want   bool
	}{
		{
			name:   "sampling config absent",
			config: &common.OdigosConfiguration{},
			want:   false,
		},
		{
			name:   "health probes config absent",
			config: hpConfig(nil, nil),
			want:   false,
		},
		{
			name:   "health probes enabled flag absent",
			config: &common.OdigosConfiguration{Sampling: &common.SamplingConfiguration{K8sHealthProbesSampling: &common.K8sHealthProbesSamplingConfiguration{}}},
			want:   false,
		},
		{
			name:   "health probes explicitly disabled",
			config: hpConfig(hpBoolPtr(false), nil),
			want:   false,
		},
		{
			name:   "health probes enabled",
			config: hpConfig(hpBoolPtr(true), nil),
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(hpScheme(t)).Build()

			require.Equal(t, tt.want, isTailSamplingEnabled(context.Background(), c, tt.config))
		})
	}
}

// Enabling health probes sampling must not resurrect tail sampling for a cluster that
// turned it off globally - that is the only switch an operator has to stop paying for the
// gateway tail sampler.
func TestIsTailSamplingEnabledGlobalDisableWinsOverK8sHealthProbesSampling(t *testing.T) {
	t.Parallel()

	namespace := env.GetCurrentNamespace()

	tests := []struct {
		name                 string
		healthProbesEnabled  *bool
		tailSamplingDisabled *bool
		samplingCRs          []client.Object
		want                 bool
	}{
		{
			name:                 "health probes enabled and tail sampling disabled",
			healthProbesEnabled:  hpBoolPtr(true),
			tailSamplingDisabled: hpBoolPtr(true),
			want:                 false,
		},
		{
			name:                 "health probes enabled and tail sampling explicitly not disabled",
			healthProbesEnabled:  hpBoolPtr(true),
			tailSamplingDisabled: hpBoolPtr(false),
			want:                 true,
		},
		{
			name:                 "sampling CR present and tail sampling disabled",
			healthProbesEnabled:  hpBoolPtr(false),
			tailSamplingDisabled: hpBoolPtr(true),
			samplingCRs:          []client.Object{hpSamplingCR(namespace)},
			want:                 false,
		},
		{
			name:                "sampling CR present without health probes sampling",
			healthProbesEnabled: hpBoolPtr(false),
			samplingCRs:         []client.Object{hpSamplingCR(namespace)},
			want:                true,
		},
		{
			name:                "sampling CR in another namespace without health probes sampling",
			healthProbesEnabled: hpBoolPtr(false),
			samplingCRs:         []client.Object{hpSamplingCR("other-namespace")},
			want:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(hpScheme(t)).WithObjects(tt.samplingCRs...).Build()

			got := isTailSamplingEnabled(context.Background(), c, hpConfig(tt.healthProbesEnabled, tt.tailSamplingDisabled))
			require.Equal(t, tt.want, got)
		})
	}
}

// The health-probes branch short circuits before the Sampling CR list, so an unreadable
// Sampling CR list must not be able to turn the gateway tail sampler off for a cluster that
// relies on health-probe sampling.
func TestIsTailSamplingEnabledDoesNotListSamplingCRsWhenHealthProbesSamplingIsEnabled(t *testing.T) {
	t.Parallel()

	listCalls := 0
	newClient := func(t *testing.T) client.Client {
		t.Helper()
		return fake.NewClientBuilder().
			WithScheme(hpScheme(t)).
			WithObjects(hpSamplingCR(env.GetCurrentNamespace())).
			WithInterceptorFuncs(interceptor.Funcs{
				List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					listCalls++
					return errors.New("sampling CRs are unreadable")
				},
			}).
			Build()
	}

	require.True(t, isTailSamplingEnabled(context.Background(), newClient(t), hpConfig(hpBoolPtr(true), nil)))
	require.Zero(t, listCalls)

	// the same unreadable list must still fail closed when health probes sampling is off
	require.False(t, isTailSamplingEnabled(context.Background(), newClient(t), hpConfig(hpBoolPtr(false), nil)))
	require.Equal(t, 1, listCalls)
}
