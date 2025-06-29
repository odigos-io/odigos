package configsections

import (
	"encoding/json"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/protobufs"
)

// SdkConfig is sunsetting, as we are now just sending the entire config to the agent.
// however, the process is still ongoing, and after we upgrade the agent,
// opampserver will still need to serve old agents from non-restarted pods for some time.
func CalcSdkRemoteConfig(remoteResourceAttributes []configresolvers.ResourceAttribute, containerConfig *odigosv1.ContainerAgentConfig) *RemoteConfigSdk {
	tracesEnabled := containerConfig.Traces != nil
	metricsEnabled := containerConfig.Metrics != nil
	logsEnabled := containerConfig.Logs != nil

	remoteConfigSdk := RemoteConfigSdk{
		RemoteResourceAttributes: remoteResourceAttributes,
		TraceSignal: TraceSignalGeneralConfig{
			Enabled:             tracesEnabled,
			DefaultEnabledValue: true, // TODO: read from instrumentation config CRD with fallback
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

	return &remoteConfigSdk
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
