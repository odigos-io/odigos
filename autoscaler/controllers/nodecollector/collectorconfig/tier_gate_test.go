package collectorconfig

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The odigosebpfreceiver is only compiled into the enterprise collector image. Defining it in an
// OSS collector's config fails at startup with "unknown type: odigosebpf" even when no pipeline
// references it, so both the receiver definition and the pipeline wiring must be tier-gated.

func TestCommonApplicationTelemetryConfig_EbpfReceiverDefinitionGatedByTier(t *testing.T) {
	for _, tc := range []struct {
		name        string
		tier        common.OdigosTier
		wantDefined bool
	}{
		{"community omits the receiver definition", common.CommunityOdigosTier, false},
		{"onprem defines the receiver", common.OnPremOdigosTier, true},
		{"cloud defines the receiver", common.CloudOdigosTier, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := CommonApplicationTelemetryConfig(&odigosv1.CollectorsGroup{}, false, "odigos-system", nil, tc.tier)

			_, defined := cfg.Receivers[odigosEbpfReceiverName]
			assert.Equal(t, tc.wantDefined, defined,
				"receivers map: %q presence should follow tier", odigosEbpfReceiverName)
		})
	}
}

// The shared commonReceivers map is package-level state. Gating must not mutate it, or the first
// enterprise-tier call would leak the receiver into every subsequent community-tier config.
func TestCommonApplicationTelemetryConfig_DoesNotMutateSharedReceivers(t *testing.T) {
	onprem := CommonApplicationTelemetryConfig(&odigosv1.CollectorsGroup{}, false, "odigos-system", nil, common.OnPremOdigosTier)
	_, defined := onprem.Receivers[odigosEbpfReceiverName]
	require.True(t, defined, "precondition: onprem should define the receiver")

	community := CommonApplicationTelemetryConfig(&odigosv1.CollectorsGroup{}, false, "odigos-system", nil, common.CommunityOdigosTier)
	_, leaked := community.Receivers[odigosEbpfReceiverName]
	assert.False(t, leaked, "community config must not inherit the receiver from a prior enterprise-tier call")

	_, leakedIntoPackageState := commonReceivers[odigosEbpfReceiverName]
	assert.False(t, leakedIntoPackageState, "package-level commonReceivers must stay free of the enterprise receiver")
}

func TestMetricsConfig_EbpfReceiverPipelineGatedByTier(t *testing.T) {
	for _, tc := range []struct {
		name      string
		tier      common.OdigosTier
		wantWired bool
	}{
		{"community omits the receiver from the metrics pipeline", common.CommunityOdigosTier, false},
		{"onprem wires the receiver into the metrics pipeline", common.OnPremOdigosTier, true},
		{"cloud wires the receiver into the metrics pipeline", common.CloudOdigosTier, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// kubeletstats keeps the pipeline alive at every tier, so the assertion below is
			// about the eBPF receiver only and not about the pipeline being dropped.
			cfg := MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{
				CommonSignalConfig: CommonSignalConfig{Tier: tc.tier},
				MetricsConfigSettings: &odigosv1.CollectorsGroupMetricsCollectionSettings{
					KubeletStats: &common.MetricsSourceKubeletStatsConfiguration{},
				},
			})

			pl, ok := cfg.Service.Pipelines[odigosMetricsPipelineName]
			require.True(t, ok, "expected a metrics pipeline for tier %q", tc.tier)
			require.True(t, contains(pl.Receivers, kubeletstatsReceiverName),
				"precondition: kubeletstats should be wired regardless of tier")
			assert.Equal(t, tc.wantWired, contains(pl.Receivers, odigosEbpfReceiverName),
				"metrics pipeline receivers %v", pl.Receivers)
		})
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
