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

// Each call must build its own receivers map, or an enterprise-tier config would leak the receiver
// into every community-tier config built after it.
func TestCommonApplicationTelemetryConfig_ReceiversAreNotSharedBetweenCalls(t *testing.T) {
	onprem := CommonApplicationTelemetryConfig(&odigosv1.CollectorsGroup{}, false, "odigos-system", nil, common.OnPremOdigosTier)
	_, defined := onprem.Receivers[odigosEbpfReceiverName]
	require.True(t, defined, "precondition: onprem should define the receiver")

	community := CommonApplicationTelemetryConfig(&odigosv1.CollectorsGroup{}, false, "odigos-system", nil, common.CommunityOdigosTier)
	_, leaked := community.Receivers[odigosEbpfReceiverName]
	assert.False(t, leaked, "community config must not inherit the receiver from a prior enterprise-tier call")
}

// The eBPF receiver carries traces and logs only. JVM runtime metrics reach the node collector
// over OTLP, so no tier wires the receiver into a metrics pipeline.
func TestMetricsConfig_EbpfReceiverNotInMetricsPipelines(t *testing.T) {
	for _, tier := range []common.OdigosTier{common.CommunityOdigosTier, common.OnPremOdigosTier, common.CloudOdigosTier} {
		t.Run(string(tier), func(t *testing.T) {
			// kubeletstats keeps the pipeline alive at every tier, so the assertion below is
			// about the eBPF receiver only and not about the pipeline being dropped.
			cfg := MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{
				CommonSignalConfig: CommonSignalConfig{Tier: tier},
				MetricsConfigSettings: &odigosv1.CollectorsGroupMetricsCollectionSettings{
					KubeletStats: &common.MetricsSourceKubeletStatsConfiguration{},
				},
			})

			pl, ok := cfg.Service.Pipelines[odigosMetricsPipelineName]
			require.True(t, ok, "expected a metrics pipeline for tier %q", tier)
			require.True(t, contains(pl.Receivers, kubeletstatsReceiverName),
				"precondition: kubeletstats should be wired regardless of tier")
			for name, pl := range cfg.Service.Pipelines {
				assert.False(t, contains(pl.Receivers, odigosEbpfReceiverName),
					"pipeline %s receivers %v", name, pl.Receivers)
			}
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

func TestCommonConfig_AuthExtensionGatedByTier(t *testing.T) {
	for _, tc := range []struct {
		name     string
		tier     common.OdigosTier
		wantAuth bool
	}{
		{"community omits the enterprise auth extension", common.CommunityOdigosTier, false},
		{"onprem enables the enterprise auth extension", common.OnPremOdigosTier, true},
		{"cloud enables the enterprise auth extension", common.CloudOdigosTier, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := CommonConfig(tc.tier)

			_, defined := cfg.Extensions[odigosEnterpriseAuthExtensionName]
			assert.Equal(t, tc.wantAuth, defined, "extensions map")
			assert.Equal(t, tc.wantAuth, contains(cfg.Service.Extensions, odigosEnterpriseAuthExtensionName),
				"service.extensions %v", cfg.Service.Extensions)

			// health_check and pprof must survive in both tiers.
			assert.True(t, contains(cfg.Service.Extensions, healthCheckExtensionName))
			assert.True(t, contains(cfg.Service.Extensions, pprofExtensionName))
		})
	}
}

// Building an enterprise config must not leak the auth extension into a community one built after it.
func TestCommonConfig_ExtensionsAreNotSharedBetweenCalls(t *testing.T) {
	onprem := CommonConfig(common.OnPremOdigosTier)
	require.True(t, contains(onprem.Service.Extensions, odigosEnterpriseAuthExtensionName))

	community := CommonConfig(common.CommunityOdigosTier)
	assert.False(t, contains(community.Service.Extensions, odigosEnterpriseAuthExtensionName),
		"community service.extensions leaked: %v", community.Service.Extensions)
	_, leaked := community.Extensions[odigosEnterpriseAuthExtensionName]
	assert.False(t, leaked, "community extensions map leaked the auth extension")
}
