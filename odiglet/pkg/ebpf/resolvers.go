package ebpf

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/common/instrumentation/types"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type k8sDetailsResolver struct {
	client client.Client
}

func (dr *k8sDetailsResolver) Resolve(ctx context.Context, event types.ProcessEvent) (K8sDetails, error) {
	pod, err := dr.podFromProcEvent(ctx, event)
	if err != nil {
		return K8sDetails{}, err
	}

	containerName, found := containerNameFromProcEvent(event)
	if !found {
		return K8sDetails{}, errContainerNameNotReported
	}

	podWorkload, err := workload.PodWorkloadObjectOrError(ctx, pod)
	if err != nil {
		return K8sDetails{}, fmt.Errorf("failed to find workload object from pod manifest owners references: %w", err)
	}

	return K8sDetails{
		pod:           pod,
		containerName: containerName,
		pw:            podWorkload,
	}, nil
}

func (dr *k8sDetailsResolver) podFromProcEvent(ctx context.Context, e types.ProcessEvent) (*corev1.Pod, error) {
	eventEnvs := e.ExecDetails.Environments

	podName, ok := eventEnvs[consts.OdigosEnvVarPodName]
	if !ok {
		return nil, errPodNameNotReported
	}

	podNamespace, ok := eventEnvs[consts.OdigosEnvVarNamespace]
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

func containerNameFromProcEvent(event types.ProcessEvent) (string, bool) {
	containerName, ok := event.ExecDetails.Environments[consts.OdigosEnvVarContainerName]
	return containerName, ok
}

type k8sConfigGroupResolver struct{}

func (cr *k8sConfigGroupResolver) Resolve(ctx context.Context, d K8sDetails, dist types.OtelDistribution) (K8sConfigGroup, error) {
	if d.pw == nil {
		return K8sConfigGroup{}, fmt.Errorf("podWorkload is not provided, cannot resolve config group")
	}
	return K8sConfigGroup{
		Pw:   *d.pw,
		Lang: dist.Language,
	}, nil
}