package utils

import (
	"context"
	"strings"

	"github.com/odigos-io/odigos/k8sutils/pkg/container"

	"github.com/odigos-io/odigos/common"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkAllPodsRunningAndContainsInstrumentation(pods *corev1.PodList) bool {
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			return false
		}

		if !isPodContainsInstrumentation(&pod) {
			return false
		}

		if !container.AllContainersReady(&pod) {
			return false
		}
	}

	return true
}

func isPodContainsInstrumentation(pod *corev1.Pod) bool {
	for _, c := range pod.Spec.Containers {
		if c.Resources.Limits != nil {
			for val := range c.Resources.Limits {
				if strings.HasPrefix(val.String(), common.OdigosResourceNamespace) {
					return true
				}
			}
		}
	}
	return false
}

func VerifyAllPodsAreInstrumented(ctx context.Context, client kubernetes.Interface, obj client.Object) (bool, error) {
	var labels map[string]string
	switch obj.(type) {
	case *appsv1.Deployment:
		deployment := obj.(*appsv1.Deployment)
		labels = deployment.Spec.Selector.MatchLabels
	case *appsv1.StatefulSet:
		statefulSet := obj.(*appsv1.StatefulSet)
		labels = statefulSet.Spec.Selector.MatchLabels
	case *appsv1.DaemonSet:
		daemonSet := obj.(*appsv1.DaemonSet)
		labels = daemonSet.Spec.Selector.MatchLabels
	}

	pods, err := client.CoreV1().Pods(obj.GetNamespace()).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: labels}),
	})

	if err != nil {
		return false, err
	}

	return checkAllPodsRunningAndContainsInstrumentation(pods), nil
}
