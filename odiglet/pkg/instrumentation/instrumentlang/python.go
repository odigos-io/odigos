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
	envOtelTracesExporter              = "OTEL_TRACES_EXPORTER"
	envOtelMetricsExporter             = "OTEL_METRICS_EXPORTER"
	envOtelLogsExporter                = "OTEL_LOGS_EXPORTER"
	envLogCorrelation                  = "OTEL_PYTHON_LOG_CORRELATION"
	envPythonPath                      = "PYTHONPATH"
	envOtelExporterOTLPTracesProtocol  = "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"
	pythonConfiguratorEnvVar           = "OTEL_PYTHON_CONFIGURATOR"
	pythonConfiguratorValue            = "odigos-python-configurator"
	envOtelExporterOTLPMetricsProtocol = "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL"
	httpProtobufProtocol               = "http/protobuf"
	pythonOdigosOpampServer            = "ODIGOS_OPAMP_SERVER_HOST"
	pythonOdigosDeviceId               = "ODIGOS_INSTRUMENTATION_DEVICE_ID"
)

func Python(deviceId string, uniqueDestinationSignals map[common.ObservabilitySignal]struct{}) *v1beta1.ContainerAllocateResponse {
	otlpEndpoint := fmt.Sprintf("http://%s:%d", env.Current.NodeIP, consts.OTLPHttpPort)
	pythonpathVal, _ := envOverwrite.ValToAppend(envPythonPath, common.OtelSdkNativeCommunity)
	opampServerHost := fmt.Sprintf("%s:%d", env.Current.NodeIP, consts.OpAMPPort)

	logsExporter := "none"
	metricsExporter := "none"
	tracesExporter := "none"

	if _, ok := uniqueDestinationSignals[common.MetricsObservabilitySignal]; ok {
		metricsExporter = "otlp"
	}
	if _, ok := uniqueDestinationSignals[common.TracesObservabilitySignal]; ok {
		tracesExporter = "otlp"
	}

	return &v1beta1.ContainerAllocateResponse{
		Envs: map[string]string{
			pythonOdigosDeviceId:          deviceId,
			pythonOdigosOpampServer:       opampServerHost,
			envLogCorrelation:             "true",
			envPythonPath:                 pythonpathVal,
			pythonConfiguratorEnvVar:      pythonConfiguratorValue,
			"OTEL_EXPORTER_OTLP_ENDPOINT": otlpEndpoint,
			envOtelTracesExporter:         tracesExporter,
			envOtelMetricsExporter:        metricsExporter,
			// Log exporter is currently set to "none" due to the data collection method, which collects logs from the file system.
			// In the future, this will be changed to "otlp" to send logs directly from the agent to the gateway.
			envOtelLogsExporter:                logsExporter,
			envOtelExporterOTLPTracesProtocol:  httpProtobufProtocol,
			envOtelExporterOTLPMetricsProtocol: httpProtobufProtocol,
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
