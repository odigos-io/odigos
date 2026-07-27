package clustercollector

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ProcessorReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	OdigosVersion string
}

func (r *ProcessorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	var processor odigosv1.Processor
	err := r.Get(ctx, req.NamespacedName, &processor)
	if err == nil {
		// TODO(remove by 2027-01): temporary cleanup of Processor CRs for actions migrated
		// to odigosConfigExtension. Delete once leftover CRs from older versions are gone.
		if commonconf.IsLegacyConfigExtensionProcessorType(processor.Spec.Type) {
			if delErr := r.Delete(ctx, &processor); delErr != nil {
				return utils.K8SUpdateErrorHandler(delErr)
			}
		}
	} else if !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	result, err := reconcileClusterCollector(ctx, r.Client, r.Scheme, r.OdigosVersion)
	if !result.IsZero() {
		return result, nil
	}
	return utils.K8SUpdateErrorHandler(err)
}
