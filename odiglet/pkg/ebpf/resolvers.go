package ebpf

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentation/detector"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type k8sDetailsResolver struct {
	client client.Client
}

func (dr *k8sDetailsResolver) Resolve(ctx context.Context, event detector.ProcessEvent) (K8sProcessDetails, error) {
	pod, err := dr.podFromProcEvent(ctx, event)
	if err != nil {
		return K8sProcessDetails{}, err
	}

	containerName, found := containerNameFromProcEvent(event)
	if !found {
		return K8sProcessDetails{}, errContainerNameNotReported
	}

	distroName, found := distroNameFromProcEvent(event)
	if !found {
		// TODO: this is ok for migration period. Once device is removed, this should be an error
	}

	podWorkload, err := workload.PodWorkloadObjectOrError(ctx, pod)
	if err != nil {
		return K8sProcessDetails{}, fmt.Errorf("failed to find workload object from pod manifest owners references: %w", err)
	}

	return K8sProcessDetails{
		pod:           pod,
		containerName: containerName,
		distroName:    distroName,
		pw:            podWorkload,
		procEvent:     event,
	}, nil
}

func (dr *k8sDetailsResolver) podFromProcEvent(ctx context.Context, e detector.ProcessEvent) (*corev1.Pod, error) {
	eventEnvs := e.ExecDetails.Environments

	podName, ok := eventEnvs[k8sconsts.OdigosEnvVarPodName]
	if !ok {
		return nil, errPodNameNotReported
	}

	podNamespace, ok := eventEnvs[k8sconsts.OdigosEnvVarNamespace]
	if !ok {
		return nil, errPodNameSpaceNotReported
	}

	pod := corev1.Pod{}
	err := dr.client.Get(ctx, client.ObjectKey{Namespace: podNamespace, Name: podName}, &pod)
	if err != nil {
		return nil, fmt.Errorf("error fetching pod object: %w", err)
	}

	return &pod, nil
}

func containerNameFromProcEvent(event detector.ProcessEvent) (string, bool) {
	containerName, ok := event.ExecDetails.Environments[k8sconsts.OdigosEnvVarContainerName]
	return containerName, ok
}

func distroNameFromProcEvent(event detector.ProcessEvent) (string, bool) {
	distronName, ok := event.ExecDetails.Environments[k8sconsts.OdigosEnvVarDistroName]
	return distronName, ok
}

type k8sConfigGroupResolver struct{}

func (cr *k8sConfigGroupResolver) Resolve(ctx context.Context, d K8sProcessDetails, dist *distro.OtelDistro) (K8sConfigGroup, error) {
	if d.pw == nil {
		return K8sConfigGroup{}, fmt.Errorf("podWorkload is not provided, cannot resolve config group")
	}
	return K8sConfigGroup{
		Pw:   *d.pw,
		Lang: dist.Language,
	}, nil
}
