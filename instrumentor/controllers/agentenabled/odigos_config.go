package agentenabled

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
)

type odigosConfigurationController struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *odigosConfigurationController) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	odigosConfiguration, err := k8sutils.GetCurrentOdigosConfiguration(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	CommonOdigosConfiguration.ImagePullSecrets = odigosConfiguration.ImagePullSecrets

	return ctrl.Result{}, err
}
