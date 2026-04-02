package common

import (
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

// MergeProfilingOtlpExporter merges Profiling.Exporter into an OTLP exporter config map.
//
// The returned map is always a shallow copy of base (even when otlp is nil) so callers can
// safely mutate the result without affecting the map they passed in.
//
// We map fields explicitly rather than json.Marshal/Unmarshal: OdigosConfiguration uses camelCase
// JSON tags (e.g. retryOnFailure, initialInterval) while the OpenTelemetry Collector exporter
// block expects snake_case keys (retry_on_failure, initial_interval, etc.).
func MergeProfilingOtlpExporter(base config.GenericMap, otlp *odigoscommon.OtlpExporterConfiguration) config.GenericMap {
	out := cloneGenericMap(base)
	if otlp == nil {
		return out
	}
	if otlp.Timeout != "" {
		out["timeout"] = otlp.Timeout
	}
	if otlp.RetryOnFailure != nil {
		retry := config.GenericMap{}
		if otlp.RetryOnFailure.Enabled != nil {
			retry["enabled"] = *otlp.RetryOnFailure.Enabled
		} else {
			retry["enabled"] = true
		}
		if otlp.RetryOnFailure.InitialInterval != "" {
			retry["initial_interval"] = otlp.RetryOnFailure.InitialInterval
		}
		if otlp.RetryOnFailure.MaxInterval != "" {
			retry["max_interval"] = otlp.RetryOnFailure.MaxInterval
		}
		if otlp.RetryOnFailure.MaxElapsedTime != "" {
			retry["max_elapsed_time"] = otlp.RetryOnFailure.MaxElapsedTime
		}
		out["retry_on_failure"] = retry
	}
	return out
}

func cloneGenericMap(m config.GenericMap) config.GenericMap {
	if len(m) == 0 {
		return config.GenericMap{}
	}
	out := make(config.GenericMap, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// K8sAttributesProfilesProcessorConfig is the k8sattributes processor config for profiles pipelines.
func K8sAttributesProfilesProcessorConfig() config.GenericMap {
	return config.GenericMap{
		"auth_type":   "serviceAccount",
		"passthrough": false,
		"extract": config.GenericMap{
			"metadata": []string{
				"k8s.namespace.name",
				"k8s.pod.name",
				"k8s.pod.uid",
				"k8s.deployment.name",
				"k8s.statefulset.name",
				"k8s.daemonset.name",
				"container.id",
			},
		},
		// Primary association by container.id (CRI/container runtime id on profile resource).
		// k8s.pod.ip is a secondary path for cases where container id is missing or the processor needs IP-based correlation.
		"pod_association": []config.GenericMap{
			{
				"sources": []config.GenericMap{
					{"from": "resource_attribute", "name": "container.id"},
				},
			},
			{
				"sources": []config.GenericMap{
					{"from": "resource_attribute", "name": "k8s.pod.ip"},
				},
			},
		},
	}
}

// ProfilingProfileDropConditions returns filterprocessor profile_conditions used on node and gateway
// profiles pipelines (drop rows without container id before k8s_attributes).
func ProfilingProfileDropConditions() []string {
	return []string{
		`resource.attributes["container.id"] == nil`,
	}
}

// ProfilingFilterProcessorConfig is the filter processor block for profiles (contrib filterprocessor).
func ProfilingFilterProcessorConfig() config.GenericMap {
	return config.GenericMap{
		"error_mode":         "ignore",
		"profile_conditions": ProfilingProfileDropConditions(),
	}
}
