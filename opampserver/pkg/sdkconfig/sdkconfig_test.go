package sdkconfig

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configsections"
	"github.com/stretchr/testify/require"
)

func TestGetFullConfigIncludesLegacySDKSection(t *testing.T) {
	manager := SdkConfigManager{logger: logr.Discard()}
	remoteResourceAttributes := []configresolvers.ResourceAttribute{
		{Key: "service.name", Value: "checkout"},
	}
	instrumentationConfig := &odigosv1.InstrumentationConfig{
		Spec: odigosv1.InstrumentationConfigSpec{
			Containers: []odigosv1.ContainerAgentConfig{
				{
					ContainerName: "app",
					Traces:        &odigosv1.AgentTracesConfig{},
					Metrics:       &odigosv1.AgentMetricsConfig{},
				},
			},
		},
	}

	remoteConfig, err := manager.GetFullConfig(
		context.Background(),
		remoteResourceAttributes,
		&k8sconsts.PodWorkload{},
		"deployment-checkout",
		"go",
		instrumentationConfig,
		"app",
	)
	require.NoError(t, err)

	configMap := remoteConfig.Config.ConfigMap
	require.Contains(t, configMap, string(configsections.RemoteConfigSdkConfigSectionName))
	require.Contains(t, configMap, string(configsections.RemoteConfigContainerConfigSectionName))

	var sdkConfig configsections.RemoteConfigSdk
	err = json.Unmarshal(configMap[string(configsections.RemoteConfigSdkConfigSectionName)].Body, &sdkConfig)
	require.NoError(t, err)
	require.Equal(t, remoteResourceAttributes, sdkConfig.RemoteResourceAttributes)
	require.True(t, sdkConfig.TraceSignal.Enabled)
	require.True(t, sdkConfig.MetricsSignal.Enabled)
	require.False(t, sdkConfig.LogsSignal.Enabled)
}
