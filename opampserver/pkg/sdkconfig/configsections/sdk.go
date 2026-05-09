package configsections

import (
	"encoding/json"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/protobufs"
)

// SdkConfig is sunsetting, but older agents in already-running pods still use
// it for signal enablement until they are restarted with container_config support.
func CalcSdkRemoteConfig(remoteResourceAttributes []configresolvers.ResourceAttribute, containerConfig *odigosv1.ContainerAgentConfig) *RemoteConfigSdk {
	return &RemoteConfigSdk{
		RemoteResourceAttributes: remoteResourceAttributes,
		TraceSignal: TraceSignalGeneralConfig{
			Enabled:             containerConfig.Traces != nil,
			DefaultEnabledValue: true,
		},
		LogsSignal: LogSignalGeneralConfig{
			Enabled:             containerConfig.Logs != nil,
			DefaultEnabledValue: true,
		},
		MetricsSignal: MetricSignalGeneralConfig{
			Enabled:             containerConfig.Metrics != nil,
			DefaultEnabledValue: true,
		},
	}
}

func SdkRemoteConfigToOpamp(remoteConfigSdk *RemoteConfigSdk) (*protobufs.AgentConfigFile, string, error) {
	remoteConfigSdkBytes, err := json.Marshal(remoteConfigSdk)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal server sdk remote config: %w", err)
	}

	sdkConfigContent := protobufs.AgentConfigFile{
		Body:        remoteConfigSdkBytes,
		ContentType: "application/json",
	}
	return &sdkConfigContent, string(RemoteConfigSdkConfigSectionName), nil
}
