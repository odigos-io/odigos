package services

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

func TestManagedByFromLabels(t *testing.T) {
	require.Equal(t, model.ManagedByUnknown, managedByFromLabels(nil))
	require.Equal(t, model.ManagedByUnknown, managedByFromLabels(map[string]string{}))
	require.Equal(t, model.ManagedByUnknown, managedByFromLabels(map[string]string{
		k8sconsts.OdigosProfilesManagedByLabel: "helm",
	}))
	require.Equal(t, model.ManagedByProfile, managedByFromLabels(map[string]string{
		k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosProfilesManagedByValue,
	}))
	require.Equal(t, model.ManagedByOdigosUI, managedByFromLabels(map[string]string{
		k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosUIManagedByValue,
	}))
	require.Equal(t, model.ManagedByInterrogationLoop, managedByFromLabels(map[string]string{
		k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosInterrogationLoopManagedByValue,
	}))
}
