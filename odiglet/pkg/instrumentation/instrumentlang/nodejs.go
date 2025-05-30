package instrumentlang

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/service"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	nodeMountPath         = "/var/odigos/nodejs"
	nodeEnvEndpoint       = "OTEL_EXPORTER_OTLP_ENDPOINT"
	nodeEnvServiceName    = "OTEL_SERVICE_NAME"
	nodeOdigosOpampServer = "ODIGOS_OPAMP_SERVER_HOST"
	nodeOdigosDeviceId    = "ODIGOS_INSTRUMENTATION_DEVICE_ID"
)

func NodeJS(deviceId string, uniqueDestinationSignals map[common.ObservabilitySignal]struct{}) *v1beta1.ContainerAllocateResponse {
	opampServerHost := service.LocalTrafficOpAmpOdigletEndpoint(env.Current.NodeIP)

	return &v1beta1.ContainerAllocateResponse{
		Envs: map[string]string{
			nodeEnvEndpoint:       service.LocalTrafficOTLPHttpDataCollectionEndpoint(env.Current.NodeIP),
			nodeEnvServiceName:    deviceId, // temporary set the device id as well, so if opamp fails we can fallback to resolve k8s attributes in the collector
			nodeOdigosOpampServer: opampServerHost,
			nodeOdigosDeviceId:    deviceId,
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
