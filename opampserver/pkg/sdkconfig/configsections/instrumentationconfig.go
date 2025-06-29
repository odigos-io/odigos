package configsections

import (
	"context"
	"encoding/json"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetWorkloadInstrumentationConfig(ctx context.Context, kubeClient client.Client, configObjectName string, ns string) (*v1alpha1.InstrumentationConfig, error) {
	instrumentationConfig := &v1alpha1.InstrumentationConfig{}
	err := kubeClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: configObjectName}, instrumentationConfig)

	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}

	return instrumentationConfig, nil
}

func FilterRelevantSdk(instrumentationConfig *v1alpha1.InstrumentationConfig, programmingLanguage string) (*protobufs.AgentConfigFile, error) {
	relevantSdkConfig := v1alpha1.SdkConfig{}

	for _, sdkConfig := range instrumentationConfig.Spec.SdkConfigs {
		if common.MapOdigosToSemConv(sdkConfig.Language) == programmingLanguage {
			relevantSdkConfig = sdkConfig
			break
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

func FilterRelevantContainerConfig(instrumentationConfig *v1alpha1.InstrumentationConfig, containerName string) (*protobufs.AgentConfigFile, error) {
	containerConfig := v1alpha1.ContainerAgentConfig{}

	for _, container := range instrumentationConfig.Spec.Containers {
		if container.ContainerName == containerName {
			containerConfig = container
			break
		}
	}

	remoteConfigContainerConfigBytes, err := json.Marshal(containerConfig)
	if err != nil {
		return nil, err
	}

	containerConfigContent := protobufs.AgentConfigFile{
		Body:        remoteConfigContainerConfigBytes,
		ContentType: "application/json",
	}

	return &containerConfigContent, nil

}
