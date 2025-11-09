package instrumentation

import (
	"context"

	"github.com/odigos-io/odigos-device-plugin/pkg/dpm"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/odigos-io/odigos/common"
)

type lister struct {
	plugins map[string]dpm.PluginInterface
}

func (l *lister) GetResourceNamespace() string {
	return common.OdigosResourceNamespace
}

func (l *lister) Discover(pluginNameLists chan dpm.PluginNameList) {
	var pluginNames []string
	for name := range l.plugins {
		pluginNames = append(pluginNames, name)
	}

	pluginNameLists <- pluginNames
}

func (l *lister) NewPlugin(s string) dpm.PluginInterface {
	return l.plugins[s]
}

func NewLister(ctx context.Context) (dpm.ListerInterface, error) {

	// each "device" has an amount that it can offer to the node (like real device),
	// and everytime it is used, it will be reduced by 1.
	// we (as a virtual device) have no limits on how much "instrumentation" we can offer to the node,
	// thus set it to a large number to avoid any pod being rejected due to no available device amount.
	initialDeviceSize := int64(1024)

	availablePlugins := map[string]dpm.PluginInterface{}

	// device that only mounts the odigos agent directory.
	// always present regardless of the otelSdksLsf
	mountDeviceFunc := func(deviceId string) *v1beta1.ContainerAllocateResponse {
		return &v1beta1.ContainerAllocateResponse{
			Mounts: []*v1beta1.Mount{
				{
					ContainerPath: k8sconsts.OdigosAgentsDirectory,
					HostPath:      k8sconsts.OdigosAgentsDirectory,
					ReadOnly:      true,
				},
			},
		}
	}
	availablePlugins["generic"] = NewPlugin(initialDeviceSize, mountDeviceFunc)

	return &lister{
		plugins: availablePlugins,
	}, nil
}
