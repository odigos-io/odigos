package actions

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconsts "github.com/odigos-io/odigos/common/consts"
)

// mapUrlTemplatizationProcessorToActionRequests maps a URL-templatization Processor event to a single
// namespace-level sync request. We enqueue only the synthetic key so one reconcile runs, lists Actions
// once, and creates or deletes the Processor. Enqueueing every related Action would cause N reconciles
// and N redundant lists for the same outcome.
func mapUrlTemplatizationProcessorToActionRequests(ctx context.Context, _ client.Client, obj client.Object) []reconcile.Request {
	if obj.GetName() != commonconsts.URLTemplatizationProcessorName {
		return nil
	}
	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			Namespace: obj.GetNamespace(),
			Name:      urlTemplatizationNamespaceSyncKey,
		},
	}}
}

func SetupWithManager(mgr ctrl.Manager) error {
	// processorToActionRequests maps Processor CR events to a single namespace-level sync request.
	// One reconcile runs, lists Actions once, and creates or deletes the shared Processor as needed.
	processorToActionRequests := func(ctx context.Context, obj client.Object) []reconcile.Request {
		return mapUrlTemplatizationProcessorToActionRequests(ctx, nil, obj)
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.Action{}).
		WithEventFilter(&predicate.GenerationChangedPredicate{}).
		// Watch only the shared URL-templatization Processor CR (by name) to avoid running the map
		// for every Processor in the cluster. When it changes, enqueue namespace-level sync.
		Watches(&odigosv1.Processor{}, handler.EnqueueRequestsFromMapFunc(processorToActionRequests),
			builder.WithPredicates(
				predicate.And(
					&predicate.GenerationChangedPredicate{},
					predicate.NewPredicateFuncs(func(object client.Object) bool {
						return object.GetName() == commonconsts.URLTemplatizationProcessorName
					}),
				),
			)).
		Complete(&ActionReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&v1.AddClusterInfo{}).
		Complete(&AddClusterInfoReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&v1.DeleteAttribute{}).
		Complete(&DeleteAttributeReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&v1.RenameAttribute{}).
		Complete(&RenameAttributeReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&v1.ProbabilisticSampler{}).
		Complete(&ProbabilisticSamplerReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}
	err = ctrl.NewControllerManagedBy(mgr).
		For(&v1.LatencySampler{}).
		Complete(&OdigosSamplingReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&v1.SpanAttributeSampler{}).
		Complete(&OdigosSamplingReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&v1.ServiceNameSampler{}).
		Complete(&OdigosSamplingReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&v1.ErrorSampler{}).
		Complete(&OdigosSamplingReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&v1.PiiMasking{}).
		Complete(&PiiMaskingReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&v1.K8sAttributesResolver{}).
		Complete(&K8sAttributesResolverReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	return nil
}

func RegisterWebhooks(mgr ctrl.Manager) error {
	err := builder.WebhookManagedBy(mgr).
		For(&odigosv1.Action{}).
		WithValidator(&ActionsValidator{}).
		Complete()
	if err != nil {
		return err
	}

	return nil
}
