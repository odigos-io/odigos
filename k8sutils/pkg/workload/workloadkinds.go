package workload

import (
	"errors"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This go file contains utils to handle the kind of odigos workloads.
// it allows transforming deployments / daemonsets / statefulsets from one representation to another

var ErrKindNotSupported = errors.New("workload kind not supported")

func IsErrorKindNotSupported(err error) bool {
	return err == ErrKindNotSupported
}

func IgnoreErrorKindNotSupported(err error) error {
	if IsErrorKindNotSupported(err) {
		return nil
	}
	return err
}

func IsValidWorkloadKind(kind k8sconsts.WorkloadKind) bool {
	switch kind {
	case k8sconsts.WorkloadKindDeployment, k8sconsts.WorkloadKindDaemonSet, k8sconsts.WorkloadKindStatefulSet, k8sconsts.WorkloadKindNamespace:
		return true
	}
	return false
}

func WorkloadKindLowerCaseFromKind(pascalCase k8sconsts.WorkloadKind) k8sconsts.WorkloadKindLowerCase {
	switch pascalCase {
	case k8sconsts.WorkloadKindDeployment:
		return k8sconsts.WorkloadKindLowerCaseDeployment
	case k8sconsts.WorkloadKindDaemonSet:
		return k8sconsts.WorkloadKindLowerCaseDaemonSet
	case k8sconsts.WorkloadKindStatefulSet:
		return k8sconsts.WorkloadKindLowerCaseStatefulSet
	}
	return ""
}

func WorkloadKindFromLowerCase(lowerCase k8sconsts.WorkloadKindLowerCase) k8sconsts.WorkloadKind {
	switch lowerCase {
	case k8sconsts.WorkloadKindLowerCaseDeployment:
		return k8sconsts.WorkloadKindDeployment
	case k8sconsts.WorkloadKindLowerCaseDaemonSet:
		return k8sconsts.WorkloadKindDaemonSet
	case k8sconsts.WorkloadKindLowerCaseStatefulSet:
		return k8sconsts.WorkloadKindStatefulSet
	}
	return ""
}

func WorkloadKindFromString(kind string) k8sconsts.WorkloadKind {
	switch strings.ToLower(kind) {
	case string(k8sconsts.WorkloadKindLowerCaseDeployment):
		return k8sconsts.WorkloadKindDeployment
	case string(k8sconsts.WorkloadKindLowerCaseDaemonSet):
		return k8sconsts.WorkloadKindDaemonSet
	case string(k8sconsts.WorkloadKindLowerCaseStatefulSet):
		return k8sconsts.WorkloadKindStatefulSet
	default:
		return k8sconsts.WorkloadKind("")
	}
}

func WorkloadKindFromClientObject(w client.Object) k8sconsts.WorkloadKind {
	switch w.(type) {
	case *v1.Deployment:
		return k8sconsts.WorkloadKindDeployment
	case *v1.DaemonSet:
		return k8sconsts.WorkloadKindDaemonSet
	case *v1.StatefulSet:
		return k8sconsts.WorkloadKindStatefulSet
	default:
		return ""
	}
}

// ClientObjectFromWorkloadKind returns a new instance of the client object for the given workload kind
// the returned instance is empty and should be used to fetch the actual object from the k8s api server
func ClientObjectFromWorkloadKind(kind k8sconsts.WorkloadKind) client.Object {
	switch kind {
	case k8sconsts.WorkloadKindDeployment:
		return &v1.Deployment{}
	case k8sconsts.WorkloadKindDaemonSet:
		return &v1.DaemonSet{}
	case k8sconsts.WorkloadKindStatefulSet:
		return &v1.StatefulSet{}
	default:
		return nil
	}
}

func ClientListObjectFromWorkloadKind(kind k8sconsts.WorkloadKind) client.ObjectList {
	switch kind {
	case k8sconsts.WorkloadKindDeployment:
		return &v1.DeploymentList{}
	case k8sconsts.WorkloadKindDaemonSet:
		return &v1.DaemonSetList{}
	case k8sconsts.WorkloadKindStatefulSet:
		return &v1.StatefulSetList{}
	default:
		return nil
	}
}
