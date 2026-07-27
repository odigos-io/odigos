package agentenabled

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	actionutil "github.com/odigos-io/odigos/k8sutils/pkg/action"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/status"
	addedToSourcesConfig "github.com/odigos-io/odigos/status/action/generated"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ActionReconciler struct {
	client.Client
	DistrosProvider           *distros.Provider
	RolloutConcurrencyLimiter *rollout.RolloutConcurrencyLimiter
}

func (r *ActionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Load the Action before reconcileAll so status is updated from the same
	// object generation that drove source config reconciliation.
	action := &odigosv1.Action{}
	err := r.Get(ctx, req.NamespacedName, action)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		// Action deleted: still reconcile sources so its config is removed.
		action = nil
	}

	result, err := reconcileAll(ctx, r.Client, r.DistrosProvider, r.RolloutConcurrencyLimiter)
	if err != nil {
		return utils.K8SUpdateErrorHandler(err)
	}
	// Sync status even when reconcileAll asks to requeue (e.g. rollout wait):
	// InstrumentationConfigs are already updated before rollout requeue.
	if action != nil {
		if syncErr := syncActionAddedToSourcesConfig(ctx, r.Client, action); syncErr != nil {
			return utils.K8SUpdateErrorHandler(syncErr)
		}
	}
	return result, nil
}

// syncActionAddedToSourcesConfig sets AddedToSourcesConfig when this action is a
// config-extension action that drives source InstrumentationConfig (added or
// removed-because-disabled), otherwise clears a stale condition if present.
func syncActionAddedToSourcesConfig(ctx context.Context, c client.Client, action *odigosv1.Action) error {
	if !actionutil.IsConfigExtension(action) {
		return clearAddedToSourcesConfig(ctx, c, action)
	}

	if action.Spec.Disabled {
		return setAddedToSourcesConfigReason(ctx, c, action, addedToSourcesConfig.AddedToSourcesConfigConfigRemovedDisabled)
	}
	return setAddedToSourcesConfigReason(ctx, c, action, addedToSourcesConfig.AddedToSourcesConfigConfigUpdated)
}

func setAddedToSourcesConfigReason(ctx context.Context, c client.Client, action *odigosv1.Action, reason status.Reason) error {
	message, _ := status.RenderMessage(reason, nil)
	return odgiosK8s.UpdateStatusConditions(ctx, c, action, &action.Status.Conditions,
		reason.K8sConditionStatus,
		addedToSourcesConfig.AddedToSourcesConfigType,
		reason.Name,
		message,
	)
}

func clearAddedToSourcesConfig(ctx context.Context, c client.Client, action *odigosv1.Action) error {
	if meta.FindStatusCondition(action.Status.Conditions, addedToSourcesConfig.AddedToSourcesConfigType) == nil {
		return nil
	}
	if !meta.RemoveStatusCondition(&action.Status.Conditions, addedToSourcesConfig.AddedToSourcesConfigType) {
		return nil
	}
	return c.Status().Update(ctx, action)
}
