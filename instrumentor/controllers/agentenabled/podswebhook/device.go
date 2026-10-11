package podswebhook

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	containerutils "github.com/odigos-io/odigos/k8sutils/pkg/container"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func InjectDeviceToContainer(container *corev1.Container, device string) {
	if container.Resources.Limits == nil {
		container.Resources.Limits = make(corev1.ResourceList)
	}
	if container.Resources.Requests == nil {
		container.Resources.Requests = make(corev1.ResourceList)
	}

	resourceName := corev1.ResourceName(device)

	container.Resources.Limits[resourceName] = resource.MustParse("1")
	container.Resources.Requests[resourceName] = resource.MustParse("1")
}

func CheckDevicePluginContainersHealth(ctx context.Context, kubeClient client.Client, odigosNamespace string) error {

	odigletDaemonsets := &appsv1.DaemonSetList{}
	selector := labels.SelectorFromSet(map[string]string{"app.kubernetes.io/name": k8sconsts.OdigletDaemonSetName})
	if err := kubeClient.List(ctx, odigletDaemonsets, &client.ListOptions{
		Namespace:     odigosNamespace,
		LabelSelector: selector,
	}); err != nil {
		return err
	}
	if len(odigletDaemonsets.Items) == 0 {
		// no odiglet daemonset: no odiglet pods, so we should not inject instrumentation
		return fmt.Errorf("no odiglet daemonset in namespace %q", odigosNamespace)
	}
	if len(odigletDaemonsets.Items) > 1 {
		return fmt.Errorf("multiple odiglet daemonsets in namespace %q", odigosNamespace)
	}
	odigletDaemonset := &odigletDaemonsets.Items[0]

	odigletPods := corev1.PodList{}
	err := kubeClient.List(ctx, &odigletPods, &client.ListOptions{
		Namespace:     odigosNamespace,
		LabelSelector: labels.SelectorFromSet(odigletDaemonset.Spec.Selector.MatchLabels),
	})
	if err != nil {
		return err
	}

	if len(odigletPods.Items) == 0 {
		return fmt.Errorf("no odiglet pods found")
	}

	for i := range odigletPods.Items {
		pod := &odigletPods.Items[i]
		for j := range pod.Status.ContainerStatuses {
			container := &pod.Status.ContainerStatuses[j]
			// consider only the "deviceplugin" container by it's name
			if container.Name != k8sconsts.OdigletDevicePluginContainerName {
				continue
			}
			// we consider "crash loop backoff" or "image pull backoff" as reasons for not injecting instrumentation.
			// it the container is initializing, starting, or anything else,
			// we assume it will ready shortly and should not block the entire cluster from injection
			if containerutils.IsContainerInBackOff(container) {
				reason := "backoff"
				if containerutils.IsContainerInCrashLoopBackOff(container) {
					reason = "crash loop backoff"
				} else if containerutils.IsContainerInImagePullBackOff(container) {
					reason = "image pull backoff"
				}
				return fmt.Errorf("odiglet %s/%s device plugin container is in %s", pod.Namespace, pod.Name, reason)
			}
		}
	}
	return nil
}
