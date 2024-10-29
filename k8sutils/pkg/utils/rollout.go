package utils

import (
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsDeploymentRolloutCompleted(deployment *appsv1.Deployment) bool {
	if deployment.Spec.Replicas != nil {
		if deployment.Status.UpdatedReplicas != *deployment.Spec.Replicas {
			return false
		}

		if deployment.Status.Replicas != *deployment.Spec.Replicas {
			return false
		}

		if deployment.Status.AvailableReplicas != *deployment.Spec.Replicas {
			return false
		}
	}

	return true
}

func IsStatefulSetRolloutCompleted(statefulSet *appsv1.StatefulSet) bool {
	if statefulSet.Spec.Replicas != nil {
		if statefulSet.Status.UpdatedReplicas != *statefulSet.Spec.Replicas {
			return false
		}

		if statefulSet.Status.Replicas != *statefulSet.Spec.Replicas {
			return false
		}

		if statefulSet.Status.ReadyReplicas != *statefulSet.Spec.Replicas {
			return false
		}
	}

	return true
}

func IsDaemonSetRolloutCompleted(daemonSet *appsv1.DaemonSet) bool {
	if daemonSet.Status.DesiredNumberScheduled != daemonSet.Status.CurrentNumberScheduled {
		return false
	}

	if daemonSet.Status.NumberReady != daemonSet.Status.DesiredNumberScheduled {
		return false
	}

	if daemonSet.Status.UpdatedNumberScheduled != daemonSet.Status.DesiredNumberScheduled {
		return false
	}

	if daemonSet.Status.NumberUnavailable > 0 {
		return false
	}

	return true
}

func IsRolloutCompleted(obj client.Object) bool {
	switch obj.(type) {
	case *appsv1.Deployment:
		deployment := obj.(*appsv1.Deployment)
		return IsDeploymentRolloutCompleted(deployment)
	case *appsv1.StatefulSet:
		statefulSet := obj.(*appsv1.StatefulSet)
		return IsStatefulSetRolloutCompleted(statefulSet)
	case *appsv1.DaemonSet:
		daemonSet := obj.(*appsv1.DaemonSet)
		return IsDaemonSetRolloutCompleted(daemonSet)
	}

	return false
}
