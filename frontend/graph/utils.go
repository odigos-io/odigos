package graph

import (
	"context"
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func envVarsToModel(envVars []v1alpha1.EnvVar) []*model.EnvVar {
	modelEnvVars := make([]*model.EnvVar, len(envVars))
	for i, envVar := range envVars {
		modelEnvVars[i] = &model.EnvVar{
			Name:  envVar.Name,
			Value: envVar.Value,
		}
	}
	return modelEnvVars
}

func runtimeDetectionStatusCondition(reason *string) model.DesiredStateProgress {
	if reason == nil {
		return model.DesiredStateProgressUnknown
	}
	switch v1alpha1.RuntimeDetectionReason(*reason) {
	case v1alpha1.RuntimeDetectionReasonDetectedSuccessfully:
		return model.DesiredStateProgressSuccess
	case v1alpha1.RuntimeDetectionReasonWaitingForDetection:
		return model.DesiredStateProgressWaiting
	case v1alpha1.RuntimeDetectionReasonNoRunningPods:
		return model.DesiredStateProgressPending
	case v1alpha1.RuntimeDetectionReasonError:
		return model.DesiredStateProgressError
	}
	return model.DesiredStateProgressUnknown
}

func getWorkloadSources(ctx context.Context, namespace string, kind model.K8sResourceKind, name string) (*v1alpha1.WorkloadSources, error) {

	sources := v1alpha1.WorkloadSources{}

	workloadSources, err := kube.DefaultClient.OdigosClient.Sources(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s,%s=%s",
			k8sconsts.WorkloadNameLabel, name,
			k8sconsts.WorkloadNamespaceLabel, namespace,
			k8sconsts.WorkloadKindLabel, string(kind),
		),
	})
	if err != nil {
		return nil, err
	}
	if len(workloadSources.Items) > 1 {
		return nil, errors.New("found multiple sources for the same workload")
	}
	if len(workloadSources.Items) > 0 {
		sources.Workload = &workloadSources.Items[0]
	}

	nsSources, err := kube.DefaultClient.OdigosClient.Sources(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s,%s=%s",
			k8sconsts.WorkloadNameLabel, namespace,
			k8sconsts.WorkloadNamespaceLabel, namespace,
			k8sconsts.WorkloadKindLabel, k8sconsts.WorkloadKindNamespace,
		),
	})
	if err != nil {
		return nil, err
	}
	if len(nsSources.Items) > 1 {
		return nil, errors.New("found multiple sources for the same namespace")
	}
	if len(nsSources.Items) > 1 {
		sources.Namespace = &nsSources.Items[0]
	}

	return &sources, nil
}
