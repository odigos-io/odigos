package nodecollector

import (
	"context"
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Component names are spelled out instead of referencing the collectorconfig constants, so that
// renaming a constant cannot make these assertions follow along silently.
const (
	odigletMetricsDomain      = "odiglet_metrics"
	kubeletStatsReceiver      = "kubeletstats"
	ownKubeletPipelineName    = "metrics/own-kubelet"
	odigletMetricsPipeline    = "metrics/odiglet-metrics"
	ownKubeletK8sAttrsName    = "k8sattributes/own-kubelet"
	ownKubeletFilterProcessor = "filter/own-kubelet"
)

func nodeCollectorsGroupWithOwnMetrics(ownMetrics *odigosv1.OdigosOwnMetricsSettings, kubeletStats *common.MetricsSourceKubeletStatsConfiguration) *odigosv1.CollectorsGroup {
	return &odigosv1.CollectorsGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "test-collector-group"},
		Spec: odigosv1.CollectorsGroupSpec{
			CollectorOwnMetricsPort: 4317,
			Metrics: &odigosv1.CollectorsGroupMetricsCollectionSettings{
				KubeletStats:     kubeletStats,
				OdigosOwnMetrics: ownMetrics,
			},
		},
	}
}

func nodeCollectorConfigDomains(t *testing.T, nodeCG *odigosv1.CollectorsGroup, clusterCollectorSignals []common.ObservabilitySignal) map[string]config.Config {
	t.Helper()
	domains, _, err := calculateCollectorConfigDomains(
		context.Background(),
		"odigos-system",
		nodeCG,
		&odigosv1.InstrumentationConfigList{},
		clusterCollectorSignals,
		nil,   /* processors */
		false, /* onGKE */
		false, /* loadBalancingNeeded */
		nil,   /* profiling */
		common.OnPremOdigosTier,
	)
	require.NoError(t, err)
	return domains
}

// assertNoDanglingComponentReferences checks the invariant that makes the kubeletstats gate matter:
// the collector refuses to start when a pipeline names a component no config domain defined, which
// would take down the node collector on every node in the cluster.
func assertNoDanglingComponentReferences(t *testing.T, domains map[string]config.Config) {
	t.Helper()
	merged, err := config.MergeConfigs(domains)
	require.NoError(t, err)

	defined := func(names ...map[string]any) func(string) bool {
		return func(name string) bool {
			for _, set := range names {
				if _, ok := set[name]; ok {
					return true
				}
			}
			return false
		}
	}
	// Connectors act as an exporter of one pipeline and a receiver of another.
	isReceiver := defined(merged.Receivers, merged.Connectors)
	isProcessor := defined(merged.Processors)
	isExporter := defined(merged.Exporters, merged.Connectors)

	for name, pipeline := range merged.Service.Pipelines {
		for _, receiver := range pipeline.Receivers {
			assert.True(t, isReceiver(receiver), "pipeline %q receives from undefined component %q", name, receiver)
		}
		for _, processor := range pipeline.Processors {
			assert.True(t, isProcessor(processor), "pipeline %q uses undefined processor %q", name, processor)
		}
		for _, exporter := range pipeline.Exporters {
			assert.True(t, isExporter(exporter), "pipeline %q exports to undefined component %q", name, exporter)
		}
	}
}

// The own-metrics kubelet pipeline reuses the kubeletstats receiver that the metrics destination
// defines, so it may only be added when that receiver is actually there.
func TestOdigletMetricsKubeletStatsGate(t *testing.T) {
	allSignals := []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.MetricsObservabilitySignal,
		common.LogsObservabilitySignal,
	}
	tracesOnly := []common.ObservabilitySignal{common.TracesObservabilitySignal}
	kubeletStats := &common.MetricsSourceKubeletStatsConfiguration{Interval: "44s"}
	ownMetrics := &odigosv1.OdigosOwnMetricsSettings{SendToOdigosMetricsStore: true, Interval: "10s"}

	tests := []struct {
		name                string
		ownMetrics          *odigosv1.OdigosOwnMetricsSettings
		kubeletStats        *common.MetricsSourceKubeletStatsConfiguration
		signals             []common.ObservabilitySignal
		wantOdigletDomain   bool
		wantKubeletPipeline bool
	}{
		{
			name:                "kubelet stats collected for a metrics destination is reused",
			ownMetrics:          ownMetrics,
			kubeletStats:        kubeletStats,
			signals:             allSignals,
			wantOdigletDomain:   true,
			wantKubeletPipeline: true,
		},
		{
			name:              "no kubelet stats to reuse",
			ownMetrics:        ownMetrics,
			signals:           allSignals,
			wantOdigletDomain: true,
		},
		{
			name: "metrics destination disabled, so the kubeletstats receiver is never defined",
			// The only case that produces an unstartable collector if the gate is dropped.
			ownMetrics:        ownMetrics,
			kubeletStats:      kubeletStats,
			signals:           tracesOnly,
			wantOdigletDomain: true,
		},
		{
			name:         "own metrics disabled",
			kubeletStats: kubeletStats,
			signals:      allSignals,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domains := nodeCollectorConfigDomains(t, nodeCollectorsGroupWithOwnMetrics(tt.ownMetrics, tt.kubeletStats), tt.signals)

			odigletMetrics, hasOdigletDomain := domains[odigletMetricsDomain]
			require.Equal(t, tt.wantOdigletDomain, hasOdigletDomain, "presence of the %q config domain", odigletMetricsDomain)

			if hasOdigletDomain {
				// The odiglet scrape itself is not conditional on the metrics destination.
				assert.Contains(t, odigletMetrics.Service.Pipelines, odigletMetricsPipeline)

				pipeline, hasKubeletPipeline := odigletMetrics.Service.Pipelines[ownKubeletPipelineName]
				require.Equal(t, tt.wantKubeletPipeline, hasKubeletPipeline, "presence of the %q pipeline", ownKubeletPipelineName)
				if hasKubeletPipeline {
					assert.Equal(t, []string{kubeletStatsReceiver}, pipeline.Receivers)
					// k8sattributes has to run first: kubeletstats emits no workload owner
					// attributes, and the filter selects on them.
					assert.Equal(t, []string{ownKubeletK8sAttrsName, ownKubeletFilterProcessor}, pipeline.Processors)
					assert.Contains(t, odigletMetrics.Processors, ownKubeletK8sAttrsName)
					assert.Contains(t, odigletMetrics.Processors, ownKubeletFilterProcessor)
					// Own metrics must never start a kubelet scrape of its own.
					assert.NotContains(t, odigletMetrics.Receivers, kubeletStatsReceiver,
						"the own-metrics domain must reuse the metrics destination receiver, not define its own")
				} else {
					assert.NotContains(t, odigletMetrics.Processors, ownKubeletFilterProcessor)
				}
			}

			assertNoDanglingComponentReferences(t, domains)
		})
	}
}
