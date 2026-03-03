package odigosconfigk8sextension

import (
	"errors"

	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/odigos-io/odigos/common/consts"
)

// k8SArgoRolloutNameAttribute is the attribute key for Argo Rollout name (no semconv key).
const k8SArgoRolloutNameAttribute = "k8s.argoproj.rollout.name"

// attrKindPairs defines the order in which workload attributes are checked.
// The first matching attribute supplies Name and Kind for the WorkloadKey.
var attrKindPairs = []struct {
	key  string
	kind string
}{
	{key: string(semconv.K8SDeploymentNameKey), kind: "Deployment"},
	{key: string(semconv.K8SStatefulSetNameKey), kind: "StatefulSet"},
	{key: string(semconv.K8SDaemonSetNameKey), kind: "DaemonSet"},
	{key: string(semconv.K8SJobNameKey), kind: "Job"},
	{key: string(semconv.K8SCronJobNameKey), kind: "CronJob"},
	{key: k8SArgoRolloutNameAttribute, kind: "Rollout"},
}

// workloadKeyFromResourceAttributes returns a key from OpenTelemetry resource
// attributes when available. It reads k8s.namespace.name and the first present
// workload name attribute (e.g. k8s.deployment.name, k8s.statefulset.name) to set
func workloadKeyFromResourceAttributes(attrs pcommon.Map) (string, error) {

	ns := getNamespace(attrs)
	kind, name := getKindAndName(attrs)
	containerName := getContainerName(attrs)

	// if the workload info cannot be calculated from the resource attributes, return an empty string.

	if ns == "" || kind == "" || name == "" || containerName == "" {
		return "", errors.New("workload info cannot be calculated from the resource attributes")
	}
	return k8sSourceKey(ns, kind, name, containerName), nil
}

func getNamespace(attrs pcommon.Map) string {
	ns, ok := attrs.Get(string(semconv.K8SNamespaceNameKey))
	if !ok {
		return ""
	}
	return ns.Str()
}

func getKindAndName(attrs pcommon.Map) (string, string) {

	for _, pair := range attrKindPairs {
		if val, ok := attrs.Get(pair.key); ok && val.Type() == pcommon.ValueTypeStr {
			return pair.kind, val.Str()
		}
	}

	// Fallback to Odigos-specific workload attributes when no k8s workload attribute matched.
	kind, ok := attrs.Get(consts.OdigosWorkloadKindAttribute)
	if !ok {
		return "", ""
	}
	name, ok := attrs.Get(consts.OdigosWorkloadNameAttribute)
	if !ok {
		return "", ""
	}
	return kind.Str(), name.Str()
}

func getContainerName(attrs pcommon.Map) string {
	containerName, ok := attrs.Get(string(semconv.K8SContainerNameKey))
	if !ok {
		return ""
	}
	return containerName.Str()
}
