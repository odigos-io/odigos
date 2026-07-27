package nodecollector

import (
	"context"

	v1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type ProcessorReconciler struct {
	nodeCollectorBaseReconciler
}

func (r *ProcessorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := commonlogger.FromContext(ctx)
	logger.Info("Reconciling Processor")

	var processor v1.Processor
	err := r.Get(ctx, req.NamespacedName, &processor)
	if err == nil {
		// TODO(remove by 2027-01): temporary cleanup of Processor CRs for actions migrated
		// to odigosConfigExtension. Delete once leftover CRs from older versions are gone.
		if commonconf.IsLegacyConfigExtensionProcessorType(processor.Spec.Type) {
			if delErr := r.Delete(ctx, &processor); delErr != nil && !apierrors.IsNotFound(delErr) {
				return ctrl.Result{}, delErr
			}
		}
	} else if !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	return r.reconcileNodeCollector(ctx)
}

func (r *ProcessorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Processor{}).
		// auto scaler only cares about the spec of each processor.
		// filter out events on resource status and metadata changes.
		WithEventFilter(&predicate.GenerationChangedPredicate{}).
		Complete(r)
}
