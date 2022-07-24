package datacollection

import (
	"context"
	odigosv1 "github.com/keyval-dev/odigos/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func Sync(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	logger := log.FromContext(ctx)
	var collectorGroups odigosv1.CollectorsGroupList
	if err := c.List(ctx, &collectorGroups); err != nil {
		logger.Error(err, "failed to list collectors groups")
		return err
	}

	var dataCollectionCollectorGroup *odigosv1.CollectorsGroup
	for _, collectorGroup := range collectorGroups.Items {
		if collectorGroup.Spec.Role == odigosv1.CollectorsGroupRoleDataCollection {
			dataCollectionCollectorGroup = &collectorGroup
			break
		}
	}

	if dataCollectionCollectorGroup == nil {
		logger.V(3).Info("data collection collector group not exists, nothing to sync")
		return nil
	}

	var instApps odigosv1.InstrumentedApplicationList
	if err := c.List(ctx, &instApps); err != nil {
		logger.Error(err, "failed to list instrumented apps")
		return err
	}

	var dests odigosv1.DestinationList
	if err := c.List(ctx, &dests); err != nil {
		logger.Error(err, "failed to list destinations")
		return err
	}

	return syncDataCollection(&instApps, &dests, dataCollectionCollectorGroup, ctx, c, scheme)
}

func syncDataCollection(instApps *odigosv1.InstrumentedApplicationList, dests *odigosv1.DestinationList,
	dataCollection *odigosv1.CollectorsGroup, ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("syncing data collection")

	_, err := syncConfigMap(instApps, dests, dataCollection, ctx, c, scheme)
	if err != nil {
		logger.Error(err, "failed to sync config map")
		return err
	}

	return nil
}
