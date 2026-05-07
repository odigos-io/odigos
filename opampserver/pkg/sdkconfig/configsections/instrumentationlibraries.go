package configsections

import (
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/opampserver/protobufs"
)

func InstrumentationLibrariesRemoteConfigToOpamp() (*protobufs.AgentConfigFile, string, error) {

	remoteConfigInstrumentationLibraries := make([]RemoteConfigInstrumentationLibrary, 0)
	remoteConfigSdkBytes, err := json.Marshal(remoteConfigInstrumentationLibraries)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal instrumentation libraries remote config: %w", err)
	}

	sdkConfigContent := protobufs.AgentConfigFile{
		Body:        remoteConfigSdkBytes,
		ContentType: "application/json",
	}
	return &sdkConfigContent, string(RemoteConfigInstrumentationLibrariesConfigSectionName), nil
}
