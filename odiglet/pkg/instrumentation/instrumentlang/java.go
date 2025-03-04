package instrumentlang

import (
	"fmt"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/service"
	"github.com/odigos-io/odigos/odiglet/pkg/env"

	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	otelResourceAttributesEnvVar = "OTEL_RESOURCE_ATTRIBUTES"
	otelResourceAttrPattern      = "service.name=%s,odigos.device=java"

	javaOtlpEndpointEnvVar        = "OTEL_EXPORTER_OTLP_ENDPOINT"
	javaOtlpProtocolEnvVar        = "OTEL_EXPORTER_OTLP_PROTOCOL"
	javaOtelLogsExporterEnvVar    = "OTEL_LOGS_EXPORTER"
	javaOtelMetricsExporterEnvVar = "OTEL_METRICS_EXPORTER"
	javaOtelTracesExporterEnvVar  = "OTEL_TRACES_EXPORTER"
	javaOtelTracesSamplerEnvVar   = "OTEL_TRACES_SAMPLER"
)

func Java(deviceId string, uniqueDestinationSignals map[common.ObservabilitySignal]struct{}) *v1beta1.ContainerAllocateResponse {
	logsExporter := "none"
	metricsExporter := "none"
	tracesExporter := "none"

	// Set the values based on the signals exists in the map
	if _, ok := uniqueDestinationSignals[common.LogsObservabilitySignal]; ok {
		logsExporter = "otlp"
	}
	if _, ok := uniqueDestinationSignals[common.MetricsObservabilitySignal]; ok {
		metricsExporter = "otlp"
	}
	if _, ok := uniqueDestinationSignals[common.TracesObservabilitySignal]; ok {
		tracesExporter = "otlp"
	}

	return &v1beta1.ContainerAllocateResponse{
		Envs: map[string]string{
			otelResourceAttributesEnvVar: fmt.Sprintf(otelResourceAttrPattern, deviceId),
			// OTEL javaagent seems to expect the endpoint to be in the format http://<host>:<port>
			// in go, we can pass the endpoint as <host>:<port>
			javaOtlpEndpointEnvVar:        fmt.Sprintf("http://%s", service.LocalTrafficOTLPDataCollectionEndpoint(env.Current.NodeIP)),
			javaOtlpProtocolEnvVar:        "grpc",
			javaOtelLogsExporterEnvVar:    logsExporter,
			javaOtelMetricsExporterEnvVar: metricsExporter,
			javaOtelTracesExporterEnvVar:  tracesExporter,
			javaOtelTracesSamplerEnvVar:   "always_on",
		},
		Mounts: []*v1beta1.Mount{
			{
				ContainerPath: "/var/odigos/java",
				HostPath:      "/var/odigos/java",
				ReadOnly:      true,
			},
		},
	}
}
