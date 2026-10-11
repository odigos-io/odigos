package common

import (
	"errors"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// ResourceAttributesToSourceID resolves namespace, workload kind and name from OTLP resource
func ResourceAttributesToSourceID(attrs pcommon.Map) (SourceID, error) {
	ns, ok := attrs.Get(string(semconv.K8SNamespaceNameKey))
	if !ok {
		return SourceID{}, errors.New("namespace not found")
	}

	var workloadKind k8sconsts.WorkloadKind
	var workloadName pcommon.Value
	var workloadFound bool

	if odigosWorkloadName, ok := attrs.Get(odigosconsts.OdigosWorkloadNameAttribute); ok {
		workloadName, workloadFound = odigosWorkloadName, true
	}

	if odigosKind, ok := attrs.Get(odigosconsts.OdigosWorkloadKindAttribute); ok {
		workloadKind = workloadKindFromMixedCaseString(odigosKind.Str())

		if !workloadFound {
			// odigos.workload.kind is present but odigos.workload.name is absent;
			// fall back to the standard semconv attribute for the resolved kind.
			switch workloadKind {
			case k8sconsts.WorkloadKindDeployment:
				workloadName, workloadFound = attrs.Get(string(semconv.K8SDeploymentNameKey))
			case k8sconsts.WorkloadKindStatefulSet:
				workloadName, workloadFound = attrs.Get(string(semconv.K8SStatefulSetNameKey))
			case k8sconsts.WorkloadKindDaemonSet:
				workloadName, workloadFound = attrs.Get(string(semconv.K8SDaemonSetNameKey))
			case k8sconsts.WorkloadKindCronJob:
				workloadName, workloadFound = attrs.Get(string(semconv.K8SCronJobNameKey))
			case k8sconsts.WorkloadKindJob:
				workloadName, workloadFound = attrs.Get(string(semconv.K8SJobNameKey))
			case k8sconsts.WorkloadKindStaticPod:
				// StaticPod names match the pod name (virtual node convention).
				workloadName, workloadFound = attrs.Get(string(semconv.K8SPodNameKey))
			case k8sconsts.WorkloadKindArgoRollout:
				workloadName, workloadFound = attrs.Get(k8sconsts.K8SArgoRolloutNameAttribute)
			case k8sconsts.WorkloadKindDeploymentConfig:
				// OpenShift DeploymentConfig reuses k8s.deployment.name (same key as Deployment).
				workloadName, workloadFound = attrs.Get(string(semconv.K8SDeploymentNameKey))
			}
		}

		if !workloadFound {
			return SourceID{}, errors.New("workload name not found")
		}
	} else {
		// No odigos.workload.kind: infer kind from whichever standard semconv attribute is present.
		if depName, ok := attrs.Get(string(semconv.K8SDeploymentNameKey)); ok {
			workloadKind = k8sconsts.WorkloadKindDeployment
			workloadName = depName
		} else if ssName, ok := attrs.Get(string(semconv.K8SStatefulSetNameKey)); ok {
			workloadKind = k8sconsts.WorkloadKindStatefulSet
			workloadName = ssName
		} else if dsName, ok := attrs.Get(string(semconv.K8SDaemonSetNameKey)); ok {
			workloadKind = k8sconsts.WorkloadKindDaemonSet
			workloadName = dsName
		} else if cjName, ok := attrs.Get(string(semconv.K8SCronJobNameKey)); ok {
			workloadKind = k8sconsts.WorkloadKindCronJob
			workloadName = cjName
		} else if jobName, ok := attrs.Get(string(semconv.K8SJobNameKey)); ok {
			workloadKind = k8sconsts.WorkloadKindJob
			workloadName = jobName
		} else if rolloutName, ok := attrs.Get(k8sconsts.K8SArgoRolloutNameAttribute); ok {
			workloadKind = k8sconsts.WorkloadKindArgoRollout
			workloadName = rolloutName
		} else {
			// Without odigos.workload.kind, DeploymentConfig is indistinguishable from Deployment
			// (both use k8s.deployment.name). StaticPod needs odigos.workload.kind or pod-level
			// conventions—otherwise we cannot map reliably to a SourceID.
			return SourceID{}, errors.New("kind not found")
		}
	}

	return SourceID{
		Name:      workloadName.Str(),
		Namespace: ns.Str(),
		Kind:      workloadKind,
	}, nil
}

func workloadKindFromMixedCaseString(s string) k8sconsts.WorkloadKind {
	if k := workload.WorkloadKindFromString(s); k != "" {
		return k
	}
	return k8sconsts.WorkloadKind(s)
}
