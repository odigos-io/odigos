package ebpf

import (
	"context"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/instrumentation/detector"
	workload "github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type k8sSettingsGetter struct {
	client client.Client
}

var _ instrumentation.SettingsGetter[K8sProcessDetails] = &k8sSettingsGetter{}

func (ksg *k8sSettingsGetter) Settings(ctx context.Context, kd K8sProcessDetails, dist instrumentation.OtelDistribution) (instrumentation.Settings, error) {
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
		ResourceAttributes: getResourceAttributes(kd.pw, kd.pod.Name, kd.procEvent),
		InitialConfig:      sdkConfig,
	}, nil
}

func (ksg *k8sSettingsGetter) instrumentationSDKConfig(ctx context.Context, kd K8sProcessDetails, dist instrumentation.OtelDistribution) (*odigosv1.SdkConfig, string, error) {
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

func getResourceAttributes(podWorkload *k8sconsts.PodWorkload, podName string, pe detector.ProcessEvent) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.K8SNamespaceName(podWorkload.Namespace),
		semconv.K8SPodName(podName),
	}

	switch podWorkload.Kind {
	case k8sconsts.WorkloadKindDeployment:
		attrs = append(attrs, semconv.K8SDeploymentName(podWorkload.Name))
	case k8sconsts.WorkloadKindStatefulSet:
		attrs = append(attrs, semconv.K8SStatefulSetName(podWorkload.Name))
	case k8sconsts.WorkloadKindDaemonSet:
		attrs = append(attrs, semconv.K8SDaemonSetName(podWorkload.Name))
	}

	if pe.ExecDetails != nil {
		envs := pe.ExecDetails.Environments

		containerName, ok := envs[k8sconsts.OdigosEnvVarContainerName]
		if ok && containerName != "" {
			attrs = append(attrs, semconv.K8SContainerName(containerName))
		}

		if pe.ExecDetails.ExePath != "" {
			attrs = append(attrs, semconv.ProcessExecutablePath(pe.ExecDetails.ExePath))
		}

		if pe.ExecDetails.CmdLine != "" {
			cmdLine := pe.ExecDetails.CmdLine
			// we're getting the command line with space as a separator,
			// original from the /proc filesystem it has a null byte as a separator
			// TODO: we should probably change the runtime detector to return the cmdline as a string slice
			// once we do that, we can add the command args resource as well
			parts := strings.Split(cmdLine, " ")
			if len(parts) > 0 {
				attrs = append(attrs, semconv.ProcessCommand(parts[0]))
			}
		}
	}

	if pe.PID != 0 {
		attrs = append(attrs, semconv.ProcessPID(pe.PID))
	}

	if pe.ExecDetails.ContainerProcessID != 0 {
		attrs = append(attrs, semconv.ProcessVpid(pe.ExecDetails.ContainerProcessID))
	}

	return attrs
}
