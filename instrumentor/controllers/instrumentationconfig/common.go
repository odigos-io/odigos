package instrumentationconfig

import (
	"context"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	rulesv1alpha1 "github.com/odigos-io/odigos/api/rules/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func calcInstrumentationConfigForWorkload(workload workload.PodWorkload, payloadcollectionrules []rulesv1alpha1.PayloadCollection) error {
	return nil
}

func getAllInstrumentedWorkloads(ctx context.Context, client client.Client) ([]workload.PodWorkload, error) {

	logger := log.FromContext(ctx)

	instrumentationConfigsList := &odigosv1alpha1.InstrumentationConfigList{}
	err := client.List(ctx, instrumentationConfigsList)
	if err != nil {
		return nil, err
	}

	workloads := make([]workload.PodWorkload, 0, len(instrumentationConfigsList.Items))
	for _, instConfig := range instrumentationConfigsList.Items {
		workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(instConfig.Name)
		if err != nil {
			logger.Error(err, "error extracting workload info from runtime object name. skipping it", "name", instConfig.Name)
			continue
		}

		workloads = append(workloads, workload.PodWorkload{
			Name:      workloadName,
			Kind:      workloadKind,
			Namespace: instConfig.Namespace,
		})
	}

	return workloads, nil
}
