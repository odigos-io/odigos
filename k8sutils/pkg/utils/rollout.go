package utils

import (
	"context"
	"strconv"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	openshiftappsv1 "github.com/openshift/api/apps/v1"

	"github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	"github.com/odigos-io/odigos/k8sutils/pkg/container"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

func VerifyAllPodsAreRunning(ctx context.Context, k8sclient kubernetes.Interface, obj metav1.Object) (bool, error) {
	if !IsWorkloadRolloutDone(obj) {
		return false, nil
	}

	labels := GetMatchLabels(obj)
	if labels == nil {
		return true, nil
	}

	labelSelector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	pods, err := k8sclient.CoreV1().Pods(obj.GetNamespace()).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(labelSelector),
	})

	if err != nil {
		return false, err
	}

	return checkAllPodsRunning(pods), nil
}

func GetMatchLabels(obj metav1.Object) map[string]string {
	var labels map[string]string
	switch obj := obj.(type) {
	case *appsv1.Deployment:
		labels = obj.Spec.Selector.MatchLabels
	case *appsv1.StatefulSet:
		labels = obj.Spec.Selector.MatchLabels
	case *appsv1.DaemonSet:
		labels = obj.Spec.Selector.MatchLabels
	case *openshiftappsv1.DeploymentConfig:
		// DeploymentConfig selector is map[string]string directly
		labels = obj.Spec.Selector
	case *argorolloutsv1alpha1.Rollout:
		labels = obj.Spec.Selector.MatchLabels
	default:
		return nil
	}
	return labels
}

// IsWorkloadRolloutDone checks if the rollout of the given workload is done.
// this is based on the kubectl implementation of checking the rollout status:
// https://github.com/kubernetes/kubectl/blob/master/pkg/polymorphichelpers/rollout_status.go
func isDeploymentRolloutDone(d *appsv1.Deployment) bool {
	if d.Generation > d.Status.ObservedGeneration {
		return false
	}

	cond := conditions.GetDeploymentCondition(d.Status, appsv1.DeploymentProgressing)
	if cond != nil && cond.Reason == conditions.TimedOutReason {
		// deployment exceeded its progress deadline
		return false
	}
	if d.Spec.Replicas != nil && d.Status.UpdatedReplicas < *d.Spec.Replicas {
		// Waiting for deployment rollout to finish
		return false
	}
	if d.Status.Replicas > d.Status.UpdatedReplicas {
		// Waiting for deployment rollout to finish old replicas are pending termination.
		return false
	}
	if d.Status.AvailableReplicas < d.Status.UpdatedReplicas {
		// Waiting for deployment rollout to finish:  not all updated replicas are available..
		return false
	}
	return true
}

func isStatefulSetRolloutDone(s *appsv1.StatefulSet) bool {
	if s.Spec.UpdateStrategy.Type != appsv1.RollingUpdateStatefulSetStrategyType {
		// rollout status is only available for RollingUpdateStatefulSetStrategyType strategy type
		return true
	}
	if s.Status.ObservedGeneration == 0 || s.Generation > s.Status.ObservedGeneration {
		// Waiting for statefulset spec update to be observed
		return false
	}
	if s.Spec.Replicas != nil && s.Status.ReadyReplicas < *s.Spec.Replicas {
		// Waiting for pods to be ready
		return false
	}
	if s.Spec.UpdateStrategy.Type == appsv1.RollingUpdateStatefulSetStrategyType && s.Spec.UpdateStrategy.RollingUpdate != nil {
		if s.Spec.Replicas != nil && s.Spec.UpdateStrategy.RollingUpdate.Partition != nil {
			if s.Status.UpdatedReplicas < (*s.Spec.Replicas - *s.Spec.UpdateStrategy.RollingUpdate.Partition) {
				// Waiting for partitioned roll out to finish
				return false
			}
		}
		// partitioned roll out complete
		return true
	}
	if s.Status.UpdateRevision != s.Status.CurrentRevision {
		// waiting for statefulset rolling update to complete
		return false
	}
	return true
}

func isDaemonSetRolloutDone(d *appsv1.DaemonSet) bool {
	if d.Spec.UpdateStrategy.Type != appsv1.RollingUpdateDaemonSetStrategyType {
		// rollout status is only available for RollingUpdateDaemonSetStrategyType strategy type
		return true
	}
	if d.Generation > d.Status.ObservedGeneration {
		// Waiting for daemon set spec update to be observed
		return false
	}
	if d.Status.UpdatedNumberScheduled < d.Status.DesiredNumberScheduled {
		// Waiting for daemon set rollout to finish
		return false
	}
	if d.Status.NumberAvailable < d.Status.DesiredNumberScheduled {
		// Waiting for daemon set rollout to finish
		return false
	}
	return true
}

func isDeploymentConfigRolloutDone(dc *openshiftappsv1.DeploymentConfig) bool {
	if dc.Generation > dc.Status.ObservedGeneration {
		return false
	}
	if dc.Status.Replicas > dc.Status.UpdatedReplicas {
		// Waiting for deploymentconfig rollout to finish old replicas are pending termination.
		return false
	}
	if dc.Status.AvailableReplicas < dc.Status.UpdatedReplicas {
		// Waiting for deploymentconfig rollout to finish: not all updated replicas are available.
		return false
	}
	if dc.Status.UnavailableReplicas > 0 {
		// Waiting for deploymentconfig rollout to finish: replicas are unavailable.
		return false
	}
	return true
}

func isArgoRolloutRolloutDone(rollout *argorolloutsv1alpha1.Rollout) bool {
	// Yes, this name is ridiculous. The function returns whether the rollout of an Argo rollout is done.

	// Check phase first - it's the most reliable indicator
	switch rollout.Status.Phase {
	case argorolloutsv1alpha1.RolloutPhaseHealthy, argorolloutsv1alpha1.RolloutPhasePaused:
		return true
	case argorolloutsv1alpha1.RolloutPhaseDegraded, argorolloutsv1alpha1.RolloutPhaseProgressing:
		return false
	}

	// Check if the spec has been observed yet
	// ObservedGeneration in Argo Rollouts is a string, so we need to parse it
	observedGen, err := strconv.ParseInt(rollout.Status.ObservedGeneration, 10, 64)
	if err != nil || rollout.Generation > observedGen {
		return false
	}
	// Default to true
	return true
}

func IsWorkloadRolloutDone(obj metav1.Object) bool {
	switch o := obj.(type) {
	case *appsv1.Deployment:
		return isDeploymentRolloutDone(o)
	case *appsv1.StatefulSet:
		return isStatefulSetRolloutDone(o)
	case *appsv1.DaemonSet:
		return isDaemonSetRolloutDone(o)
	case *openshiftappsv1.DeploymentConfig:
		return isDeploymentConfigRolloutDone(o)
	case *argorolloutsv1alpha1.Rollout:
		return isArgoRolloutRolloutDone(o)
	default:
		return false
	}
}
