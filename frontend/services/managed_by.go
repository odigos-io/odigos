package services

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

// ManagedByFromLabels maps the odigos.io/managed-by label to the GraphQL ManagedBy enum.
func ManagedByFromLabels(labels map[string]string) model.ManagedBy {
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
	case k8sconsts.OdigosLiveTrafficLearningManagedByValue:
		return model.ManagedByLiveTrafficLearning
	default:
		return model.ManagedByUnknown
	}
}

// managedByFromLabels is kept for in-package call sites.
func managedByFromLabels(labels map[string]string) model.ManagedBy {
	return ManagedByFromLabels(labels)
}
