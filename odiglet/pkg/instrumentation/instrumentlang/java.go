package instrumentlang

import (
	"fmt"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/consts"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	otelResourceAttributesEnvVar = "OTEL_RESOURCE_ATTRIBUTES"
	otelResourceAttrPatteern     = "service.name=%s,odigos.device=java"
	javaToolOptionsEnvVar        = "JAVA_TOOL_OPTIONS"
	javaOptsEnvVar               = "JAVA_OPTS"
	javaOtlpEndpointEnvVar       = "OTEL_EXPORTER_OTLP_ENDPOINT"
	javaOtlpProtocolEnvVar       = "OTEL_EXPORTER_OTLP_PROTOCOL"
	javaOtelLogsExporterEnvVar   = "OTEL_LOGS_EXPORTER"
	javaOtelTracesSamplerEnvVar  = "OTEL_TRACES_SAMPLER"
	javaToolOptions              = "-javaagent:/var/odigos/java/javaagent.jar"
)

func Java(deviceId string) *v1beta1.ContainerAllocateResponse {
	otlpEndpoint := fmt.Sprintf("http://%s:%d", env.Current.NodeIP, consts.OTLPPort)

	return &v1beta1.ContainerAllocateResponse{
		Envs: map[string]string{
			otelResourceAttributesEnvVar: fmt.Sprintf(otelResourceAttrPatteern, deviceId),
			javaToolOptionsEnvVar:        javaToolOptions,
			javaOptsEnvVar:               javaToolOptions,
			javaOtlpEndpointEnvVar:       otlpEndpoint,
			javaOtlpProtocolEnvVar:	      "grpc",
			javaOtelLogsExporterEnvVar:   "none",
			javaOtelTracesSamplerEnvVar:  "always_on",
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
