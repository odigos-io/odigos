package clustercollector

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	actionutil "github.com/odigos-io/odigos/k8sutils/pkg/action"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/status"
	addedToCollectorConfig "github.com/odigos-io/odigos/status/action/generated"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ActionReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	OdigosVersion string
}

func (r *ActionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	result, err := reconcileClusterCollector(ctx, r.Client, r.Scheme, r.OdigosVersion)
	if err != nil {
		return result, err
	}
	if !result.IsZero() {
		return result, nil
	}

	// sync the AddedToCollectorConfig condition for this action after a successful
	// cluster collector sync. usually a no-op when nothing changed.
	return utils.K8SUpdateErrorHandler(syncActionAddedToCollectorConfig(ctx, r.Client, req.NamespacedName))
}

// syncActionAddedToCollectorConfig sets AddedToCollectorConfig when this action is an
// odigosConfigExtension action (added or removed-because-disabled). The collector role
// matches ConvertActionsToConfigExtensionProcessors (SQL moves to the node collector
// when span metrics are enabled).
// Processor-backed actions get this condition from the actions controller by mirroring
// the owned Processor status; they are left untouched here.
func syncActionAddedToCollectorConfig(ctx context.Context, c client.Client, key types.NamespacedName) error {
	action := &odigosv1.Action{}
	err := c.Get(ctx, key, action)
	if err != nil {
		return err
	}

	if !actionutil.IsConfigExtension(action) {
		return nil
	}

	spanMetricsEnabled, err := nodeSpanMetricsEnabled(ctx, c)
	if err != nil {
		return err
	}
	return setConfigExtensionActionAddedToCollectorConfig(ctx, c, action, spanMetricsEnabled)
}

func syncAllConfigExtensionActionStatuses(ctx context.Context, c client.Client, actionList odigosv1.ActionList, spanMetricsEnabled bool) error {
	for i := range actionList.Items {
		action := &actionList.Items[i]
		if !actionutil.IsConfigExtension(action) {
			continue
		}
		if err := setConfigExtensionActionAddedToCollectorConfig(ctx, c, action, spanMetricsEnabled); err != nil {
			return err
		}
	}
	return nil
}

func setConfigExtensionActionAddedToCollectorConfig(ctx context.Context, c client.Client, action *odigosv1.Action, spanMetricsEnabled bool) error {
	role := commonconf.ConfigExtensionActionCollectorRole(action, spanMetricsEnabled)
	updatedReason, removedReason, err := addedToCollectorConfigReasonsForRole(role)
	if err != nil {
		return err
	}
	if action.Spec.Disabled {
		return setAddedToCollectorConfigReason(ctx, c, action, removedReason)
	}
	return setAddedToCollectorConfigReason(ctx, c, action, updatedReason)
}

func addedToCollectorConfigReasonsForRole(role odigosv1.CollectorsGroupRole) (status.Reason, status.Reason, error) {
	switch role {
	case odigosv1.CollectorsGroupRoleClusterGateway:
		return addedToCollectorConfig.AddedToCollectorConfigConfigUpdated_ClusterGateway,
			addedToCollectorConfig.AddedToCollectorConfigConfigRemovedDisabled_ClusterGateway,
			nil
	case odigosv1.CollectorsGroupRoleNodeCollector:
		return addedToCollectorConfig.AddedToCollectorConfigConfigUpdated_NodeCollector,
			addedToCollectorConfig.AddedToCollectorConfigConfigRemovedDisabled_NodeCollector,
			nil
	default:
		return status.Reason{}, status.Reason{}, fmt.Errorf("unsupported collector role %q for AddedToCollectorConfig", role)
	}
}

func nodeSpanMetricsEnabled(ctx context.Context, c client.Client) (bool, error) {
	nodeCG := &odigosv1.CollectorsGroup{}
	err := c.Get(ctx, client.ObjectKey{
		Namespace: env.GetCurrentNamespace(),
		Name:      k8sconsts.OdigosNodeCollectorCollectorGroupName,
	}, nodeCG)
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return commonconf.NodeCollectorsGroupSpanMetricsEnabled(nodeCG), nil
}

func setAddedToCollectorConfigReason(ctx context.Context, c client.Client, action *odigosv1.Action, reason status.Reason) error {
	message, _ := status.RenderMessage(reason, nil)
	return odgiosK8s.UpdateStatusConditions(ctx, c, action, &action.Status.Conditions,
		reason.K8sConditionStatus,
		addedToCollectorConfig.AddedToCollectorConfigType,
		reason.Name,
		message,
	)
}
