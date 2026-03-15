package runtime_details

import (
	"context"
	"errors"
	"fmt"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	criwrapper "github.com/odigos-io/odigos/k8sutils/pkg/cri"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	kubecommon "github.com/odigos-io/odigos/odiglet/pkg/kube/common"
	"github.com/odigos-io/odigos/odiglet/pkg/process"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// startupRuntimeDetection is a Runnable that performs a
// single batch runtime-detection pass for all relevant pods on this node.
// It is used to optimize the initial CPU burst needed to scan the processes on the node.
// It is done by a single /proc scan
type startupRuntimeDetection struct {
	client               client.Client
	criClient            *criwrapper.CriClient
	runtimeDetectionEnvs map[string]struct{}
}

var _ manager.Runnable = &startupRuntimeDetection{}

func (s *startupRuntimeDetection) Start(ctx context.Context) error {
	logger := commonlogger.FromContext(ctx)

	icCount, err := s.scan(ctx)
	if err != nil {
		logger.Error(err, "failed to perform runtime-detection initial scan, some of the instrumented workloads might have stale runtime details")
	} else {
		logger.Info("Completed runtime details initial scan", "workloads count", icCount)
	}

	return nil
}

func (s *startupRuntimeDetection) scan(ctx context.Context) (int, error) {
	var icList odigosv1.InstrumentationConfigList
	if err := s.client.List(ctx, &icList); err != nil {
		return 0, fmt.Errorf("failed to list instrumentation configs: %w", err)
	}

	var podList corev1.PodList
	if err := s.client.List(ctx, &podList); err != nil {
		return 0, fmt.Errorf("failed to list pods: %w", err)
	}

	icPods := make([]struct  {
		ic   *odigosv1.InstrumentationConfig
		pods []corev1.Pod
	}, 0, len(icList.Items))

	// group pods and instrumentation config they are associated to
	for i, ic := range icList.Items {
		pods, err := kubecommon.MatchingPodsForWorkloadOnNode(&ic, podList)
		if err != nil {
			return 0, fmt.Errorf("failed to get matching pods for ic: %w", err)
		}
		icPods = append(icPods, struct{ic *odigosv1.InstrumentationConfig; pods []corev1.Pod}{
			ic: &icList.Items[i],
			pods: pods,
		})
	}

	if len(icPods) == 0 {
		return 0, nil
	}

	// Build the full set of (podUID, containerName) across all pods.
	var allPCs []process.PodContainerUID
	for _, entry := range icPods {
		for i := range entry.pods {
			uid := workload.PodUID(&entry.pods[i])
			for _, c := range entry.pods[i].Spec.Containers {
				allPCs = append(allPCs, process.PodContainerUID{
					PodUID:        uid,
					ContainerName: c.Name,
				})
			}
		}
	}

	// Single /proc scan for all containers on the node, group the processes by (pod uid, container name)
	groups, err := process.GroupByPodContainer(allPCs)
	if err != nil {
		return 0, fmt.Errorf("startup runtime detection: failed to group processes: %w", err)
	}

	// For each IC, run runtimeInspection with pre-grouped PIDs.
	// and persist the result to the matching instrumentation config
	var inspectionErr error
	for _, entry := range icPods {
		results, err := runtimeInspectionFromGroupedPIDs(ctx, entry.pods, groups, s.criClient, s.runtimeDetectionEnvs)
		if err != nil {
			inspectionErr = errors.Join(inspectionErr, err)
			continue
		}

		// persist the result with retry to handle possible conflict errors
		err = wait.ExponentialBackoff(wait.Backoff{
			Duration: 100 * time.Millisecond,
			Factor:   2.0,
			Jitter:   0.1,
			Steps:    5,
		}, func() (bool, error) {
			err := persistRuntimeDetailsToInstrumentationConfig(ctx, s.client, entry.ic, results)
			if err != nil {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			inspectionErr = errors.Join(inspectionErr, err)
			continue
		}
	}

	return len(icPods), inspectionErr
}
