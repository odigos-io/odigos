package instrumentlang

import (
	"fmt"
	"github.com/keyval-dev/odigos/odiglet/pkg/env"
	"github.com/keyval-dev/odigos/odiglet/pkg/instrumentation/consts"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	pythonVolumeName        = "agentdir-python"
	pythonMountPath         = "/otel-auto-instrumentation"
	envOtelTracesExporter   = "OTEL_TRACES_EXPORTER"
	envOtelMetricsExporter  = "OTEL_METRICS_EXPORTER"
	envValOtelHttpExporter  = "otlp_proto_http"
	envLogCorrelation       = "OTEL_PYTHON_LOG_CORRELATION"
	pythonInitContainerName = "copy-python-agent"
	envPythonPath           = "PYTHONPATH"
)

func Python(deviceId string) *v1beta1.ContainerAllocateResponse {
	otlpEndpoint := fmt.Sprintf("http://%s:%d", env.Current.NodeIP, consts.OTLPHttpPort)
	return &v1beta1.ContainerAllocateResponse{
		Envs: map[string]string{
			envLogCorrelation:             "true",
			envPythonPath:                 "/var/odigos/python/opentelemetry/instrumentation/auto_instrumentation:/var/odigos/python",
			"OTEL_EXPORTER_OTLP_ENDPOINT": otlpEndpoint,
			"OTEL_RESOURCE_ATTRIBUTES":    fmt.Sprintf("service.name=%s,odigos.device=python", deviceId),
			envOtelTracesExporter:         envValOtelHttpExporter,
			envOtelMetricsExporter:        "",
		},
		Mounts: []*v1beta1.Mount{
			{
				ContainerPath: "/var/odigos/python",
				HostPath:      "/var/odigos/python",
				ReadOnly:      true,
			},
		},
	}
}
