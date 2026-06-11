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
	{key: string(semconv.K8SCronJobNameKey), kind: "CronJob"},
	{key: string(semconv.K8SJobNameKey), kind: "Job"},
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
	if kind, ok := getStringAttr(attrs, consts.OdigosWorkloadKindAttribute); ok {
		if name, ok := getStringAttr(attrs, consts.OdigosWorkloadNameAttribute); ok {
			return kind, name
		}

		if nameAttr := workloadNameAttrForKind(kind); nameAttr != "" {
			if name, ok := getStringAttr(attrs, nameAttr); ok {
				return kind, name
			}
		}

		return "", ""
	}

	for _, pair := range attrKindPairs {
		if name, ok := getStringAttr(attrs, pair.key); ok {
			return pair.kind, name
		}
	}

	return "", ""
}

func getStringAttr(attrs pcommon.Map, key string) (string, bool) {
	val, ok := attrs.Get(key)
	if !ok || val.Type() != pcommon.ValueTypeStr {
		return "", false
	}
	return val.Str(), true
}

func workloadNameAttrForKind(kind string) string {
	switch kind {
	case "Deployment":
		return string(semconv.K8SDeploymentNameKey)
	case "StatefulSet":
		return string(semconv.K8SStatefulSetNameKey)
	case "DaemonSet":
		return string(semconv.K8SDaemonSetNameKey)
	case "CronJob":
		return string(semconv.K8SCronJobNameKey)
	case "Job":
		return string(semconv.K8SJobNameKey)
	case "DeploymentConfig":
		return string(semconv.K8SDeploymentNameKey)
	case "Rollout":
		return k8SArgoRolloutNameAttribute
	default:
		return ""
	}
}

func getContainerName(attrs pcommon.Map) string {
	containerName, ok := attrs.Get(string(semconv.K8SContainerNameKey))
	if !ok {
		return ""
	}
	return containerName.Str()
}

// workloadContainerKeyFromResourceAttributes builds a workload-level cache key prefix
// from resource attributes. Unlike workloadKeyFromResourceAttributes, it does not
// require k8s.container.name. Returns the key in "ns/Kind/name/" format.
func workloadContainerKeyFromResourceAttributes(attrs pcommon.Map) (string, error) {
	ns := getNamespace(attrs)
	kind, name := getKindAndName(attrs)
	if ns == "" || kind == "" || name == "" {
		return "", errors.New("workload info cannot be calculated from the resource attributes")
	}
	return WorkloadKeyString(ns, kind, name) + "/", nil
}
