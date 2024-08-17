package workload

import (
	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This go file contains utils to handle the kind of odigos workloads.
// it allows transforming deployments / daemonsets / statefulsets from one representation to another

// 1. the pascal case representation of the workload kind
// it is used in k8s api objects as the `Kind` field.
type WorkloadKind string

const (
	WorkloadKindDeployment  WorkloadKind = "Deployment"
	WorkloadKindDaemonSet   WorkloadKind = "DaemonSet"
	WorkloadKindStatefulSet WorkloadKind = "StatefulSet"
)

// 2. the lower case representation of the workload kind
// is used in odigos with the object name for instrumentation config and runtime details
type WorkloadKindLowerCase string

const (
	WorkloadKindLowerCaseDeployment  WorkloadKindLowerCase = "deployment"
	WorkloadKindLowerCaseDaemonSet   WorkloadKindLowerCase = "daemonset"
	WorkloadKindLowerCaseStatefulSet WorkloadKindLowerCase = "statefulset"
)

func WorkloadKindLowerCaseFromKind(pascalCase WorkloadKind) WorkloadKindLowerCase {
	switch pascalCase {
	case WorkloadKindDeployment:
		return WorkloadKindLowerCaseDeployment
	case WorkloadKindDaemonSet:
		return WorkloadKindLowerCaseDaemonSet
	case WorkloadKindStatefulSet:
		return WorkloadKindLowerCaseStatefulSet
	}
	return ""
}

func WorkloadKindFromLowerCase(lowerCase WorkloadKindLowerCase) WorkloadKind {
	switch lowerCase {
	case WorkloadKindLowerCaseDeployment:
		return WorkloadKindDeployment
	case WorkloadKindLowerCaseDaemonSet:
		return WorkloadKindDaemonSet
	case WorkloadKindLowerCaseStatefulSet:
		return WorkloadKindStatefulSet
	}
	return ""
}

func WorkloadKindFromClientObject(w client.Object) WorkloadKind {
	switch w.(type) {
	case *v1.Deployment:
		return WorkloadKindDeployment
	case *v1.DaemonSet:
		return WorkloadKindDaemonSet
	case *v1.StatefulSet:
		return WorkloadKindStatefulSet
	default:
		return ""
	}
}

func ClientObjectFromWorkloadKind(kind WorkloadKind) client.Object {
	switch kind {
	case WorkloadKindDeployment:
		return &v1.Deployment{}
	case WorkloadKindDaemonSet:
		return &v1.DaemonSet{}
	case WorkloadKindStatefulSet:
		return &v1.StatefulSet{}
	default:
		return nil
	}
}
