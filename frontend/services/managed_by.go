package services

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

// managedByFromLabels maps the odigos.io/managed-by label to the GraphQL ManagedBy enum.
func managedByFromLabels(labels map[string]string) model.ManagedBy {
	if labels == nil {
		return model.ManagedByUnknown
	}
	switch labels[k8sconsts.OdigosProfilesManagedByLabel] {
	case k8sconsts.OdigosProfilesManagedByValue:
		return model.ManagedByProfile
	case k8sconsts.OdigosUIManagedByValue:
		return model.ManagedByOdigosUI
	case k8sconsts.OdigosInterrogationLoopManagedByValue:
		return model.ManagedByInterrogationLoop
	default:
		return model.ManagedByUnknown
	}
}
