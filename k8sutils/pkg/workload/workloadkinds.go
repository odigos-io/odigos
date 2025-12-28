package workload

import (
	"errors"
	"strings"

	openshiftappsv1 "github.com/openshift/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/version"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
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
	case k8sconsts.WorkloadKindDeployment, k8sconsts.WorkloadKindDaemonSet,
		k8sconsts.WorkloadKindStatefulSet, k8sconsts.WorkloadKindNamespace, k8sconsts.WorkloadKindCronJob,
		k8sconsts.WorkloadKindDeploymentConfig, k8sconsts.WorkloadKindArgoRollout:
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
	case k8sconsts.WorkloadKindNamespace:
		return k8sconsts.WorkloadKindLowerCaseNamespace
	case k8sconsts.WorkloadKindCronJob:
		return k8sconsts.WorkloadKindLowerCaseCronJob
	case k8sconsts.WorkloadKindJob:
		return k8sconsts.WorkloadKindLowerCaseJob
	case k8sconsts.WorkloadKindDeploymentConfig:
		return k8sconsts.WorkloadKindLowerCaseDeploymentConfig
	case k8sconsts.WorkloadKindArgoRollout:
		return k8sconsts.WorkloadKindLowerCaseArgoRollout
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
	case k8sconsts.WorkloadKindLowerCaseCronJob:
		return k8sconsts.WorkloadKindCronJob
	case k8sconsts.WorkloadKindLowerCaseJob:
		return k8sconsts.WorkloadKindJob
	case k8sconsts.WorkloadKindLowerCaseDeploymentConfig:
		return k8sconsts.WorkloadKindDeploymentConfig
	case k8sconsts.WorkloadKindLowerCaseArgoRollout:
		return k8sconsts.WorkloadKindArgoRollout
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
	case string(k8sconsts.WorkloadKindLowerCaseCronJob):
		return k8sconsts.WorkloadKindCronJob
	case string(k8sconsts.WorkloadKindLowerCaseJob):
		return k8sconsts.WorkloadKindJob
	case string(k8sconsts.WorkloadKindLowerCaseDeploymentConfig):
		return k8sconsts.WorkloadKindDeploymentConfig
	case string(k8sconsts.WorkloadKindLowerCaseArgoRollout):
		return k8sconsts.WorkloadKindArgoRollout
	default:
		return k8sconsts.WorkloadKind("")
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
	case k8sconsts.WorkloadKindNamespace:
		return &corev1.Namespace{}
	case k8sconsts.WorkloadKindCronJob:
		ver, err := utils.ClusterVersion()
		if err != nil {
			return &batchv1beta1.CronJob{}
		}

		if ver.LessThan(version.MustParseSemantic("1.21.0")) {
			return &batchv1beta1.CronJob{}
		} else {
			return &batchv1.CronJob{}
		}
	case k8sconsts.WorkloadKindJob:
		return &batchv1.Job{}
	case k8sconsts.WorkloadKindDeploymentConfig:
		return &openshiftappsv1.DeploymentConfig{}
	case k8sconsts.WorkloadKindArgoRollout:
		return &argorolloutsv1alpha1.Rollout{}
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
	case k8sconsts.WorkloadKindCronJob:
		ver, err := utils.ClusterVersion()
		if err != nil {
			return &batchv1beta1.CronJobList{}
		}

		if ver != nil && ver.LessThan(version.MustParseSemantic("1.21.0")) {
			return &batchv1beta1.CronJobList{}
		} else {
			return &batchv1.CronJobList{}
		}
	case k8sconsts.WorkloadKindJob:
		return &batchv1.JobList{}
	case k8sconsts.WorkloadKindDeploymentConfig:
		return &openshiftappsv1.DeploymentConfigList{}
	case k8sconsts.WorkloadKindArgoRollout:
		return &argorolloutsv1alpha1.RolloutList{}
	default:
		return nil
	}
}
