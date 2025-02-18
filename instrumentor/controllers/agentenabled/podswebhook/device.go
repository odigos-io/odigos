package podswebhook

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
