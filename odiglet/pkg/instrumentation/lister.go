package instrumentation

import (
	"github.com/keyval-dev/odigos/odiglet/pkg/instrumentation/fs"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"log"
)

type lister struct{}

func (l *lister) GetResourceNamespace() string {
	return "instrumentation.odigos.io"
}

func (l *lister) Discover(pluginNameLists chan dpm.PluginNameList) {
	pluginNameLists <- []string{"java"}
}

func (l *lister) NewPlugin(s string) dpm.PluginInterface {
	log.Printf("NewPlugin: %s", s)
	return NewInstrumentationPlugin()
}

func NewLister() (dpm.ListerInterface, error) {
	err := fs.CopyAgentsDirectoryToHost()
	if err != nil {
		return nil, err
	}

	return &lister{}, nil
}
