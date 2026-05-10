package connection

import (
	"encoding/json"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configsections"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateWorkloadRemoteConfigUpdatesLegacySDKSection(t *testing.T) {
	cache := NewConnectionsCache()
	workload := k8sconsts.PodWorkload{
		Name:      "checkout",
		Namespace: "default",
		Kind:      k8sconsts.WorkloadKindDeployment,
	}
	initialConfigMap := &protobufs.AgentConfigMap{
		ConfigMap: map[string]*protobufs.AgentConfigFile{
			"container_config": {
				Body:        []byte(`{"containerName":"app"}`),
				ContentType: "application/json",
			},
		},
	}
	attrs := []configresolvers.ResourceAttribute{
		{Key: "service.name", Value: "checkout"},
	}
	cache.AddConnection("instance-1", &ConnectionInfo{
		Workload: workload,
		Pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "checkout-pod", Namespace: "default"},
		},
		ContainerName: "app",
		Pid:           123,
		AgentRemoteConfig: &protobufs.AgentRemoteConfig{
			Config:     initialConfigMap,
			ConfigHash: CalcRemoteConfigHash(initialConfigMap),
		},
		RemoteResourceAttributes: attrs,
	})

	err := cache.UpdateWorkloadRemoteConfig(workload, []odigosv1.ContainerAgentConfig{
		{
			ContainerName: "app",
			Traces:        &odigosv1.AgentTracesConfig{},
			Logs:          &odigosv1.AgentLogsConfig{},
		},
	})

	require.NoError(t, err)
	updatedConnection, ok := cache.GetConnection("instance-1")
	require.True(t, ok)
	sdkSection := updatedConnection.AgentRemoteConfig.Config.ConfigMap["SDK"]
	require.NotNil(t, sdkSection)

	var sdkConfig configsections.RemoteConfigSdk
	require.NoError(t, json.Unmarshal(sdkSection.Body, &sdkConfig))
	require.Equal(t, attrs, sdkConfig.RemoteResourceAttributes)
	require.True(t, sdkConfig.TraceSignal.Enabled)
	require.False(t, sdkConfig.MetricsSignal.Enabled)
	require.True(t, sdkConfig.LogsSignal.Enabled)
}
