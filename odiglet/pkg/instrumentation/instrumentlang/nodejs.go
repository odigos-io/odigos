package instrumentlang

import (
	"fmt"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/consts"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	nodeMountPath             = "/var/odigos/nodejs"
	nodeEnvNodeDebug          = "OTEL_NODEJS_DEBUG"
	nodeEnvTraceExporter      = "OTEL_TRACES_EXPORTER"
	nodeEnvEndpoint           = "OTEL_EXPORTER_OTLP_ENDPOINT"
	nodeEnvServiceName        = "OTEL_SERVICE_NAME"
	nodeEnvNodeOptions        = "NODE_OPTIONS"
	nodeEnvResourceAttributes = "OTEL_RESOURCE_ATTRIBUTES"
)

func NodeJS(deviceId string) *v1beta1.ContainerAllocateResponse {
	otlpEndpoint := fmt.Sprintf("http://%s:%d", env.Current.NodeIP, consts.OTLPPort)
	return &v1beta1.ContainerAllocateResponse{
		Envs: map[string]string{
			nodeEnvNodeDebug:          "true",
			nodeEnvTraceExporter:      "otlp",
			nodeEnvEndpoint:           otlpEndpoint,
			nodeEnvServiceName:        deviceId,
			nodeEnvResourceAttributes: "odigos.device=nodejs",
			nodeEnvNodeOptions:        fmt.Sprintf("--require %s/autoinstrumentation.js", nodeMountPath),
		},
		Mounts: []*v1beta1.Mount{
			{
				ContainerPath: nodeMountPath,
				HostPath:      nodeMountPath,
				ReadOnly:      true,
			},
		},
	}
}
