package graph

import (
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

func TestBuildUpdatedContainerOverridesPreservesAllowConcurrentAgents(t *testing.T) {
	allow := true
	distro := "nodejs-community"
	existing := []v1alpha1.ContainerOverride{
		{
			ContainerName:         "app",
			AllowConcurrentAgents: &allow,
			OtelDistroName:        &distro,
		},
		{
			ContainerName: "sidecar",
		},
	}

	updated := buildUpdatedContainerOverrides(existing, "app", nil, &v1alpha1.RuntimeDetailsByContainer{
		ContainerName:  "app",
		Language:       common.JavascriptProgrammingLanguage,
		RuntimeVersion: "20.0.0",
	})

	require.Len(t, updated, 2)
	require.Equal(t, "sidecar", updated[0].ContainerName)

	app := updated[1]
	require.Equal(t, "app", app.ContainerName)
	require.NotNil(t, app.AllowConcurrentAgents)
	require.True(t, *app.AllowConcurrentAgents)
	require.NotNil(t, app.RuntimeInfo)
	require.Equal(t, common.JavascriptProgrammingLanguage, app.RuntimeInfo.Language)
	require.Nil(t, app.OtelDistroName)
}

func TestBuildUpdatedContainerOverridesLeavesNilWhenUnset(t *testing.T) {
	updated := buildUpdatedContainerOverrides(nil, "app", nil, nil)
	require.Len(t, updated, 1)
	require.Nil(t, updated[0].AllowConcurrentAgents)
}
