package services

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RecoverFromRollback sets the RollbackRecoveryAtAnnotation on the workload's Source.
// The sourceinstrumentation controller propagates this annotation to the InstrumentationConfig.
// The agentenabled controller compares it with the processed annotation: if they differ,
// it clears RollbackOccurred and updates the processed annotation, allowing a retry.
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

	// If we didn't find any sources, create an empty one so we can update it and retry instrumentation.
	if sources.Workload == nil {
		sources.Workload = &odigosv1alpha1.Source{
			ObjectMeta: metav1.ObjectMeta{
				Name:      workloadName,
				Namespace: namespace,
			},
			Spec: odigosv1alpha1.SourceSpec{
				Workload: podWorkload,
			},
		}
		if err := kubeClient.Create(ctx, sources.Workload); err != nil {
			return fmt.Errorf("failed to create Source for %s/%s/%s: %w", namespace, kind, workloadName, err)
		}
		return nil
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Re-fetches the sources to avoid conflicts with other controllers.
		latestSources, err := odigosv1alpha1.GetSources(ctx, kubeClient, podWorkload)
		if err != nil {
			return fmt.Errorf("failed to get sources for workload %s/%s/%s: %w", namespace, kind, workloadName, err)
		}
		if latestSources.Workload == nil {
			return fmt.Errorf("source not found for workload %s/%s/%s during retry", namespace, kind, workloadName)
		}

		// Update the annotation and spec timestamp to record the recovery.
		now := metav1.NewTime(time.Now())
		if latestSources.Workload.Annotations == nil {
			latestSources.Workload.Annotations = make(map[string]string)
		}
		latestSources.Workload.Annotations[k8sconsts.RollbackRecoveryAtAnnotation] = now.Format(time.RFC3339)

		return kubeClient.Update(ctx, latestSources.Workload)
	})

	return err
}
