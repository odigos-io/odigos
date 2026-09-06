package recommendations

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
)

type RecommendationsSyncReconciler struct {
	client.Client
}

func (r *RecommendationsSyncReconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	if err := syncRecommendations(ctx, r.Client); err != nil {
		return k8sutils.K8SNoEffectiveConfigErrorHandler(err)
	}
	return ctrl.Result{}, nil
}
