package utils

import (
	"context"

	"github.com/odigos-io/odigos/k8sutils/pkg/container"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/batch/v1"
	v1beta1 "k8s.io/api/batch/v1beta1"
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
	labels := GetMatchLabels(obj)
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
	case *v1.CronJob:
		labels = obj.Spec.JobTemplate.Spec.Selector.MatchLabels
	case *v1beta1.CronJob:
		labels = obj.Spec.JobTemplate.Spec.Selector.MatchLabels
	}
	return labels
}
