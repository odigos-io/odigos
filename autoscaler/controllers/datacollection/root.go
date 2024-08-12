package datacollection

import (
	"context"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var dm = &DelayManager{}

const (
	syncDaemonsetRetry = 3
)

func Sync(ctx context.Context, c client.Client, scheme *runtime.Scheme, imagePullSecrets []string, odigosVersion string) error {
	logger := log.FromContext(ctx)

	var instApps odigosv1.InstrumentedApplicationList
	if err := c.List(ctx, &instApps); err != nil {
		logger.Error(err, "Failed to list instrumented apps")
		return err
	}

	if len(instApps.Items) == 0 {
		logger.V(3).Info("No instrumented apps")
		return nil
	}

	odigosNs := env.GetCurrentNamespace()
	var dataCollectionCollectorGroup odigosv1.CollectorsGroup
	err := c.Get(ctx, client.ObjectKey{Namespace: odigosNs, Name: consts.OdigosNodeCollectorCollectorGroupName}, &dataCollectionCollectorGroup)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	var dests odigosv1.DestinationList
	if err := c.List(ctx, &dests); err != nil {
		logger.Error(err, "Failed to list destinations")
		return err
	}

	var processors odigosv1.ProcessorList
	if err := c.List(ctx, &processors); err != nil {
		logger.Error(err, "Failed to list processors")
		return err
	}

	return syncDataCollection(&instApps, &dests, &processors, &dataCollectionCollectorGroup, ctx, c, scheme, imagePullSecrets, odigosVersion)
}

func syncDataCollection(instApps *odigosv1.InstrumentedApplicationList, dests *odigosv1.DestinationList, processors *odigosv1.ProcessorList,
	dataCollection *odigosv1.CollectorsGroup, ctx context.Context, c client.Client,
	scheme *runtime.Scheme, imagePullSecrets []string, odigosVersion string) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Syncing data collection")

	_, err := SyncConfigMap(instApps, dests, processors, dataCollection, ctx, c, scheme)
	if err != nil {
		logger.Error(err, "Failed to sync config map")
		return err
	}

	dm.RunSyncDaemonSetWithDelayAndSkipNewCalls(time.Duration(env.GetSyncDaemonSetDelay())*time.Second, syncDaemonsetRetry, dests, dataCollection, ctx, c, scheme, imagePullSecrets, odigosVersion)

	return nil
}
