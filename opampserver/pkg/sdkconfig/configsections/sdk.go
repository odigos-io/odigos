package configsections

import (
	"encoding/json"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/protobufs"
)

// SdkConfig is kept as a compatibility section for older agents that still
// derive initial signal enablement from the legacy "SDK" remote config entry.
func CalcSdkRemoteConfig(remoteResourceAttributes []configresolvers.ResourceAttribute, containerConfig *odigosv1.ContainerAgentConfig) *RemoteConfigSdk {
	tracesEnabled := containerConfig != nil && containerConfig.Traces != nil
	metricsEnabled := containerConfig != nil && containerConfig.Metrics != nil
	logsEnabled := containerConfig != nil && containerConfig.Logs != nil

	return &RemoteConfigSdk{
		RemoteResourceAttributes: remoteResourceAttributes,
		TraceSignal: TraceSignalGeneralConfig{
			Enabled:             tracesEnabled,
			DefaultEnabledValue: true,
		},
		LogsSignal: LogSignalGeneralConfig{
			Enabled:             logsEnabled,
			DefaultEnabledValue: true,
		},
		MetricsSignal: MetricSignalGeneralConfig{
			Enabled:             metricsEnabled,
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
