package instrumentation

import (
	"context"

	"github.com/odigos-io/odigos-device-plugin/pkg/dpm"
	odigosclientset "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"k8s.io/client-go/rest"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/odigos-io/odigos/procdiscovery/pkg/libc"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
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

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Logger.Error(err, "Failed to init Kubernetes API client")
	}
	odigosKubeClient, err := odigosclientset.NewForConfig(cfg)
	if err != nil {
		log.Logger.Error(err, "Failed to init odigos client")
	}

	availablePlugins := map[string]dpm.PluginInterface{}
	for lang, otelSdkLsfMap := range otelSdksLsf {
		for otelSdk, lsf := range otelSdkLsfMap {
			if otelSdk.SdkType == common.EbpfOtelSdkType && !isEbpfSupported {
				continue
			}
			pluginName := common.InstrumentationPluginName(lang, otelSdk, nil)
			availablePlugins[pluginName] = NewPlugin(maxPods, lsf, odigosKubeClient)

			if libc.ShouldInspectForLanguage(lang) {
				musl := common.Musl
				pluginNameMusl := common.InstrumentationPluginName(lang, otelSdk, &musl)
				availablePlugins[pluginNameMusl] = NewMuslPlugin(lang, maxPods, lsf, odigosKubeClient)
			}
		}
	}

	// device that only mounts the odigos agent directory.
	// always present regardless of the otelSdksLsf
	mountDeviceFunc := func(deviceId string, uniqueDestinationSignals map[common.ObservabilitySignal]struct{}) *v1beta1.ContainerAllocateResponse {
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
	availablePlugins["generic"] = NewPlugin(maxPods, mountDeviceFunc, odigosKubeClient)

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
