package instrumentlang

import (
	"fmt"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/consts"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	javaToolOptionsEnvVar         = "JAVA_TOOL_OPTIONS"
	javaOptsEnvVar                = "JAVA_OPTS"
	javaOtlpEndpointEnvVar        = "OTEL_EXPORTER_OTLP_ENDPOINT"
	javaOtlpProtocolEnvVar        = "OTEL_EXPORTER_OTLP_PROTOCOL"
	javaOtelLogsExporterEnvVar    = "OTEL_LOGS_EXPORTER"
	javaOtelMetricsExporterEnvVar = "OTEL_METRICS_EXPORTER"
	javaOtelTracesExporterEnvVar  = "OTEL_TRACES_EXPORTER"
	javaOtelTracesSamplerEnvVar   = "OTEL_TRACES_SAMPLER"
)

func Java(uniqueDestinationSignals map[common.ObservabilitySignal]struct{}) *v1beta1.ContainerAllocateResponse {
	otlpEndpoint := fmt.Sprintf("http://%s:%d", env.Current.NodeIP, consts.OTLPPort)
	javaOptsVal, _ := envOverwrite.ValToAppend(javaOptsEnvVar, common.OtelSdkNativeCommunity)
	javaToolOptionsVal, _ := envOverwrite.ValToAppend(javaToolOptionsEnvVar, common.OtelSdkNativeCommunity)

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
			javaToolOptionsEnvVar:         javaToolOptionsVal,
			javaOptsEnvVar:                javaOptsVal,
			javaOtlpEndpointEnvVar:        otlpEndpoint,
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
