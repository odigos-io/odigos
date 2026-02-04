package common

import (
	"context"
	"errors"
	"fmt"

	k8spod "github.com/odigos-io/odigos/k8sutils/pkg/pod"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsPodInCurrentNode(pod *corev1.Pod) bool {
	return pod.Spec.NodeName == env.Current.NodeName
}

func GetPodExternalURL(ip string, ports []corev1.ContainerPort) string {
	if ports != nil && len(ports) > 0 {
		return fmt.Sprintf("http://%s:%d", ip, ports[0].ContainerPort)
	}

	return ""
}

func WorkloadPodsOnCurrentNode(c client.Client, ctx context.Context, ic *odigosv1.InstrumentationConfig) ([]corev1.Pod, error) {
	// find pods that are managed by the workload,
	// filter out pods that are being deleted or not ready,
	// note that the controller-runtime cache should only contain pods from the current node
	// however, we still double check that the pods the returned are from the current node
	var podList corev1.PodList
	err := c.List(ctx, &podList, client.InNamespace(ic.Namespace))
	if err != nil {
		return nil, err
	}

	var selectedPods []corev1.Pod
	for _, pod := range podList.Items {
		// skip pods that are being deleted or not ready
		if k8spod.IsPodDeleting(&pod) {
			continue
		}
		if !k8spod.AllContainersReady(&pod) {
			continue
		}
		// making sure the pod is in the current node
		if !IsPodInCurrentNode(&pod) {
			continue
		}
		podWorkload, err := workload.PodWorkloadObject(ctx, &pod)
		if errors.Is(err, workload.ErrKindNotSupported) {
			continue
		}
		if podWorkload == nil {
			// pod is not managed by a workload, no runtime details detection needed
			continue
		}

		// get instrumentation config name for the pod
		instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(podWorkload.Name, podWorkload.Kind)
		if instrumentationConfigName == ic.Name {
			selectedPods = append(selectedPods, pod)
		}
	}

	return selectedPods, nil
}
