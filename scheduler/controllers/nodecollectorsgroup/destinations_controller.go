package nodecollectorsgroup

import (
	"context"

	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DestinationsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DestinationsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	err := sync(ctx, r.Client, r.Scheme)
	return utils.K8SNoEffectiveConfigErrorHandler(err)
}
