package collectorprofiles

import (
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/services/common"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// NormalizeWorkloadKind maps API/UI strings to canonical WorkloadKind values for source keys.
// This is Odigos-specific (not from Pyroscope): GraphQL and resource attributes may use mixed casing
// or synonyms; keys must match k8sconsts and the semconv branches in SourceKeyFromResource.
func NormalizeWorkloadKind(kindStr string) k8sconsts.WorkloadKind {
	switch strings.ToLower(kindStr) {
	case "deployment":
		return k8sconsts.WorkloadKindDeployment
	case "daemonset":
		return k8sconsts.WorkloadKindDaemonSet
	case "statefulset":
		return k8sconsts.WorkloadKindStatefulSet
	case "cronjob":
		return k8sconsts.WorkloadKindCronJob
	case "job":
		return k8sconsts.WorkloadKindJob
	case "deploymentconfig":
		return k8sconsts.WorkloadKindDeploymentConfig
	case "rollout":
		return k8sconsts.WorkloadKindArgoRollout
	case "namespace":
		return k8sconsts.WorkloadKindNamespace
	case "staticpod":
		return k8sconsts.WorkloadKindStaticPod
	default:
		return k8sconsts.WorkloadKind(kindStr)
	}
}

// SourceKeyFromSourceID returns a stable string key for the given SourceID.
// Format: "namespace/kind/name" so it matches keys derived from profile resource attributes.
func SourceKeyFromSourceID(id common.SourceID) string {
	return id.Namespace + "/" + string(id.Kind) + "/" + id.Name
}

// SourceKeyFromResource extracts namespace, kind and name from OTLP resource attributes
// (Kubernetes semconv plus Odigos workload fallbacks). Key format matches SourceKeyFromSourceID.
func SourceKeyFromResource(attrs pcommon.Map) (string, bool) {
	ns, ok := attrs.Get(string(semconv.K8SNamespaceNameKey))
	if !ok || ns.Str() == "" {
		return "", false
	}
	namespace := ns.Str()

	var kind k8sconsts.WorkloadKind
	var name string
	var found bool

	if name, found = getStr(attrs, string(semconv.K8SDeploymentNameKey)); found {
		kind = k8sconsts.WorkloadKindDeployment
	} else if name, found = getStr(attrs, string(semconv.K8SStatefulSetNameKey)); found {
		kind = k8sconsts.WorkloadKindStatefulSet
	} else if name, found = getStr(attrs, string(semconv.K8SDaemonSetNameKey)); found {
		kind = k8sconsts.WorkloadKindDaemonSet
	} else if name, found = getStr(attrs, string(semconv.K8SCronJobNameKey)); found {
		kind = k8sconsts.WorkloadKindCronJob
	} else if name, found = getStr(attrs, string(semconv.K8SJobNameKey)); found {
		kind = k8sconsts.WorkloadKindJob
	} else if name, found = getStr(attrs, k8sconsts.K8SArgoRolloutNameAttribute); found {
		kind = k8sconsts.WorkloadKindArgoRollout
	} else {
		odigosKind, kindFound := getStr(attrs, odigosconsts.OdigosWorkloadKindAttribute)
		odigosName, nameFound := getStr(attrs, odigosconsts.OdigosWorkloadNameAttribute)
		if kindFound && nameFound && odigosName != "" {
			// Odigos attributes are free-form strings; normalize so the key matches SourceKeyFromSourceID.
			kind = NormalizeWorkloadKind(odigosKind)
			name = odigosName
			found = true
		}
	}
	if !found || name == "" {
		return "", false
	}

	return namespace + "/" + string(kind) + "/" + name, true
}

func getStr(attrs pcommon.Map, key string) (string, bool) {
	v, ok := attrs.Get(key)
	if !ok {
		return "", false
	}
	return v.Str(), true
}

// allNamesArePlaceholders reports whether every frame name is synthetic (no resolved symbols).
func allNamesArePlaceholders(names []string) bool {
	for _, n := range names {
		if n == "" || n == "total" || n == "other" {
			continue
		}
		if isSyntheticFrameName(n) {
			continue
		}
		return false
	}
	return true
}

func isSyntheticFrameName(n string) bool {
	if len(n) > 6 && n[:6] == "frame_" {
		return true
	}
	if len(n) > 2 && n[:2] == "0x" {
		return true
	}
	return false
}
