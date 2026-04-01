package common

import (
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

// MergeProfilingOtlpExporter merges Profiling.Exporter into an OTLP exporter config map.
func MergeProfilingOtlpExporter(base config.GenericMap, otlp *odigoscommon.OtlpExporterConfiguration) config.GenericMap {
	out := config.GenericMap{}
	for k, v := range base {
		out[k] = v
	}
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
