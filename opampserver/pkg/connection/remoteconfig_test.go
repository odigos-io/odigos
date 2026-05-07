package connection

import (
	"encoding/json"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configsections"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalcRemoteConfigHashConsistent(t *testing.T) {
	remoteConfig := protobufs.AgentConfigMap{
		ConfigMap: map[string]*protobufs.AgentConfigFile{
			"key1": {
				Body: []byte("value1"),
			},
			"key2": {
				Body: []byte("value2"),
			},
			"key3": {
				Body: []byte("value3"),
			},
			"key4": {
				Body: []byte("value4"),
			},
			"key5": {
				Body: []byte("value5"),
			},
		},
	}

	hash1 := CalcRemoteConfigHash(&remoteConfig)
	hash2 := CalcRemoteConfigHash(&remoteConfig)
	assert.Equal(t, hash1, hash2)
}

func TestUpdateWorkloadRemoteConfigRefreshesLegacySDKSection(t *testing.T) {
	cache := NewConnectionsCache()
	workload := k8sconsts.PodWorkload{
		Namespace: "shop",
		Kind:      k8sconsts.WorkloadKindDeployment,
		Name:      "checkout",
	}
	resourceAttributes := []configresolvers.ResourceAttribute{
		{Key: "service.name", Value: "checkout"},
	}
	cache.AddConnection("instance-1", &ConnectionInfo{
		Workload:      workload,
		ContainerName: "app",
		AgentRemoteConfig: &protobufs.AgentRemoteConfig{Config: &protobufs.AgentConfigMap{
			ConfigMap: map[string]*protobufs.AgentConfigFile{
				string(configsections.RemoteConfigContainerConfigSectionName): {Body: []byte("{}"), ContentType: "application/json"},
			},
		}},
		RemoteResourceAttributes: resourceAttributes,
	})

	err := cache.UpdateWorkloadRemoteConfig(workload, []odigosv1.ContainerAgentConfig{
		{
			ContainerName: "app",
			Traces:        &odigosv1.AgentTracesConfig{},
			Metrics:       &odigosv1.AgentMetricsConfig{},
		},
	})
	require.NoError(t, err)

	conn, ok := cache.GetConnection("instance-1")
	require.True(t, ok)
	sdkConfigFile := conn.AgentRemoteConfig.Config.ConfigMap[string(configsections.RemoteConfigSdkConfigSectionName)]
	require.NotNil(t, sdkConfigFile)

	var sdkConfig configsections.RemoteConfigSdk
	require.NoError(t, json.Unmarshal(sdkConfigFile.Body, &sdkConfig))
	require.True(t, sdkConfig.TraceSignal.Enabled)
	require.True(t, sdkConfig.MetricsSignal.Enabled)
	require.False(t, sdkConfig.LogsSignal.Enabled)
	require.Equal(t, resourceAttributes, sdkConfig.RemoteResourceAttributes)
}
