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
// CronJob is checked before Job: a pod created by a CronJob carries both
// k8s.cronjob.name and the k8s.job.name of the Job the CronJob created, and only
// the CronJob can be an Odigos Source.
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
	for _, pair := range attrKindPairs {
		if val, ok := attrs.Get(pair.key); ok && val.Type() == pcommon.ValueTypeStr {
			val.CopyTo(identity.PutEmpty(pair.key))
			return identity
		}
	}
	if kind, ok := attrs.Get(consts.OdigosWorkloadKindAttribute); ok {
		kind.CopyTo(identity.PutEmpty(consts.OdigosWorkloadKindAttribute))
	}
	if name, ok := attrs.Get(consts.OdigosWorkloadNameAttribute); ok {
		name.CopyTo(identity.PutEmpty(consts.OdigosWorkloadNameAttribute))
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
	// The Odigos-specific attributes are injected for every Source (by the pods webhook and by
	// odiglet) and name the workload the InstrumentationConfig is named after, so they are
	// preferred: the semconv keys are ambiguous. A DeploymentConfig reuses k8s.deployment.name,
	// and a CronJob's pod also carries the k8s.job.name of the per-run Job.
	if kind, name, ok := getOdigosKindAndName(attrs); ok {
		return kind, name
	}

	for _, pair := range attrKindPairs {
		if val, ok := attrs.Get(pair.key); ok && val.Type() == pcommon.ValueTypeStr {
			return pair.kind, val.Str()
		}
	}

	return "", ""
}

// getOdigosKindAndName reads the odigos.workload.kind / odigos.workload.name pair.
// Both must be present and non-empty for the pair to identify a workload.
func getOdigosKindAndName(attrs pcommon.Map) (string, string, bool) {
	kind, ok := attrs.Get(consts.OdigosWorkloadKindAttribute)
	if !ok || kind.Type() != pcommon.ValueTypeStr || kind.Str() == "" {
		return "", "", false
	}
	name, ok := attrs.Get(consts.OdigosWorkloadNameAttribute)
	if !ok || name.Type() != pcommon.ValueTypeStr || name.Str() == "" {
		return "", "", false
	}
	return kind.Str(), name.Str(), true
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
