package ebpf

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	obisdk "github.com/odigos-io/odigos/odiglet/pkg/ebpf/sdks/obi"
	"github.com/odigos-io/odigos/instrumentation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func obiProcessLifecycleCallback(obiManager *obisdk.Manager, c client.Client) func(context.Context, int, *K8sProcessDetails, bool) {
	return func(ctx context.Context, pid int, pd *K8sProcessDetails, instrumented bool) {
		if pd == nil || pd.Pw == nil || pd.ContainerName == "" {
			return
		}
		if !instrumented {
			obiManager.SyncMetrics(pid, false)
			return
		}

		containerConfig, err := containerConfigForProcess(ctx, c, pd)
		if err != nil || containerConfig == nil {
			return
		}
		obiManager.SyncMetrics(pid, containerConfig.Metrics != nil)
	}
}

func obiProcessConfigCallback(obiManager *obisdk.Manager) func(context.Context, int, *K8sProcessDetails, instrumentation.Config) {
	return func(_ context.Context, pid int, _ *K8sProcessDetails, config instrumentation.Config) {
		containerConfig, ok := config.(*odigosv1.ContainerAgentConfig)
		if !ok {
			obiManager.SyncMetrics(pid, false)
			return
		}
		obiManager.SyncMetrics(pid, containerConfig.Metrics != nil)
	}
}

func containerConfigForProcess(ctx context.Context, c client.Client, pd *K8sProcessDetails) (*odigosv1.ContainerAgentConfig, error) {
	ic := &odigosv1.InstrumentationConfig{}
	icKey := client.ObjectKey{
		Namespace: pd.Pw.Namespace,
		Name:      workload.CalculateWorkloadRuntimeObjectName(pd.Pw.Name, pd.Pw.Kind),
	}
	if err := c.Get(ctx, icKey, ic); err != nil {
		return nil, err
	}

	for idx := range ic.Spec.Containers {
		containerConfig := &ic.Spec.Containers[idx]
		if containerConfig.ContainerName == pd.ContainerName {
			return containerConfig, nil
		}
	}
	return nil, nil
}
