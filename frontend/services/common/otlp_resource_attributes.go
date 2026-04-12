package common

import (
	"errors"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// ResourceAttributesToSourceID resolves namespace, workload kind and name from OTLP resource
// attributes using the same rules as the node collector traffic metrics processor
// (see odigostrafficmetrics res_attributes_keys and metricAttributesToSourceID history).
func ResourceAttributesToSourceID(attrs pcommon.Map) (SourceID, error) {
	ns, ok := attrs.Get(string(semconv.K8SNamespaceNameKey))
	if !ok {
		return SourceID{}, errors.New("namespace not found")
	}

	var kind k8sconsts.WorkloadKind
	var name pcommon.Value
	var found bool

	if odigosWorkloadName, ok := attrs.Get(odigosconsts.OdigosWorkloadNameAttribute); ok {
		name, found = odigosWorkloadName, true
	}

	if odigosKind, ok := attrs.Get(odigosconsts.OdigosWorkloadKindAttribute); ok {
		kind = k8sconsts.WorkloadKind(odigosKind.Str())

		if !found {
			switch kind {
			case k8sconsts.WorkloadKindDeployment:
				name, found = attrs.Get(string(semconv.K8SDeploymentNameKey))
			case k8sconsts.WorkloadKindStatefulSet:
				name, found = attrs.Get(string(semconv.K8SStatefulSetNameKey))
			case k8sconsts.WorkloadKindDaemonSet:
				name, found = attrs.Get(string(semconv.K8SDaemonSetNameKey))
			case k8sconsts.WorkloadKindCronJob:
				name, found = attrs.Get(string(semconv.K8SCronJobNameKey))
			case k8sconsts.WorkloadKindJob:
				name, found = attrs.Get(string(semconv.K8SJobNameKey))
			case k8sconsts.WorkloadKindArgoRollout:
				name, found = attrs.Get(k8sconsts.K8SArgoRolloutNameAttribute)
			}
		}

		if !found {
			return SourceID{}, errors.New("workload name not found")
		}
	} else {
		if depName, ok := attrs.Get(string(semconv.K8SDeploymentNameKey)); ok {
			kind = k8sconsts.WorkloadKindDeployment
			name = depName
		} else if ssName, ok := attrs.Get(string(semconv.K8SStatefulSetNameKey)); ok {
			kind = k8sconsts.WorkloadKindStatefulSet
			name = ssName
		} else if dsName, ok := attrs.Get(string(semconv.K8SDaemonSetNameKey)); ok {
			kind = k8sconsts.WorkloadKindDaemonSet
			name = dsName
		} else if cjName, ok := attrs.Get(string(semconv.K8SCronJobNameKey)); ok {
			kind = k8sconsts.WorkloadKindCronJob
			name = cjName
		} else if jobName, ok := attrs.Get(string(semconv.K8SJobNameKey)); ok {
			kind = k8sconsts.WorkloadKindJob
			name = jobName
		} else if rolloutName, ok := attrs.Get(k8sconsts.K8SArgoRolloutNameAttribute); ok {
			kind = k8sconsts.WorkloadKindArgoRollout
			name = rolloutName
		} else {
			return SourceID{}, errors.New("kind not found")
		}
	}

	return SourceID{
		Name:      name.Str(),
		Namespace: ns.Str(),
		Kind:      kind,
	}, nil
}
