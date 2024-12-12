package ebpf

import (
	"context"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/instrumentation"
	workload "github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/kube/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type k8sSettingsGetter struct {
	client client.Client
}

var _ instrumentation.SettingsGetter[K8sDetails] = &k8sSettingsGetter{}

func (ksg *k8sSettingsGetter) Settings(ctx context.Context, kd K8sDetails, dist instrumentation.OtelDistribution) (instrumentation.Settings, error) {
	sdkConfig, serviceName, err := ksg.instrumentationSDKConfig(ctx, kd, dist)
	if err != nil {
		return instrumentation.Settings{}, err
	}

	OtelServiceName := serviceName
	if serviceName == "" {
		OtelServiceName = kd.pw.Name
	}

	return instrumentation.Settings{
		ServiceName:        OtelServiceName,
		ResourceAttributes: utils.GetResourceAttributes(kd.pw, kd.pod.Name),
		InitialConfig:      sdkConfig,
	}, nil
}

func (ksg *k8sSettingsGetter) instrumentationSDKConfig(ctx context.Context, kd K8sDetails, dist instrumentation.OtelDistribution) (*odigosv1.SdkConfig, string, error) {
	instrumentationConfig := odigosv1.InstrumentationConfig{}
	instrumentationConfigKey := client.ObjectKey{
		Namespace: kd.pw.Namespace,
		Name:      workload.CalculateWorkloadRuntimeObjectName(kd.pw.Name, kd.pw.Kind),
	}
	if err := ksg.client.Get(ctx, instrumentationConfigKey, &instrumentationConfig); err != nil {
		// this can be valid when the instrumentation config is deleted and current pods will go down soon
		return nil, "", err
	}
	for _, config := range instrumentationConfig.Spec.SdkConfigs {
		if config.Language == dist.Language {
			return &config, instrumentationConfig.Spec.ServiceName, nil
		}
	}
	return nil, "", fmt.Errorf("no sdk config found for language %s", dist.Language)
}