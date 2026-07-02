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
	attrs := []configresolvers.ResourceAttribute{
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
		attrs,
		&k8sconsts.PodWorkload{Name: "checkout", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment},
		"deployment-checkout",
		"python",
		instrumentationConfig,
		"app",
	)

	require.NoError(t, err)
	sdkSection := remoteConfig.Config.ConfigMap["SDK"]
	require.NotNil(t, sdkSection)

	var sdkConfig configsections.RemoteConfigSdk
	require.NoError(t, json.Unmarshal(sdkSection.Body, &sdkConfig))
	require.Equal(t, attrs, sdkConfig.RemoteResourceAttributes)
	require.True(t, sdkConfig.TraceSignal.Enabled)
	require.True(t, sdkConfig.MetricsSignal.Enabled)
	require.False(t, sdkConfig.LogsSignal.Enabled)
}
