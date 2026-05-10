package configsections

import (
	"encoding/json"
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/stretchr/testify/require"
)

func TestCalcSdkRemoteConfigDerivesSignalsFromContainerConfig(t *testing.T) {
	attrs := []configresolvers.ResourceAttribute{
		{Key: "service.name", Value: "checkout"},
	}
	containerConfig := &odigosv1.ContainerAgentConfig{
		ContainerName: "app",
		Traces:        &odigosv1.AgentTracesConfig{},
		Logs:          &odigosv1.AgentLogsConfig{},
	}

	sdkConfig := CalcSdkRemoteConfig(attrs, containerConfig)

	require.Equal(t, attrs, sdkConfig.RemoteResourceAttributes)
	require.True(t, sdkConfig.TraceSignal.Enabled)
	require.True(t, sdkConfig.TraceSignal.DefaultEnabledValue)
	require.False(t, sdkConfig.MetricsSignal.Enabled)
	require.True(t, sdkConfig.MetricsSignal.DefaultEnabledValue)
	require.True(t, sdkConfig.LogsSignal.Enabled)
	require.True(t, sdkConfig.LogsSignal.DefaultEnabledValue)
}

func TestSdkRemoteConfigToOpampUsesLegacySDKSectionName(t *testing.T) {
	sdkConfig := &RemoteConfigSdk{
		TraceSignal: TraceSignalGeneralConfig{Enabled: true, DefaultEnabledValue: true},
	}

	configFile, sectionName, err := SdkRemoteConfigToOpamp(sdkConfig)
	require.NoError(t, err)
	require.Equal(t, "SDK", sectionName)
	require.Equal(t, "application/json", configFile.ContentType)

	var decoded RemoteConfigSdk
	require.NoError(t, json.Unmarshal(configFile.Body, &decoded))
	require.True(t, decoded.TraceSignal.Enabled)
	require.True(t, decoded.TraceSignal.DefaultEnabledValue)
}
