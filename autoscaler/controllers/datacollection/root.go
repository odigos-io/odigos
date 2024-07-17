package datacollection

import (
	"context"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
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
	var collectorGroups odigosv1.CollectorsGroupList
	if err := c.List(ctx, &collectorGroups); err != nil {
		logger.Error(err, "Failed to list collectors groups")
		return err
	}

	var dataCollectionCollectorGroup *odigosv1.CollectorsGroup
	for _, collectorGroup := range collectorGroups.Items {
		if collectorGroup.Spec.Role == odigosv1.CollectorsGroupRoleNodeCollector {
			dataCollectionCollectorGroup = &collectorGroup
			break
		}
	}

	if dataCollectionCollectorGroup == nil {
		logger.V(3).Info("Data collection collector group doesn't exist, nothing to sync")
		return nil
	}

	var instApps odigosv1.InstrumentedApplicationList
	if err := c.List(ctx, &instApps); err != nil {
		logger.Error(err, "Failed to list instrumented apps")
		return err
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

	return syncDataCollection(&instApps, &dests, &processors, dataCollectionCollectorGroup, ctx, c, scheme, imagePullSecrets, odigosVersion)
}

func syncDataCollection(instApps *odigosv1.InstrumentedApplicationList, dests *odigosv1.DestinationList, processors *odigosv1.ProcessorList,
	dataCollection *odigosv1.CollectorsGroup, ctx context.Context, c client.Client,
	scheme *runtime.Scheme, imagePullSecrets []string, odigosVersion string) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Syncing data collection")

	_, err := syncConfigMap(instApps, dests, processors, dataCollection, ctx, c, scheme)
	if err != nil {
		logger.Error(err, "Failed to sync config map")
		return err
	}

	dm.RunSyncDaemonSetWithDelayAndSkipNewCalls(time.Duration(env.GetSyncDaemonSetDelay())*time.Second, syncDaemonsetRetry, dests, dataCollection, ctx, c, scheme, imagePullSecrets, odigosVersion)

	return nil
}
