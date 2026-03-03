package odigosconfigk8sextension

import (
	"errors"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/odigos-io/odigos/common/consts"
)

// k8SArgoRolloutNameAttribute is the attribute key for Argo Rollout name (no semconv key).
const k8SArgoRolloutNameAttribute = "k8s.argoproj.rollout.name"

// attrKindPairs defines the order in which workload attributes are checked.
// The first matching attribute supplies Name and Kind for the WorkloadKey.
var attrKindPairs = map[string]string{
	string(semconv.K8SDeploymentNameKey):  "Deployment",
	string(semconv.K8SStatefulSetNameKey): "StatefulSet",
	string(semconv.K8SDaemonSetNameKey):   "DaemonSet",
	string(semconv.K8SJobNameKey):         "Job",
	string(semconv.K8SCronJobNameKey):     "CronJob",
	k8SArgoRolloutNameAttribute:           "Rollout",
}

// workloadKeyFromResourceAttributes returns a key from OpenTelemetry resource
// attributes when available. It reads k8s.namespace.name and the first present
// workload name attribute (e.g. k8s.deployment.name, k8s.statefulset.name) to set
func workloadKeyFromResourceAttributes(attrs []attribute.KeyValue) (string, error) {

	ns := getNamespace(attrs)
	kind, name := getKindAndName(attrs)
	containerName := getContainerName(attrs)

	// if the workload info cannot be calculated from the resource attributes, return an empty string.

	if ns == "" || kind == "" || name == "" || containerName == "" {
		return "", errors.New("workload info cannot be calculated from the resource attributes")
	}
	return k8sSourceKey(ns, kind, name, containerName), nil
}

func getNamespace(attrs []attribute.KeyValue) string {
	for _, attr := range attrs {
		if string(attr.Key) == string(semconv.K8SNamespaceNameKey) {
			return attr.Value.AsString()
		}
	}
	return ""
}
func getKindAndName(attrs []attribute.KeyValue) (string, string) {
	for _, attr := range attrs {
		k := string(attr.Key)
		if kind, found := attrKindPairs[k]; found {
			return kind, attr.Value.AsString()
		}
	}

	// Fallback to Odigos-specific workload attributes when no k8s workload attribute matched.
	var kind, name string
	for _, attr := range attrs {
		if string(attr.Key) == consts.OdigosWorkloadNameAttribute {
			name = attr.Value.AsString()
		} else if string(attr.Key) == consts.OdigosWorkloadKindAttribute {
			kind = attr.Value.AsString()
		}
	}
	return kind, name
}

func getContainerName(attrs []attribute.KeyValue) string {
	for _, attr := range attrs {
		if string(attr.Key) == string(semconv.K8SContainerNameKey) {
			return attr.Value.AsString()
		}
	}
	return ""
}
