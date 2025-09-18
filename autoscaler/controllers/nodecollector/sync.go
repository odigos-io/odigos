package nodecollector

import (
	"context"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var dm = &DelayManager{}

const (
	syncDaemonsetRetry = 3
)

func reconcileNodeCollector(ctx context.Context, c client.Client, scheme *runtime.Scheme, imagePullSecrets []string, odigosVersion string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var ics odigosv1.InstrumentationConfigList
	if err := c.List(ctx, &ics); err != nil {
		return ctrl.Result{}, err
	}

	odigosNs := env.GetCurrentNamespace()

	dataCollectionCollectorGroup := new(odigosv1.CollectorsGroup)
	err := c.Get(ctx, client.ObjectKey{Namespace: odigosNs, Name: k8sconsts.OdigosNodeCollectorCollectorGroupName}, dataCollectionCollectorGroup)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		dataCollectionCollectorGroup = nil
	}

	var clusterCollectorCollectorGroup odigosv1.CollectorsGroup
	err = c.Get(ctx, client.ObjectKey{Namespace: odigosNs, Name: k8sconsts.OdigosClusterCollectorConfigMapName}, &clusterCollectorCollectorGroup)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var processors odigosv1.ProcessorList
	if err := c.List(ctx, &processors); err != nil {
		logger.Error(err, "Failed to list processors")
		return ctrl.Result{}, err
	}

	err = syncDataCollection(&ics, clusterCollectorCollectorGroup.Status.ReceiverSignals, &processors, dataCollectionCollectorGroup, ctx, c, scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func syncDataCollection(sources *odigosv1.InstrumentationConfigList, signals []common.ObservabilitySignal, processors *odigosv1.ProcessorList,
	dataCollection *odigosv1.CollectorsGroup, ctx context.Context, c client.Client,
	scheme *runtime.Scheme) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Syncing data collection")

	err := syncService(ctx, c, scheme, dataCollection)
	if err != nil {
		logger.Error(err, "Failed to sync service")
		return err
	}

	err = SyncConfigMap(sources, signals, processors, dataCollection, ctx, c, scheme)
	if err != nil {
		logger.Error(err, "Failed to sync config map")
		return err
	}

	dm.RunSyncDaemonSetWithDelayAndSkipNewCalls(time.Duration(env.GetSyncDaemonSetDelay())*time.Second, syncDaemonsetRetry, signals, dataCollection, ctx, c)

	return nil
}
