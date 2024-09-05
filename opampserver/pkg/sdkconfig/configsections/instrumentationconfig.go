package configsections

import (
	"context"
	"encoding/json"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetWorkloadInstrumentationConfig(ctx context.Context, kubeClient client.Client, configObjectName string, ns string, programmingLanguage string) (*protobufs.AgentConfigFile, error) {
	instrumentationSdkConfig := &v1alpha1.InstrumentationConfig{}
	err := kubeClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: configObjectName}, instrumentationSdkConfig)
	// if the crd is not found, just use the empty one which we initialized above
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}

	relevantSdkConfig := v1alpha1.SdkConfig{}

	for _, sdkConfig := range instrumentationSdkConfig.Spec.SdkConfigs {
		if string(sdkConfig.Language) == programmingLanguage {
			relevantSdkConfig = sdkConfig
		}
	}

	remoteConfigInstrumentationConfigBytes, err := json.Marshal(relevantSdkConfig)
	if err != nil {
		return nil, err
	}

	instrumentationConfigContent := protobufs.AgentConfigFile{
		Body:        remoteConfigInstrumentationConfigBytes,
		ContentType: "application/json",
	}

	return &instrumentationConfigContent, nil

}
