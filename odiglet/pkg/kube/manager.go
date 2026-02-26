package kube

import (
	"fmt"

	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentation"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/kube/instrumentation_ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/kube/runtime_details"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	criwrapper "github.com/odigos-io/odigos/k8sutils/pkg/cri"
	"github.com/odigos-io/odigos/k8sutils/pkg/feature"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(odigosv1.AddToScheme(scheme))
}

type KubeManagerOptions struct {
	Mgr                     ctrl.Manager
	ConfigUpdates           chan<- instrumentation.ConfigUpdate[ebpf.K8sConfigGroup]
	InstrumentationRequests chan<- instrumentation.Request[ebpf.K8sProcessGroup, ebpf.K8sConfigGroup, *ebpf.K8sProcessDetails]
	CriClient               *criwrapper.CriClient
	// map where keys are the names of the environment variables that participate in append mechanism
	// they need to be recorded by runtime detection into the runtime info, and this list instruct what to collect.
	AppendEnvVarNames map[string]struct{}
}

func CreateManager(instrumentationMgrOpts ebpf.InstrumentationManagerOptions) (ctrl.Manager, error) {
	commonlogger.Logger().Info("Starting reconcilers for runtime details")
	ctrl.SetLogger(commonlogger.FromSlogHandler())

	currentNodeSelector := fields.OneTermEqualSelector("spec.nodeName", env.Current.NodeName)

	metricsBindAddress := "0"
	if feature.ServiceInternalTrafficPolicy(feature.GA) {
		// If the internal traffic policy is enabled, it means we are not bound to the hose network,
		// we can create metrics server without worrying about conflicts.
		metricsBindAddress = fmt.Sprintf(":%d", k8sconsts.OdigletMetricsServerPort)
	}

	return manager.New(config.GetConfigOrDie(), manager.Options{
		Scheme: scheme,
		Cache: cache.Options{
			// ManagedFields are removed to save space. This can save a lot of space and recommended in the cache package.
			// running `kubectl get .... --show-managed-fields` will show the managed fields.
			DefaultTransform: cache.TransformStripManagedFields(),
			ByObject: map[client.Object]cache.ByObject{
				&corev1.Pod{}: {
					Field: currentNodeSelector,
				},
			},
		},
		Metrics: metricsserver.Options{
			BindAddress: metricsBindAddress,
		},
		HealthProbeBindAddress: fmt.Sprintf(":%d", instrumentationMgrOpts.OdigletHealthProbeBindPort),
	})
}

func SetupWithManager(kubeManagerOptions KubeManagerOptions, distributionGetter *distros.Getter) error {
	err := runtime_details.SetupWithManager(kubeManagerOptions.Mgr, kubeManagerOptions.CriClient, kubeManagerOptions.AppendEnvVarNames)
	if err != nil {
		return err
	}

	err = instrumentation_ebpf.SetupWithManager(kubeManagerOptions.Mgr, kubeManagerOptions.ConfigUpdates, kubeManagerOptions.InstrumentationRequests, distributionGetter)
	if err != nil {
		return err
	}

	return nil
}
