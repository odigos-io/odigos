package connection

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configsections"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestUpdateWorkloadRemoteConfigUpdatesLegacySDKSection(t *testing.T) {
	cache := NewConnectionsCache()
	workload := k8sconsts.PodWorkload{
		Namespace: "default",
		Kind:      k8sconsts.WorkloadKindDeployment,
		Name:      "checkout",
	}
	remoteResourceAttributes := []configresolvers.ResourceAttribute{
		{Key: "service.name", Value: "checkout"},
	}

	cache.AddConnection("instance-1", &ConnectionInfo{
		Workload:        workload,
		Pod:             &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "checkout-123"}},
		ContainerName:   "app",
		Pid:             123,
		LastMessageTime: time.Now(),
		AgentRemoteConfig: &protobufs.AgentRemoteConfig{
			Config: &protobufs.AgentConfigMap{
				ConfigMap: map[string]*protobufs.AgentConfigFile{
					string(configsections.RemoteConfigContainerConfigSectionName): &protobufs.AgentConfigFile{Body: []byte("{}")},
				},
			},
		},
		RemoteResourceAttributes: remoteResourceAttributes,
	})

	err := cache.UpdateWorkloadRemoteConfig(workload, []odigosv1.ContainerAgentConfig{
		{
			ContainerName: "app",
			Traces:        &odigosv1.AgentTracesConfig{},
			Logs:          &odigosv1.AgentLogsConfig{},
		},
	})
	require.NoError(t, err)

	conn, ok := cache.GetConnection("instance-1")
	require.True(t, ok)

	configMap := conn.AgentRemoteConfig.Config.ConfigMap
	require.Contains(t, configMap, string(configsections.RemoteConfigSdkConfigSectionName))
	require.Contains(t, configMap, string(configsections.RemoteConfigContainerConfigSectionName))

	var sdkConfig configsections.RemoteConfigSdk
	err = json.Unmarshal(configMap[string(configsections.RemoteConfigSdkConfigSectionName)].Body, &sdkConfig)
	require.NoError(t, err)
	require.Equal(t, remoteResourceAttributes, sdkConfig.RemoteResourceAttributes)
	require.True(t, sdkConfig.TraceSignal.Enabled)
	require.True(t, sdkConfig.LogsSignal.Enabled)
	require.False(t, sdkConfig.MetricsSignal.Enabled)
}
