package collectorconfig

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

const (
	profilesPipelineName = "profiles"
	// k8sattributesProfilesProcessorName is a dedicated instance for the profiles pipeline (pod association for eBPF/OTLP profiles).
	k8sattributesProfilesProcessorName = "k8sattributes/profiles"
	// resourceProfilesServiceNameProcessorName sets service.name from workload name when not already set (e.g. from pod labels).
	resourceProfilesServiceNameProcessorName = "resource/profiles-service-name"
)

// ProfilesConfig returns the config domain for the profiles pipeline.
// Profiles are received from: (1) the in-node "profiling" receiver (eBPF, all processes), and (2) OTLP in (e.g. external eBPF profiler).
// k8sattributes enriches with pod/namespace/deployment for the current node.
// Pod association uses container.id first: raw eBPF samples only have container.id (from cgroup). The k8sattributes processor
// (contrib) already normalizes container IDs when building the pod cache: it strips the runtime prefix (e.g. "containerd://",
// "cri-o://") from pod status before keying ByID (see internal/kube/client.go). So raw container.id from the profiler matches
// without appending any runtime-specific prefix; works for all runtimes (containerd, cri-o, docker).
// Fallback: connection for samples without container.id (e.g. host processes).
func ProfilesConfig(nodeCG *odigosv1.CollectorsGroup) config.Config {
	processors := config.GenericMap{
		k8sattributesProfilesProcessorName: config.GenericMap{
			"auth_type":   "serviceAccount",
			"passthrough": false,
			// Pod association: one association per source. Try container.id first (eBPF), then connection (fallback).
			"pod_association": []config.GenericMap{
				{"sources": []config.GenericMap{{"from": "resource_attribute", "name": "container.id"}}},
				{"sources": []config.GenericMap{{"from": "connection"}}},
			},
			"extract": config.GenericMap{
				// Include container.* in metadata so the processor populates the container ID index (ByID),
				// required for pod_association by container.id to work (contrib #43689).
				"metadata": []string{
					"k8s.pod.name",
					"k8s.namespace.name",
					"k8s.deployment.name",
					"k8s.statefulset.name",
					"k8s.daemonset.name",
					"k8s.node.name",
					"k8s.container.name",
					"container.id",
				},
				// service.name from pod labels when present (app.kubernetes.io/name preferred, then app).
				"labels": []config.GenericMap{
					{"tag_name": "service.name", "key": "app.kubernetes.io/name"},
					{"tag_name": "service.name", "key": "app"},
				},
			},
			"filter": config.GenericMap{
				"node_from_env_var": "NODE_NAME",
			},
		},
		// Set service.name from workload name when not already set by k8sattributes (e.g. from labels).
		resourceProfilesServiceNameProcessorName: config.GenericMap{
			"attributes": []config.GenericMap{
				{"key": "service.name", "from_attribute": "k8s.deployment.name", "action": "insert"},
				{"key": "service.name", "from_attribute": "k8s.statefulset.name", "action": "insert"},
				{"key": "service.name", "from_attribute": "k8s.daemonset.name", "action": "insert"},
			},
		},
	}

	return config.Config{
		Processors: processors,
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				profilesPipelineName: {
					Receivers:  []string{profilingReceiverName, OTLPInReceiverName},
					// No batch processor: generic batch does not support the profiles signal (same as gateway).
					Processors: []string{memoryLimiterProcessorName, nodeNameProcessorName, resourceDetectionProcessorName, k8sattributesProfilesProcessorName, resourceProfilesServiceNameProcessorName},
					Exporters:  []string{clusterCollectorProfilesExporterName},
				},
			},
		},
	}
}
