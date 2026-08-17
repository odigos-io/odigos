package collectorconfig

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/config"
)

func TestAddKubeletStatsToOwnMetrics_ReusesDestinationReceiver(t *testing.T) {
	got := AddKubeletStatsToOwnMetrics(OdigletMetricsConfig("odigos-system"), "odigos-system")

	if _, exists := got.Receivers[kubeletstatsReceiverName]; exists {
		t.Fatal("must not define kubeletstats in own-metrics domain")
	}
	if _, exists := got.Processors[ownKubeletK8sAttrName]; !exists {
		t.Fatal("missing k8sattributes/own-kubelet")
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
	if len(pipeline.Processors) < 2 || pipeline.Processors[0] != ownKubeletK8sAttrName || pipeline.Processors[1] != ownKubeletFilterName {
		t.Fatalf("processors want [%s %s], got %v", ownKubeletK8sAttrName, ownKubeletFilterName, pipeline.Processors)
	}
}

// The node collector runs on every node, so an unfiltered k8sattributes informer means a
// cluster-wide pod (and replicaset) LIST/WATCH per node.
func TestAddKubeletStatsToOwnMetrics_InformerIsScoped(t *testing.T) {
	got := AddKubeletStatsToOwnMetrics(OdigletMetricsConfig("odigos-system"), "odigos-system")

	k8sAttr, ok := got.Processors[ownKubeletK8sAttrName].(config.GenericMap)
	if !ok {
		t.Fatalf("%s is not a config map, got %T", ownKubeletK8sAttrName, got.Processors[ownKubeletK8sAttrName])
	}

	filter, ok := k8sAttr["filter"].(config.GenericMap)
	if !ok {
		t.Fatalf("%s must set a filter to scope its informers, got %#v", ownKubeletK8sAttrName, k8sAttr["filter"])
	}
	if filter["namespace"] != "odigos-system" {
		t.Fatalf("filter.namespace want odigos-system, got %#v", filter["namespace"])
	}
	if filter["node_from_env_var"] != k8sconsts.NodeNameEnvVar {
		t.Fatalf("filter.node_from_env_var want %s, got %#v", k8sconsts.NodeNameEnvVar, filter["node_from_env_var"])
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
