package sdkconfig

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configsections"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DestinationReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	ConnectionCache *connection.ConnectionsCache
}

func (d *DestinationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	// when there is a change in the destination CRD, we need to update the SDK configs
	// to reflect a potential change in the enabled signals
	tracesEnabled, _, err := configresolvers.CalcEnabledSignals(ctx, d.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

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

func (i *DestinationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.Destination{}).
		Complete(i)
}
