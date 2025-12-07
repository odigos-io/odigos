package common

import (
	"context"
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	k8scontainer "github.com/odigos-io/odigos/k8sutils/pkg/container"
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

func GetPodWorkloadObject(pod *corev1.Pod) (*k8sconsts.PodWorkload, error) {
	for _, owner := range pod.OwnerReferences {
		workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(owner)
		if err != nil {
			return nil, workload.IgnoreErrorKindNotSupported(err)
		}

		return &k8sconsts.PodWorkload{
			Name:      workloadName,
			Kind:      workloadKind,
			Namespace: pod.Namespace,
		}, nil
	}

	// Pod does not necessarily have to be managed by a controller
	return nil, nil
}

func WorkloadPodsOnCurrentNode(c client.Client, ctx context.Context, ic *odigosv1.InstrumentationConfig) ([]corev1.Pod, error) {
	// find pods that are managed by the workload,
	// filter out pods that are being deleted or not ready,
	// note that the controller-runtime cache is assumed here to only contain pods in the same node as the odiglet
	var podList corev1.PodList
	err := c.List(ctx, &podList, client.InNamespace(ic.Namespace))
	if err != nil {
		return nil, err
	}

	var selectedPods []corev1.Pod
	for _, pod := range podList.Items {
		// skip pods that are being deleted or not ready
		if pod.DeletionTimestamp != nil || !k8scontainer.AllContainersReady(&pod) {
			continue
		}
		podWorkload, err := GetPodWorkloadObject(&pod)
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
