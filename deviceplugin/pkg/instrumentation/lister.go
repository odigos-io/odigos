package instrumentation

import (
	"context"

	"github.com/odigos-io/odigos-device-plugin/pkg/dpm"

	"github.com/odigos-io/odigos/common"
)

const GenericPluginName = "generic"

type lister struct {
	genericPlugin dpm.PluginInterface
}

func (l *lister) GetResourceNamespace() string {
	return common.OdigosResourceNamespace
}

func (l *lister) Discover(pluginNameLists chan dpm.PluginNameList) {
	pluginNames := dpm.PluginNameList{GenericPluginName}
	pluginNameLists <- pluginNames
}

func (l *lister) NewPlugin(s string) dpm.PluginInterface {
	if s == GenericPluginName {
		return l.genericPlugin
	}
	return nil
}

func NewLister(ctx context.Context) (dpm.ListerInterface, error) {

	// each "device" has an amount that it can offer to the node (like real device),
	// and everytime it is used, it will be reduced by 1.
	// we (as a virtual device) have no limits on how much "instrumentation" we can offer to the node,
	// thus set it to a large number to avoid any pod being rejected due to no available device amount.
	initialDeviceSize := int64(1024)

	return &lister{
		genericPlugin: NewGenericPlugin(initialDeviceSize),
	}, nil
}
