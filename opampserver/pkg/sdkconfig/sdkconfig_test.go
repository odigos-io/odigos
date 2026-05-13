package sdkconfig

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configsections"
	"github.com/stretchr/testify/require"
)

func TestGetFullConfigIncludesLegacySDKSection(t *testing.T) {
	manager := &SdkConfigManager{}
	resourceAttributes := []configresolvers.ResourceAttribute{
		{Key: "service.name", Value: "checkout"},
		{Key: "k8s.namespace.name", Value: "shop"},
	}
	instrumentationConfig := &odigosv1.InstrumentationConfig{
		Spec: odigosv1.InstrumentationConfigSpec{
			Containers: []odigosv1.ContainerAgentConfig{
				{
					ContainerName: "app",
					Traces:        &odigosv1.AgentTracesConfig{},
					Logs:          &odigosv1.AgentLogsConfig{},
				},
			},
		},
	}

	remoteConfig, err := manager.GetFullConfig(
		context.Background(),
		resourceAttributes,
		&k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "checkout"},
		"deployment-checkout",
		"python",
		instrumentationConfig,
		"app",
	)
	require.NoError(t, err)

	sdkConfigFile := remoteConfig.Config.ConfigMap[string(configsections.RemoteConfigSdkConfigSectionName)]
	require.NotNil(t, sdkConfigFile)

	var sdkConfig configsections.RemoteConfigSdk
	require.NoError(t, json.Unmarshal(sdkConfigFile.Body, &sdkConfig))
	require.True(t, sdkConfig.TraceSignal.Enabled)
	require.True(t, sdkConfig.LogsSignal.Enabled)
	require.False(t, sdkConfig.MetricsSignal.Enabled)
	require.Equal(t, resourceAttributes, sdkConfig.RemoteResourceAttributes)

	require.NotNil(t, remoteConfig.Config.ConfigMap[string(configsections.RemoteConfigContainerConfigSectionName)])
}
