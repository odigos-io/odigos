package nodecollector

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
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

	result, err := r.reconcileNodeCollector(ctx)
	if err != nil {
		return utils.K8SUpdateErrorHandler(err)
	}
	if !result.IsZero() {
		return result, nil
	}

	// sync the AddedToCollectorConfig condition for this processor after a successful
	// node collector sync. usually a no-op when nothing changed.
	return utils.K8SUpdateErrorHandler(commonconf.SyncProcessorAddedToCollectorConfig(
		ctx, r.Client, req.NamespacedName, odigosv1.CollectorsGroupRoleNodeCollector,
	))
}

func (r *ProcessorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.Processor{}).
		// auto scaler only cares about the spec of each processor.
		// filter out events on resource status and metadata changes.
		WithEventFilter(&predicate.GenerationChangedPredicate{}).
		Complete(r)
}
