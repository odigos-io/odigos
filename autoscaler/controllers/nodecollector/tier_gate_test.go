package nodecollector

import (
	"context"
	"strings"
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The profiling receiver is only compiled into the enterprise collector image, so a community
// collector fails to start on a config that defines it. Profiling can be enabled in the odigos
// configuration at any tier, which makes the tier gate the only thing keeping the profiles
// pipeline out of a community node collector config.

func TestCalculateCollectorConfigDomains_ProfilingPipelineGatedByTier(t *testing.T) {
	profilingEnabled := true

	for _, tc := range []struct {
		name          string
		tier          common.OdigosTier
		profiling     *common.ProfilingConfiguration
		wantProfiling bool
	}{
		{"community omits the profiling pipeline", common.CommunityOdigosTier, &common.ProfilingConfiguration{Enabled: &profilingEnabled}, false},
		{"onprem writes the profiling pipeline", common.OnPremOdigosTier, &common.ProfilingConfiguration{Enabled: &profilingEnabled}, true},
		{"cloud writes the profiling pipeline", common.CloudOdigosTier, &common.ProfilingConfiguration{Enabled: &profilingEnabled}, true},
		{"onprem without profiling config omits the pipeline", common.OnPremOdigosTier, nil, false},
		{"onprem with profiling disabled omits the pipeline", common.OnPremOdigosTier, &common.ProfilingConfiguration{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configDomains, mergedConfigYaml, err := calculateCollectorConfigDomains(
				context.Background(),
				"odigos-system",
				&odigosv1.CollectorsGroup{
					ObjectMeta: metav1.ObjectMeta{Name: "test-collector-group"},
					Spec: odigosv1.CollectorsGroupSpec{
						CollectorOwnMetricsPort: 4317,
					},
				},
				&odigosv1.InstrumentationConfigList{},
				[]common.ObservabilitySignal{common.TracesObservabilitySignal},
				nil,          /* processors */
				false,        /* onGKE */
				false,        /* loadBalancingNeeded */
				tc.profiling, /* profiling */
				tc.tier,      /* tier */
			)

			require.NoError(t, err)

			profilingDomain, present := configDomains["profiling"]
			require.Equal(t, tc.wantProfiling, present, "presence of the profiling config domain should follow the tier")
			if tc.wantProfiling {
				_, receiverDefined := profilingDomain.Receivers[commonconf.ProfilingReceiver]
				assert.True(t, receiverDefined, "expected the profiling receiver to be defined for tier %q", tc.tier)
				assert.Contains(t, profilingDomain.Service.Pipelines, "profiles")
			}

			// the merged config is what the collector actually loads
			assert.Equal(t, tc.wantProfiling, strings.Contains(mergedConfigYaml, commonconf.ProfilingNodeToGatewayExporter),
				"merged collector config should only export profiles for enterprise tiers")

			// the traces pipeline is unaffected by the profiling gate
			assert.Contains(t, configDomains, "traces")
		})
	}
}
