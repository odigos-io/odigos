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

func workloadIdentityFromResourceAttributes(attrs pcommon.Map) (string, pcommon.Map, error) {
	cacheKey, err := workloadKeyFromResourceAttributes(attrs)
	if err != nil {
		return "", pcommon.NewMap(), err
	}
	return cacheKey, identifyingResourceAttributes(attrs), nil
}

func identifyingResourceAttributes(attrs pcommon.Map) pcommon.Map {
	identity := pcommon.NewMap()
	if ns := getNamespace(attrs); ns != "" {
		identity.PutStr(string(semconv.K8SNamespaceNameKey), ns)
	}
	if containerName := getContainerName(attrs); containerName != "" {
		identity.PutStr(string(semconv.K8SContainerNameKey), containerName)
	}

	kind, kindOk := attrs.Get(consts.OdigosWorkloadKindAttribute)
	name, nameOk := attrs.Get(consts.OdigosWorkloadNameAttribute)
	if kindOk && nameOk {
		kind.CopyTo(identity.PutEmpty(consts.OdigosWorkloadKindAttribute))
		name.CopyTo(identity.PutEmpty(consts.OdigosWorkloadNameAttribute))
		return identity
	}

	for _, pair := range attrKindPairs {
		if val, ok := attrs.Get(pair.key); ok && val.Type() == pcommon.ValueTypeStr {
			val.CopyTo(identity.PutEmpty(pair.key))
			return identity
		}
	}
	return identity
}

func getNamespace(attrs pcommon.Map) string {
	ns, ok := attrs.Get(string(semconv.K8SNamespaceNameKey))
	if !ok {
		return ""
	}
	return ns.Str()
}

func getKindAndName(attrs pcommon.Map) (string, string) {

	// Prefer odigos-specific attributes if exists (to use openshift DeploymentConfig if exists instead of Deployment)
	kind, kindOk := attrs.Get(consts.OdigosWorkloadKindAttribute)
	name, nameOk := attrs.Get(consts.OdigosWorkloadNameAttribute)
	if kindOk && nameOk {
		return kind.Str(), name.Str()
	}

	for _, pair := range attrKindPairs {
		if val, ok := attrs.Get(pair.key); ok && val.Type() == pcommon.ValueTypeStr {
			return pair.kind, val.Str()
		}
	}

	return "", ""
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
