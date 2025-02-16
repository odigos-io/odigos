package configsections

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/protobufs"
)

func CalcSdkRemoteConfig(remoteResourceAttributes []configresolvers.ResourceAttribute, signals []common.ObservabilitySignal) *RemoteConfigSdk {
	tracesEnabled := slices.Contains(signals, common.TracesObservabilitySignal)
	metricsEnabled := slices.Contains(signals, common.MetricsObservabilitySignal)
	logsEnabled := slices.Contains(signals, common.LogsObservabilitySignal)

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
