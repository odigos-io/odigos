package actions

import (
	"context"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SharedURLTemplatizationProcessorReconciler watches the single shared URL-templatization Processor CR
// (name from consts.URLTemplatizationProcessorName). When that object is created, updated, or deleted,
// it runs SyncUrlTemplatizationProcessor with URLTemplatizationSyncApplyFull so desired state is
// recomputed from all Actions plus the node CollectorsGroup (e.g. span metrics). That corrects drift
// from manual edits to the Processor and edge cases where the Action controller did not run (e.g.
// last URL-templatization Action removed while the autoscaler was unavailable); delete-when-empty is
// still primarily enforced via Action reconcile.
type SharedURLTemplatizationProcessorReconciler struct {
	client.Client
}

func (r *SharedURLTemplatizationProcessorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := commonlogger.FromContext(ctx)
	if err := SyncUrlTemplatizationProcessor(ctx, r.Client, URLTemplatizationSyncApplyFull); err != nil {
		logger.Error(err, "Sync shared URL-templatization processor failed")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
