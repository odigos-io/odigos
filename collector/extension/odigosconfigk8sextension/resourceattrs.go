package odigosconfigk8sextension

import (
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/odigos-io/odigos/common/consts"
)

// K8SArgoRolloutNameAttribute is the attribute key for Argo Rollout name (no semconv key).
const K8SArgoRolloutNameAttribute = "k8s.argoproj.rollout.name"

// attrKindPairs defines the order in which workload attributes are checked.
// The first matching attribute supplies Name and Kind for the WorkloadKey.
var attrKindPairs = []struct {
	attr string
	kind string
}{
	{string(semconv.K8SDeploymentNameKey), "Deployment"},
	{string(semconv.K8SStatefulSetNameKey), "StatefulSet"},
	{string(semconv.K8SDaemonSetNameKey), "DaemonSet"},
	{string(semconv.K8SJobNameKey), "Job"},
	{string(semconv.K8SCronJobNameKey), "CronJob"},
	{K8SArgoRolloutNameAttribute, "Rollout"},
}

// WorkloadKeyFromResourceAttributes returns a key from OpenTelemetry resource
// attributes when available. It reads k8s.namespace.name and the first present
// workload name attribute (e.g. k8s.deployment.name, k8s.statefulset.name) to set
func WorkloadKeyFromResourceAttributes(attrs pcommon.Map) (string, error) {
	ns := getNamespace(attrs)
	kind, name := getKindAndName(attrs)
	containerName := getContainerName(attrs)

	// if the workload info cannot be calculated from the resource attributes, return an empty string.

	if ns == "" || kind == "" || name == "" || containerName == "" {
		return "", errors.New("workload info cannot be calculated from the resource attributes")
	}
	return K8sSourceKey(ns, kind, name, containerName), nil
}

func getNamespace(attrs pcommon.Map) string {
	if v, ok := attrs.Get(string(semconv.K8SNamespaceNameKey)); ok && v.Type() == pcommon.ValueTypeStr {
		return v.Str()
	}
	return ""
}

func getKindAndName(attrs pcommon.Map) (string, string) {
	for _, p := range attrKindPairs {
		if v, ok := attrs.Get(p.attr); ok && v.Type() == pcommon.ValueTypeStr && v.Str() != "" {
			return p.kind, v.Str()
		}
	}

	// Fallback to Odigos-specific workload attributes when no k8s workload attribute matched.
	if nameVal, ok := attrs.Get(consts.OdigosWorkloadNameAttribute); ok && nameVal.Type() == pcommon.ValueTypeStr && nameVal.Str() != "" {
		name := nameVal.Str()
		if kindVal, ok := attrs.Get(consts.OdigosWorkloadKindAttribute); ok && kindVal.Type() == pcommon.ValueTypeStr && kindVal.Str() != "" {
			kind := kindVal.Str()
			return kind, name
		}
	}

	return "", ""
}

func getContainerName(attrs pcommon.Map) string {
	if v, ok := attrs.Get(string(semconv.K8SContainerNameKey)); ok && v.Type() == pcommon.ValueTypeStr {
		return v.Str()
	}
	return ""
}
