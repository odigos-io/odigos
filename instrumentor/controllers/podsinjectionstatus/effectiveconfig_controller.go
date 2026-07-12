package podsinjectionstatus

import (
	"context"
	"errors"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EffectiveConfigReconciler struct {
	client.Client
}

func (r *EffectiveConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_, err := utils.GetCurrentOdigosConfiguration(ctx, r.Client)
	if err != nil {
		return utils.K8SNoEffectiveConfigErrorHandler(err)
	}

	allInstrumentationConfigs := odigosv1.InstrumentationConfigList{}
	err = r.Client.List(ctx, &allInstrumentationConfigs)
	if err != nil {
		return ctrl.Result{}, err
	}

	var allErrs error
	for _, ic := range allInstrumentationConfigs.Items {
		pw, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(ic.Name, ic.Namespace)
		if err != nil {
			allErrs = errors.Join(allErrs, err)
			continue
		}

		err = syncWorkload(ctx, r.Client, pw)
		if err != nil {
			allErrs = errors.Join(allErrs, err)
		}
	}

	return utils.K8SUpdateErrorHandler(allErrs)
}
