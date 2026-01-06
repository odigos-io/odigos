package ebpf

import (
	"context"
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/instrumentation/detector"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// k8sDetailsResolver is responsible for resolving K8sProcessDetails from a ProcessEvent
// It fetches the associated Pod object and extracts relevant information such as
// container name, distribution details, and workload information.
type k8sDetailsResolver struct {
	client             client.Client
	distributionGetter *distros.Getter
}

var _ instrumentation.ProcessDetailsResolver[K8sProcessGroup, K8sConfigGroup, *K8sProcessDetails] = &k8sDetailsResolver{}

func (dr *k8sDetailsResolver) Resolve(ctx context.Context, event detector.ProcessEvent) (*K8sProcessDetails, error) {
	pod, err := dr.podFromProcEvent(ctx, event)
	if err != nil {
		return nil, err
	}

	containerName, found := containerNameFromProcEvent(event)
	if !found {
		return nil, errContainerNameNotReported
	}

	distroName, found := distroNameFromProcEvent(event)
	if !found {
		return nil, fmt.Errorf("distro name not reported in process event for container: %s", containerName)
	}

	podWorkload, err := workload.PodWorkloadObjectOrError(ctx, pod)
	if err != nil {
		return nil, fmt.Errorf("failed to find workload object from pod manifest owners references: %w", err)
	}

	distro := dr.distributionGetter.GetDistroByName(distroName)
	if distro == nil {
		return nil, fmt.Errorf("distro %s not found for container: %s", distroName, containerName)
	}

	// verify the language of the process event matches the detected language for the container
	// for containers with multiple processes or a script that spawns other processes, the language
	// being detected depends on timing and we may get events for un-related processes.
	if ok := inspectors.VerifyLanguage(process.Details{
		ProcessID: event.PID,
		ExePath:   event.ExecDetails.ExePath,
		CmdLine:   event.ExecDetails.CmdLine,
		Environments: process.ProcessEnvs{
			DetailedEnvs: event.ExecDetails.Environments,
		},
	}, distro.Language, log.Logger); !ok {
		return nil, errors.Join(instrumentation.ErrProcessLanguageNotMatchesDistribution,
			fmt.Errorf("process with exe path %s does not match the detected language (%s) for container: %s", event.ExecDetails.ExePath, distro.Language, containerName))
	}

	return &K8sProcessDetails{
		Pod:           pod,
		ContainerName: containerName,
		Distro:        distro,
		Pw:            podWorkload,
		ProcEvent:     event,
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
