package ebpf

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/instrumentation/detector"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/process"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// k8sDetailsResolver is responsible for resolving K8sProcessDetails from a ProcessEvent
// It fetches the associated Pod object and extracts relevant information such as
// container name, distribution details, and workload information.
type k8sDetailsResolver struct {
	client             client.Client
	distributionGetter *distros.Getter

	// podContainerOfPid, when set, replaces process.PodContainerOfPid (tests).
	podContainerOfPid func(pid int) (process.PodContainerKey, bool)
}

var _ instrumentation.ProcessDetailsResolver[K8sProcessGroup, K8sConfigGroup, *K8sProcessDetails] = &k8sDetailsResolver{}

func (dr *k8sDetailsResolver) Resolve(ctx context.Context, event detector.ProcessEvent) (*K8sProcessDetails, error) {
	if _, ok := event.ExecDetails.Environments[k8sconsts.OdigosEnvVarPodName]; !ok {
		return dr.resolveWithoutEnv(ctx, event)
	}
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
		// TODO: this is ok for migration period. Once device is removed, this should be an error
	}

	podWorkload, err := workload.PodWorkloadObjectOrError(pod)
	if err != nil {
		return nil, fmt.Errorf("failed to find workload object from pod manifest owners references: %w", err)
	}

	distro := dr.distributionGetter.GetDistroByName(distroName)

	return &K8sProcessDetails{
		Pod:           pod,
		ContainerName: containerName,
		Distro:        distro,
		Pw:            podWorkload,
		ProcEvent:     event,
	}, nil
}

// resolveWithoutEnv resolves a process that Odigos did not inject its env into: one started, for example
// by forking, in a pod created before its workload was instrumented, whose processes the odiglet
// instruments without a restart (InstrumentationConfigReconciler.sendInstrumentationRequest). The
// detector reports such a process as a fork of a tracked one. Its pod and container come from its
// kubelet mounts, and its distribution from the workload's InstrumentationConfig, for a container that
// sendInstrumentationRequest would instrument too.
func (dr *k8sDetailsResolver) resolveWithoutEnv(ctx context.Context, event detector.ProcessEvent) (*K8sProcessDetails, error) {
	podContainerOfPid := dr.podContainerOfPid
	if podContainerOfPid == nil {
		podContainerOfPid = process.PodContainerOfPid
	}
	key, ok := podContainerOfPid(event.PID)
	if !ok {
		return nil, errPodNameNotReported
	}

	// The cache holds the pods of this node only.
	var pods corev1.PodList
	if err := dr.client.List(ctx, &pods); err != nil {
		return nil, fmt.Errorf("error listing pods: %w", err)
	}
	var pod *corev1.Pod
	for i := range pods.Items {
		if workload.PodUID(&pods.Items[i]) == key.PodUID {
			pod = &pods.Items[i]
			break
		}
	}
	if pod == nil {
		return nil, errPodNameNotReported
	}

	podWorkload, err := workload.PodWorkloadObjectOrError(pod)
	if err != nil {
		return nil, fmt.Errorf("failed to find workload object from pod manifest owners references: %w", err)
	}
	ic := odigosv1.InstrumentationConfig{}
	icKey := client.ObjectKey{Namespace: pod.Namespace, Name: workload.CalculateWorkloadRuntimeObjectName(podWorkload.Name, podWorkload.Kind)}
	if err := dr.client.Get(ctx, icKey, &ic); err != nil {
		return nil, fmt.Errorf("error fetching instrumentation config: %w", err)
	}
	for _, c := range ic.Spec.Containers {
		if c.ContainerName != key.ContainerName {
			continue
		}
		// Without these the process is instrumented only once its pod is recreated with the env.
		if c.OtelDistroName == "" || !c.AgentEnabled || !c.PodManifestInjectionOptional {
			return nil, errPodNameNotReported
		}
		return &K8sProcessDetails{
			Pod:           pod,
			ContainerName: key.ContainerName,
			Distro:        dr.distributionGetter.GetDistroByName(c.OtelDistroName),
			Pw:            podWorkload,
			ProcEvent:     event,
		}, nil
	}
	return nil, errContainerNameNotReported
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
