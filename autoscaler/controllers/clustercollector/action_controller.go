package clustercollector

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/status"
	addedToCollectorConfig "github.com/odigos-io/odigos/status/action/generated"
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

// syncActionAddedToCollectorConfig sets AddedToCollectorConfig (ClusterGateway) when
// this action is an odigosConfigExtension action (added or removed-because-disabled).
// Processor-backed actions get this condition from the actions controller by mirroring
// the owned Processor status; they are left untouched here.
func syncActionAddedToCollectorConfig(ctx context.Context, c client.Client, key types.NamespacedName) error {
	action := &odigosv1.Action{}
	err := c.Get(ctx, key, action)
	if err != nil {
		return err
	}

	if !commonconf.IsConfigExtension(action) {
		return nil
	}

	if action.Spec.Disabled {
		return setAddedToCollectorConfigReason(ctx, c, action, addedToCollectorConfig.AddedToCollectorConfigConfigRemovedDisabled_ClusterGateway)
	}
	return setAddedToCollectorConfigReason(ctx, c, action, addedToCollectorConfig.AddedToCollectorConfigConfigUpdated_ClusterGateway)
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
