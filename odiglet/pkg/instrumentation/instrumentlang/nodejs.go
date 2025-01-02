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
	nodeMountPath         = "/var/odigos/nodejs"
	nodeEnvEndpoint       = "OTEL_EXPORTER_OTLP_ENDPOINT"
	nodeEnvNodeOptions    = "NODE_OPTIONS"
	nodeOdigosOpampServer = "ODIGOS_OPAMP_SERVER_HOST"
	nodeOdigosDeviceId    = "ODIGOS_INSTRUMENTATION_DEVICE_ID"
)

func NodeJS(uniqueDestinationSignals map[common.ObservabilitySignal]struct{}) *v1beta1.ContainerAllocateResponse {
	otlpEndpoint := fmt.Sprintf("http://%s:%d", env.Current.NodeIP, consts.OTLPHttpPort)
	nodeOptionsVal, _ := envOverwrite.ValToAppend(nodeEnvNodeOptions, common.OtelSdkNativeCommunity)
	opampServerHost := fmt.Sprintf("%s:%d", env.Current.NodeIP, consts.OpAMPPort)

	return &v1beta1.ContainerAllocateResponse{
		Envs: map[string]string{
			nodeEnvEndpoint:       otlpEndpoint,
			nodeEnvNodeOptions:    nodeOptionsVal,
			nodeOdigosOpampServer: opampServerHost,
			nodeOdigosDeviceId:    "123123123", //TODO(edenfed): this is not needed anymore, delete it from nodejs instrumentation and then delete from here
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
