package collectorconfig

import (
	"testing"

	"github.com/odigos-io/odigos/common/config"
)

func TestAddKubeletStatsToOwnMetrics_ReusesDestinationReceiver(t *testing.T) {
	got := AddKubeletStatsToOwnMetrics(OdigletMetricsConfig("odigos-system"), "odigos-system")

	if _, exists := got.Receivers[kubeletstatsReceiverName]; exists {
		t.Fatal("must not define kubeletstats in own-metrics domain")
	}
	if _, exists := got.Processors["k8sattributes/own-kubelet"]; exists {
		t.Fatal("must not use k8sattributes/own-kubelet — filter on container names only")
	}
	if _, exists := got.Processors[ownKubeletFilterName]; !exists {
		t.Fatal("missing filter/own-kubelet")
	}
	pipeline, exists := got.Service.Pipelines[ownKubeletMetricsPipeline]
	if !exists {
		t.Fatal("missing metrics/own-kubelet pipeline")
	}
	if !contains(pipeline.Receivers, kubeletstatsReceiverName) {
		t.Fatal("own-kubelet pipeline must reuse destination kubeletstats receiver")
	}
	if len(pipeline.Processors) != 1 || pipeline.Processors[0] != ownKubeletFilterName {
		t.Fatalf("processors want [%s], got %v", ownKubeletFilterName, pipeline.Processors)
	}
}

func TestOwnMetricsKubeletFilterScopesToTheOdigosNamespace(t *testing.T) {
	// Odigos can be installed in any namespace. A hardcoded one would either drop the component
	// metrics this pipeline exists for, or forward cpu/memory for every container in the cluster.
	got := ownMetricsKubeletProcessorConfig("custom-odigos-ns")

	filter, ok := got[ownKubeletFilterName].(config.GenericMap)
	if !ok {
		t.Fatal("missing filter/own-kubelet")
	}
	metrics, ok := filter["metrics"].(config.GenericMap)
	if !ok {
		t.Fatal("filter has no metrics conditions")
	}
	conditions, ok := metrics["metric"].([]string)
	if !ok {
		t.Fatal("filter has no metric conditions")
	}

	var scoped bool
	for _, condition := range conditions {
		if condition == `resource.attributes["k8s.namespace.name"] != "custom-odigos-ns"` {
			scoped = true
		}
	}
	if !scoped {
		t.Errorf("no condition drops metrics from outside the odigos namespace, conditions: %v", conditions)
	}
}

func TestAddKubeletStatsToOwnMetrics_EmptyConfig(t *testing.T) {
	got := AddKubeletStatsToOwnMetrics(config.Config{}, "odigos-system")
	if _, exists := got.Receivers[kubeletstatsReceiverName]; exists {
		t.Fatal("must not define kubeletstats")
	}
	if got.Service.Pipelines[ownKubeletMetricsPipeline].Receivers[0] != kubeletstatsReceiverName {
		t.Fatal("pipeline must still reference destination kubeletstats")
	}
}
