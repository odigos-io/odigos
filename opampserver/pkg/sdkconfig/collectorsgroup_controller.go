package sdkconfig

import (
	"context"
	"slices"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configsections"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type CollectorsGroupReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	ConnectionCache *connection.ConnectionsCache
}

func (d *CollectorsGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// we are configuring the SDKs which sends data to node collectors group.
	// thus, we only care about this specific CR
	if req.Name != consts.OdigosNodeCollectorCollectorGroupName {
		return ctrl.Result{}, nil
	}

	logger := log.FromContext(ctx)

	var collectorsGroup odigosv1.CollectorsGroup
	err := d.Client.Get(ctx, req.NamespacedName, &collectorsGroup)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	tracesEnabled := slices.Contains(collectorsGroup.Status.ReceiverSignals, common.TracesObservabilitySignal)

	d.ConnectionCache.UpdateAllConnectionConfigs(func(connInfo *connection.ConnectionInfo) *protobufs.AgentConfigMap {

		remoteConfigSdk := configsections.CalcSdkRemoteConfig(connInfo.RemoteResourceAttributes, tracesEnabled)
		opampRemoteConfigSdk, sdkSectionName, err := configsections.SdkRemoteConfigToOpamp(remoteConfigSdk)
		if err != nil {
			logger.Info("failed to calculate SDK remote config", "error", err)
			return nil
		}

		return &protobufs.AgentConfigMap{
			ConfigMap: map[string]*protobufs.AgentConfigFile{
				sdkSectionName: opampRemoteConfigSdk,
			},
		}
	})

	return ctrl.Result{}, nil
}

func (i *CollectorsGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.CollectorsGroup{}).
		Complete(i)
}
