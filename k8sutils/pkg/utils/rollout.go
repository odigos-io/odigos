package utils

import (
	"context"

	"github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	"github.com/odigos-io/odigos/k8sutils/pkg/container"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkAllPodsRunning(pods *corev1.PodList) bool {
	for i := range pods.Items {
		pod := &pods.Items[i]
		if pod.Status.Phase != corev1.PodRunning {
			return false
		}

		// Check if restart count is 0
		for j := range pod.Status.ContainerStatuses {
			if pod.Status.ContainerStatuses[j].RestartCount != 0 {
				return false
			}
		}

		if !container.AllContainersReady(pod) {
			return false
		}
	}

	return true
}

func VerifyAllPodsAreRunning(ctx context.Context, k8sclient kubernetes.Interface, obj client.Object) (bool, error) {
	if !IsWorkloadRolloutDone(obj) {
		return false, nil
	}

	labels := GetMatchLabels(obj)
	if labels == nil {
		return true, nil
	}
	pods, err := k8sclient.CoreV1().Pods(obj.GetNamespace()).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: labels}),
	})

	if err != nil {
		return false, err
	}

	return checkAllPodsRunning(pods), nil
}

func GetMatchLabels(obj client.Object) map[string]string {
	var labels map[string]string
	switch obj := obj.(type) {
	case *appsv1.Deployment:
		labels = obj.Spec.Selector.MatchLabels
	case *appsv1.StatefulSet:
		labels = obj.Spec.Selector.MatchLabels
	case *appsv1.DaemonSet:
		labels = obj.Spec.Selector.MatchLabels
	default:
		return nil
	}
	return labels
}

// IsWorkloadRolloutDone checks if the rollout of the given workload is done.
// this is based on the kubectl implementation of checking the rollout status:
// https://github.com/kubernetes/kubectl/blob/master/pkg/polymorphichelpers/rollout_status.go
func IsWorkloadRolloutDone(obj client.Object) bool {
	switch o := obj.(type) {
	case *appsv1.Deployment:
		if o.Generation <= o.Status.ObservedGeneration {
			cond := conditions.GetDeploymentCondition(o.Status, appsv1.DeploymentProgressing)
			if cond != nil && cond.Reason == conditions.TimedOutReason {
				// deployment exceeded its progress deadline
				return false
			}
			if o.Spec.Replicas != nil && o.Status.UpdatedReplicas < *o.Spec.Replicas {
				// Waiting for deployment rollout to finish
				return false
			}
			if o.Status.Replicas > o.Status.UpdatedReplicas {
				// Waiting for deployment rollout to finish old replicas are pending termination.
				return false
			}
			if o.Status.AvailableReplicas < o.Status.UpdatedReplicas {
				// Waiting for deployment rollout to finish:  not all updated replicas are available..
				return false
			}
			return true
		}
		return false
	case *appsv1.StatefulSet:
		if o.Spec.UpdateStrategy.Type != appsv1.RollingUpdateStatefulSetStrategyType {
			// rollout status is only available for RollingUpdateStatefulSetStrategyType strategy type
			return true
		}
		if o.Status.ObservedGeneration == 0 || o.Generation > o.Status.ObservedGeneration {
			// Waiting for statefulset spec update to be observed
			return false
		}
		if o.Spec.Replicas != nil && o.Status.ReadyReplicas < *o.Spec.Replicas {
			// Waiting for pods to be ready
			return false
		}
		if o.Spec.UpdateStrategy.Type == appsv1.RollingUpdateStatefulSetStrategyType && o.Spec.UpdateStrategy.RollingUpdate != nil {
			if o.Spec.Replicas != nil && o.Spec.UpdateStrategy.RollingUpdate.Partition != nil {
				if o.Status.UpdatedReplicas < (*o.Spec.Replicas - *o.Spec.UpdateStrategy.RollingUpdate.Partition) {
					// Waiting for partitioned roll out to finish
					return false
				}
			}
			// partitioned roll out complete
			return true
		}
		if o.Status.UpdateRevision != o.Status.CurrentRevision {
			// waiting for statefulset rolling update to complete
			return false
		}
		return true
	case *appsv1.DaemonSet:
		if o.Spec.UpdateStrategy.Type != appsv1.RollingUpdateDaemonSetStrategyType {
			// rollout status is only available for RollingUpdateDaemonSetStrategyType strategy type
			return true
		}
		if o.Generation <= o.Status.ObservedGeneration {
			if o.Status.UpdatedNumberScheduled < o.Status.DesiredNumberScheduled {
				// Waiting for daemon set rollout to finish
				return false
			}
			if o.Status.NumberAvailable < o.Status.DesiredNumberScheduled {
				// Waiting for daemon set rollout to finish
				return false
			}
			return true
		}
		// Waiting for daemon set spec update to be observed
		return false
	default:
		return false
	}
}
