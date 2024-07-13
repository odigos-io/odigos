package configsections

import (
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/protobufs"
)

func CalcSdkRemoteConfig(remoteResourceAttributes []configresolvers.ResourceAttribute, tracesEnabled bool) *RemoteConfigSdk {

	remoteConfigSdk := RemoteConfigSdk{
		RemoteResourceAttributes: remoteResourceAttributes,
		TraceSignal: TraceSignalGeneralConfig{
			Enabled:             tracesEnabled,
			DefaultEnabledValue: true, // TODO: read from instrumentation config CRD with fallback
		},
	}

	return &remoteConfigSdk
}

func SdkRemoteConfigToOpamp(remoteConfigSdk *RemoteConfigSdk) (*protobufs.AgentConfigFile, string, error) {

	remoteConfigSdkBytes, err := json.Marshal(remoteConfigSdk)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal server offered resource attributes: %w", err)
	}

	sdkConfigContent := protobufs.AgentConfigFile{
		Body:        remoteConfigSdkBytes,
		ContentType: "application/json",
	}
	return &sdkConfigContent, string(RemoteConfigSdkConfigSectionName), nil
}
