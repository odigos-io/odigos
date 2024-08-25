package configsections

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CalcInstrumentationLibrariesRemoteConfig(ctx context.Context, kubeClient client.Client, configObjectName string, ns string) ([]RemoteConfigInstrumentationLibrary, error) {

	instrumentationSdkConfig := &v1alpha1.InstrumentationConfig{}
	err := kubeClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: configObjectName}, instrumentationSdkConfig)
	// if the crd is not found, just use the empty one which we initialized above
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}

	instrumentationLibrariesConfig := make([]RemoteConfigInstrumentationLibrary, 0)
	for _, sdkConfig := range instrumentationSdkConfig.Spec.SdkConfigs {
		for _, instrumentationConfig := range sdkConfig.InstrumentationLibraryConfigs {

			var tracesEnabled *bool
			if instrumentationConfig.TraceConfig != nil {
				tracesEnabled = instrumentationConfig.TraceConfig.Enabled
			}

			instrumentationLibrariesConfig = append(instrumentationLibrariesConfig, RemoteConfigInstrumentationLibrary{
				Name: instrumentationConfig.InstrumentationLibraryId.InstrumentationLibraryName,
				Traces: RemoteConfigInstrumentationLibraryTraces{
					Enabled: tracesEnabled,
				},
			})
		}
	}

	return instrumentationLibrariesConfig, nil
}

func InstrumentationLibrariesRemoteConfigToOpamp(remoteConfigInstrumentationLibraries []RemoteConfigInstrumentationLibrary) (*protobufs.AgentConfigFile, string, error) {

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
