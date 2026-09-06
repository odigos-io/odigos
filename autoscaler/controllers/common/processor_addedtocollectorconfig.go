package common

import (
	"context"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	"github.com/odigos-io/odigos/status"
	addedToCollectorConfig "github.com/odigos-io/odigos/status/processor/generated"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SyncProcessorAddedToCollectorConfig sets AddedToCollectorConfig for the given collector
// role when this processor targets that role (added or removed-because-disabled),
// otherwise removes a stale condition for that role if present.
func SyncProcessorAddedToCollectorConfig(ctx context.Context, c client.Client, key types.NamespacedName, role odigosv1.CollectorsGroupRole) error {
	updatedReason, removedReason, err := addedToCollectorConfigReasonsForRole(role)
	if err != nil {
		return err
	}

	processor := &odigosv1.Processor{}
	if err := c.Get(ctx, key, processor); err != nil {
		return err
	}

	if processorHasCollectorRole(processor, role) {
		if processor.Spec.Disabled {
			return setProcessorAddedToCollectorConfigReason(ctx, c, processor, removedReason)
		}
		return setProcessorAddedToCollectorConfigReason(ctx, c, processor, updatedReason)
	}

	// Processor is not for this role — clear a leftover condition for this role so
	// status does not stay stale after a role change. Leave other roles' reasons alone.
	cond := meta.FindStatusCondition(processor.Status.Conditions, addedToCollectorConfig.AddedToCollectorConfigType)
	if cond == nil {
		return nil
	}
	if cond.Reason != updatedReason.Name && cond.Reason != removedReason.Name {
		return nil
	}
	changed := meta.RemoveStatusCondition(&processor.Status.Conditions, addedToCollectorConfig.AddedToCollectorConfigType)
	if !changed {
		return nil
	}
	return c.Status().Update(ctx, processor)
}

func addedToCollectorConfigReasonsForRole(role odigosv1.CollectorsGroupRole) (updated, removed status.Reason, err error) {
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

func setProcessorAddedToCollectorConfigReason(ctx context.Context, c client.Client, processor *odigosv1.Processor, reason status.Reason) error {
	message, _ := status.RenderMessage(reason, nil)
	return odgiosK8s.UpdateStatusConditions(ctx, c, processor, &processor.Status.Conditions,
		reason.K8sConditionStatus,
		addedToCollectorConfig.AddedToCollectorConfigType,
		reason.Name,
		message,
	)
}

func processorHasCollectorRole(processor *odigosv1.Processor, role odigosv1.CollectorsGroupRole) bool {
	for _, r := range processor.Spec.CollectorRoles {
		if r == role {
			return true
		}
	}
	return false
}
