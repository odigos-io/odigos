package instrumentlang

import (
	"github.com/odigos-io/odigos/common"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

func Python(deviceId string, uniqueDestinationSignals map[common.ObservabilitySignal]struct{}) *v1beta1.ContainerAllocateResponse {

	return &v1beta1.ContainerAllocateResponse{
		Mounts: []*v1beta1.Mount{
			{
				ContainerPath: "/var/odigos/python",
				HostPath:      "/var/odigos/python",
				ReadOnly:      true,
			},
		},
	}
}
