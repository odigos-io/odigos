package actions

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	crpredicate "sigs.k8s.io/controller-runtime/pkg/predicate"
)

// URLTemplateNodeCGReconciler watches the node CollectorsGroup (e.g. span metrics toggled) and re-syncs
// the shared URL-templatization Processor. Uses URLTemplatizationSyncApplyFull so roles are patched
// even when the Processor CR already exists (unlike Action reconcile with URLTemplatizationSyncCreateIfMissing).
type URLTemplateNodeCGReconciler struct {
	client.Client
}

func (r *URLTemplateNodeCGReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := commonlogger.FromContext(ctx)
	if err := SyncUrlTemplatizationProcessor(ctx, r.Client, URLTemplatizationSyncApplyFull); err != nil {
		logger.Error(err, "sync URL-templatization processor after node CollectorsGroup change failed")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// urlTemplateNodeCGSpanMetricsTogglePredicate matches SyncUrlTemplatizationProcessor: only span metrics
// on/off (Metrics.SpanMetrics nil vs non-nil) affects shared Processor collector roles.
type urlTemplateNodeCGSpanMetricsTogglePredicate struct{}

func (urlTemplateNodeCGSpanMetricsTogglePredicate) Create(event.CreateEvent) bool {
	return true
}

func (urlTemplateNodeCGSpanMetricsTogglePredicate) Delete(event.DeleteEvent) bool {
	return true
}

func (urlTemplateNodeCGSpanMetricsTogglePredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}
	oldCG, ok1 := e.ObjectOld.(*odigosv1.CollectorsGroup)
	newCG, ok2 := e.ObjectNew.(*odigosv1.CollectorsGroup)
	if !ok1 || !ok2 {
		return true
	}
	oldOn := oldCG.Spec.Metrics != nil && oldCG.Spec.Metrics.SpanMetrics != nil
	newOn := newCG.Spec.Metrics != nil && newCG.Spec.Metrics.SpanMetrics != nil
	return oldOn != newOn
}

func (urlTemplateNodeCGSpanMetricsTogglePredicate) Generic(event.GenericEvent) bool {
	return false
}

var _ crpredicate.Predicate = urlTemplateNodeCGSpanMetricsTogglePredicate{}
