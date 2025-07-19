package instrumentation

import (
	"context"

	"github.com/odigos-io/odigos-device-plugin/pkg/dpm"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/odigos-io/odigos/procdiscovery/pkg/libc"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/deviceplugin/pkg/log"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
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

func NewLister(ctx context.Context, otelSdksLsf OtelSdksLsf) (dpm.ListerInterface, error) {

	// each "device" has an amount that it can offer to the node (like real device),
	// and everytime it is used, it will be reduced by 1.
	// we (as a virtual device) have no limits on how much "instrumentation" we can offer to the node,
	// thus set it to a large number to avoid any pod being rejected due to no available device amount.
	initialDeviceSize := int64(1024)

	isEbpfSupported := env.Current.IsEBPFSupported()

	availablePlugins := map[string]dpm.PluginInterface{}
	for lang, otelSdkLsfMap := range otelSdksLsf {
		for otelSdk, lsf := range otelSdkLsfMap {
			if otelSdk.SdkType == common.EbpfOtelSdkType && !isEbpfSupported {
				continue
			}
			pluginName := common.InstrumentationPluginName(lang, otelSdk, nil)
			availablePlugins[pluginName] = NewPlugin(initialDeviceSize, lsf)

			if libc.ShouldInspectForLanguage(lang) {
				musl := common.Musl
				pluginNameMusl := common.InstrumentationPluginName(lang, otelSdk, &musl)
				availablePlugins[pluginNameMusl] = NewMuslPlugin(lang, initialDeviceSize, lsf)
			}
		}
	}

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
