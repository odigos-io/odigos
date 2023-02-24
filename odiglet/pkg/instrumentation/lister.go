package instrumentation

import (
	"context"
	"github.com/keyval-dev/odigos/odiglet/pkg/env"
	"github.com/keyval-dev/odigos/odiglet/pkg/instrumentation/fs"
	"github.com/keyval-dev/odigos/odiglet/pkg/instrumentation/instrumentlang"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultMaxDevices = 100
)

type lister struct {
	plugins map[string]dpm.PluginInterface
}

func (l *lister) GetResourceNamespace() string {
	return "instrumentation.odigos.io"
}

func (l *lister) Discover(pluginNameLists chan dpm.PluginNameList) {
	var pluginNames []string
	for name, _ := range l.plugins {
		pluginNames = append(pluginNames, name)
	}

	pluginNameLists <- pluginNames
}

func (l *lister) NewPlugin(s string) dpm.PluginInterface {
	return l.plugins[s]
}

func NewLister(ctx context.Context, clientset *kubernetes.Clientset) (dpm.ListerInterface, error) {
	maxPods, err := getInitialDeviceAmount(clientset)
	if err != nil {
		return nil, err
	}
	var availablePlugins = map[string]dpm.PluginInterface{
		"java":   NewPlugin(maxPods, instrumentlang.Java),
		"python": NewPlugin(maxPods, instrumentlang.Python),
		"nodejs": NewPlugin(maxPods, instrumentlang.NodeJS),
		"dotnet": NewPlugin(maxPods, instrumentlang.DotNet),
	}

	err = fs.CopyAgentsDirectoryToHost()
	if err != nil {
		return nil, err
	}

	return &lister{
		plugins: availablePlugins,
	}, nil
}

func getInitialDeviceAmount(clientset *kubernetes.Clientset) (int64, error) {
	// get max pods per current node
	node, err := clientset.CoreV1().Nodes().Get(context.Background(), env.Current.NodeName, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}

	maxPods, ok := node.Status.Allocatable.Pods().AsInt64()
	if !ok {
		log.Logger.V(0).Info("Failed to get max pods from node status, using default value", "default", defaultMaxDevices)
		maxPods = defaultMaxDevices
	}

	return maxPods, nil
}
