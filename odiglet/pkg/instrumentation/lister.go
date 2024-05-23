package instrumentation

import (
	"context"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/fs"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
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

// with this type, odiglet can determine which language specific function to use
// for each otel sdk in a each programming language.
// Otel SDKs frequently requires to set some environment variables and mount some fs dirs for it to work.
type OtelSdksLsf map[common.ProgrammingLanguage]map[common.OtelSdk]LangSpecificFunc

func NewLister(ctx context.Context, clientset *kubernetes.Clientset, otelSdksLsf OtelSdksLsf) (dpm.ListerInterface, error) {
	maxPods, err := getInitialDeviceAmount(clientset)
	if err != nil {
		return nil, err
	}

	isEbpfSupported := env.Current.IsEBPFSupported()

	availablePlugins := map[string]dpm.PluginInterface{}
	for lang, otelSdkLsfMap := range otelSdksLsf {
		for otelSdk, lsf := range otelSdkLsfMap {
			if otelSdk.SdkType == common.EbpfOtelSdkType && !isEbpfSupported {
				continue
			}
			pluginName := common.InstrumentationPluginName(lang, otelSdk)
			availablePlugins[pluginName] = NewPlugin(maxPods, lsf)
		}
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
