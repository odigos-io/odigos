package agentenabled

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/status"
	addedToSourcesConfig "github.com/odigos-io/odigos/status/action/generated"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ActionReconciler struct {
	client.Client
	DistrosProvider           *distros.Provider
	RolloutConcurrencyLimiter *rollout.RolloutConcurrencyLimiter
}

func (r *ActionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// This reconciler is fired everytime an agent-level action (URLTemplatization,
	// SpanRenamer, or config-extension actions that update sources config) is
	// created, updated or deleted. Reconcile all workloads so InstrumentationConfigs
	// pick up the change, then sync AddedToSourcesConfig for config-ext actions.
	result, err := reconcileAll(ctx, r.Client, r.DistrosProvider, r.RolloutConcurrencyLimiter)
	if err != nil {
		return utils.K8SUpdateErrorHandler(err)
	}
	if !result.IsZero() {
		return result, nil
	}
	return utils.K8SUpdateErrorHandler(syncActionAddedToSourcesConfig(ctx, r.Client, req.NamespacedName))
}

// syncActionAddedToSourcesConfig sets AddedToSourcesConfig when this action is a
// config-extension action that drives source InstrumentationConfig (added or
// removed-because-disabled), otherwise clears a stale condition if present.
func syncActionAddedToSourcesConfig(ctx context.Context, c client.Client, key types.NamespacedName) error {
	action := &odigosv1.Action{}
	err := c.Get(ctx, key, action)
	if err != nil {
		return err
	}

	if !isConfigExtensionSourcesAction(action) {
		return clearAddedToSourcesConfig(ctx, c, action)
	}

	if action.Spec.Disabled {
		return setAddedToSourcesConfigReason(ctx, c, action, addedToSourcesConfig.AddedToSourcesConfigConfigRemovedDisabled)
	}
	return setAddedToSourcesConfigReason(ctx, c, action, addedToSourcesConfig.AddedToSourcesConfigConfigUpdated)
}

func isConfigExtensionSourcesAction(action *odigosv1.Action) bool {
	return action.Spec.PiiMasking != nil ||
		action.Spec.DbQueryTemplatization != nil ||
		action.Spec.InferDbAttributes != nil
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
