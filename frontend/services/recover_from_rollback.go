package services

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RecoverFromRollback sets RecoveredFromRollbackAt on the workload's Source with the current timestamp.
// It flows through the sourceinstrumentation controller to the InstrumentationConfig spec.
// The agentenabled controller compares spec vs annotation timestamps: if they differ, it clears
// RollbackOccurred and copies the timestamp to the annotation, allowing a retry.
func RecoverFromRollback(ctx context.Context, kubeClient client.Client, namespace, workloadName, kind string) error {
	podWorkload := k8sconsts.PodWorkload{
		Name:      workloadName,
		Namespace: namespace,
		Kind:      k8sconsts.WorkloadKind(kind),
	}
	sources, err := odigosv1alpha1.GetSources(ctx, kubeClient, podWorkload)
	if err != nil {
		return fmt.Errorf("failed to get sources for workload %s/%s/%s: %w", namespace, kind, workloadName, err)
	}

	if sources.Workload == nil {
		return fmt.Errorf("no workload-level Source found for %s/%s/%s", namespace, kind, workloadName)
	}

	now := metav1.NewTime(time.Now())
	sources.Workload.Spec.RecoveredFromRollbackAt = &now
	if err := kubeClient.Update(ctx, sources.Workload); err != nil {
		return fmt.Errorf("failed to update Source with RecoveredFromRollbackAt: %w", err)
	}

	return nil
}
