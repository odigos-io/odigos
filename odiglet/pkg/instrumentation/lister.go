package instrumentation

import (
	"context"
	"github.com/keyval-dev/odigos/odiglet/pkg/instrumentation/devices"
	"github.com/keyval-dev/odigos/odiglet/pkg/instrumentation/fs"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"k8s.io/client-go/kubernetes"
	"log"
)

type lister struct {
	plugins map[string]dpm.PluginInterface
}

func (l *lister) GetResourceNamespace() string {
	return "instrumentation.odigos.io"
}

func (l *lister) Discover(pluginNameLists chan dpm.PluginNameList) {
	pluginNameLists <- []string{"java"}
}

func (l *lister) NewPlugin(s string) dpm.PluginInterface {
	log.Printf("NewPlugin: %s", s)
	return l.plugins[s]
}

func NewLister(ctx context.Context, clientset *kubernetes.Clientset) (dpm.ListerInterface, error) {
	kubeletClient, err := devices.NewKubeletClient()
	if err != nil {
		return nil, err
	}

	idManager, err := devices.NewIDManager(kubeletClient, clientset)
	if err != nil {
		return nil, err
	}

	var availablePlugins = map[string]dpm.PluginInterface{
		"java": NewJavaPlugin(idManager),
	}

	err = fs.CopyAgentsDirectoryToHost()
	if err != nil {
		return nil, err
	}

	return &lister{
		plugins: availablePlugins,
	}, nil
}
