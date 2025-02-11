package instrumentlang

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
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
	otlpEndpoint := fmt.Sprintf("http://%s:%d", env.Current.NodeIP, commonconsts.OTLPHttpPort)
	opampServerHost := fmt.Sprintf("%s:%d", env.Current.NodeIP, k8sconsts.OpAMPPort)

	return &v1beta1.ContainerAllocateResponse{
		Envs: map[string]string{
			nodeEnvEndpoint:       otlpEndpoint,
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
