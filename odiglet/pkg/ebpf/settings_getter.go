package ebpf

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros/distro"
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

var _ instrumentation.SettingsGetter[K8sProcessGroup, K8sConfigGroup, *K8sProcessDetails] = &k8sSettingsGetter{}

func (ksg *k8sSettingsGetter) Settings(ctx context.Context, logger logr.Logger, kd *K8sProcessDetails, dist *distro.OtelDistro) (instrumentation.Settings, error) {
	sdkConfig, serviceName, err := ksg.instrumentationSDKConfig(ctx, kd, dist.Language)
	if err != nil {
		return instrumentation.Settings{}, err
	}

	OtelServiceName := serviceName
	if serviceName == "" {
		OtelServiceName = kd.Pw.Name
	}

	resourceAttributes, err := getResourceAttributes(kd.Pw, kd.Pod, kd.ProcEvent)
	if err != nil {
		commonlogger.Logger().With("subsystem", "settings-getter").Warn("error getting resource attributes", "err", err)
	}

	return instrumentation.Settings{
		ServiceName:        OtelServiceName,
		ResourceAttributes: resourceAttributes,
		InitialConfig:      sdkConfig,
	}, nil
}

func (ksg *k8sSettingsGetter) instrumentationSDKConfig(ctx context.Context, kd *K8sProcessDetails, lang common.ProgrammingLanguage) (*odigosv1.SdkConfig, string, error) {
	instrumentationConfig := odigosv1.InstrumentationConfig{}
	instrumentationConfigKey := client.ObjectKey{
		Namespace: kd.Pw.Namespace,
		Name:      workload.CalculateWorkloadRuntimeObjectName(kd.Pw.Name, kd.Pw.Kind),
	}
	if err := ksg.client.Get(ctx, instrumentationConfigKey, &instrumentationConfig); err != nil {
		// this can be valid when the instrumentation config is deleted and current pods will go down soon
		return nil, "", err
	}
	for _, config := range instrumentationConfig.Spec.SdkConfigs {
		if config.Language == lang {
			return &config, instrumentationConfig.Spec.ServiceName, nil
		}
	}
	return nil, "", fmt.Errorf("no sdk config found for language %s", lang)
}

// parseOtelResourceAttributes parses the OTEL_RESOURCE_ATTRIBUTES environment variable
// which is in the format "key1=value1,key2=value2" and returns a slice of attribute.KeyValue
func parseOtelResourceAttributes(envValue string) ([]attribute.KeyValue, error) {
	if envValue == "" {
		return nil, nil
	}

	var errs []error

	var attrs []attribute.KeyValue
	pairs := strings.Split(envValue, ",")

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			// Skip malformed pairs
			errs = append(errs, fmt.Errorf("malformed otel resource attribute pair: %s", pair))
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key != "" && value != "" {
			attrs = append(attrs, attribute.String(key, value))
		} else {
			errs = append(errs, fmt.Errorf("empty key or value in otel resource attribute pair: %s", pair))
		}
	}

	return attrs, errors.Join(errs...)
}

// appendUniqueAttributes appends new attributes to the existing slice, skipping any that already exist
func appendUniqueAttributes(existing []attribute.KeyValue, new []attribute.KeyValue) []attribute.KeyValue {
	// Create a map to track existing keys for quick lookup
	existingKeys := make(map[attribute.Key]bool)
	for _, attr := range existing {
		existingKeys[attr.Key] = true
	}

	// Append only attributes with keys that don't already exist
	for _, attr := range new {
		if !existingKeys[attr.Key] {
			existing = append(existing, attr)
		}
	}

	return existing
}

func getResourceAttributes(podWorkload *k8sconsts.PodWorkload, pod *corev1.Pod, pe detector.ProcessEvent) ([]attribute.KeyValue, error) {
	var errs []error
	// we should add all the resource attributes we want regardless of the OTEL_RESOURCE_ATTRIBUTE
	// which might be added to the target pod by our webhook or present by the user.
	// some pods might be instrumented without restart, so we can't fully rely on the webhook to put all the values
	// in the environment variable.
	attrs := []attribute.KeyValue{
		semconv.K8SNamespaceName(podWorkload.Namespace),
		semconv.K8SPodName(pod.Name),
		attribute.String(consts.OdigosWorkloadKindAttribute, string(podWorkload.Kind)),
		attribute.String(consts.OdigosWorkloadNameAttribute, podWorkload.Name),
	}

	switch podWorkload.Kind {
	case k8sconsts.WorkloadKindDeployment:
		attrs = append(attrs, semconv.K8SDeploymentName(podWorkload.Name))
	case k8sconsts.WorkloadKindStatefulSet:
		attrs = append(attrs, semconv.K8SStatefulSetName(podWorkload.Name))
	case k8sconsts.WorkloadKindDaemonSet:
		attrs = append(attrs, semconv.K8SDaemonSetName(podWorkload.Name))
	case k8sconsts.WorkloadKindCronJob:
		attrs = append(attrs, semconv.K8SCronJobName(podWorkload.Name))
	case k8sconsts.WorkloadKindJob:
		attrs = append(attrs, semconv.K8SJobName(podWorkload.Name))
	// pods and static pods workload already have the k8s.pod.name attribute
	}

	if pe.ExecDetails != nil {
		envs := pe.ExecDetails.Environments

		containerName, ok := envs[k8sconsts.OdigosEnvVarContainerName]
		if ok && containerName != "" {
			attrs = append(attrs, semconv.K8SContainerName(containerName))
		}

		// Parse OTEL_RESOURCE_ATTRIBUTES environment variable
		if otelResourceAttrs, ok := envs[k8sconsts.OtelResourceAttributesEnvVar]; ok {
			parsedAttrs, err := parseOtelResourceAttributes(otelResourceAttrs)
			if err != nil {
				errs = append(errs, err)
			}
			attrs = appendUniqueAttributes(attrs, parsedAttrs)
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

	return attrs, errors.Join(errs...)
}
