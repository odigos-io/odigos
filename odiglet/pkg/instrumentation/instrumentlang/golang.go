package instrumentlang

import "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

// Go is a dummy device available only on eBPF nodes. This will help us schedule applications that needs eBPF
// instrumentation on eBPF nodes only.
func Go(deviceId string) *v1beta1.ContainerAllocateResponse {
	return &v1beta1.ContainerAllocateResponse{}
}
