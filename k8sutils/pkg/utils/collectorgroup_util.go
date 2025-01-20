package utils

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

func ApplyCollectorGroup(ctx context.Context, c client.Client, collectorGroup *odigosv1.CollectorsGroup) error {
	logger := log.FromContext(ctx)
	logger.Info("Applying collector group", "collectorGroupName", collectorGroup.Name)

	err := c.Patch(ctx, collectorGroup, client.Apply, client.ForceOwnership, client.FieldOwner("scheduler"))
	if err != nil {
		logger.Error(err, "Failed to apply collector group")
		return err
	}

	return nil
}

func GetCollectorGroup(ctx context.Context, c client.Client, namespace string, collectorGroupName string) (*odigosv1.CollectorsGroup, error) {
	var collectorGroup odigosv1.CollectorsGroup
	err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: collectorGroupName}, &collectorGroup)

	return &collectorGroup, err
}

func DeleteCollectorGroup(ctx context.Context, c client.Client, namespace string, collectorGroupName string) error {
	logger := log.FromContext(ctx).WithValues("collectorGroupName", collectorGroupName)
	logger.Info("Deleting collector group")

	collectorGroup, err := GetCollectorGroup(ctx, c, namespace, collectorGroupName)
	if errors.IsNotFound(err) {
		logger.V(3).Info("collector group doesn't exist, nothing to delete")
		return nil
	}

	if err = c.Delete(ctx, collectorGroup, &client.DeleteOptions{}); err != nil {
		logger.Error(err, "Failed to delete collector")
		return err
	}

	return nil
}
