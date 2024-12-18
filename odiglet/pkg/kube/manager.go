package kube

import (
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/instrumentation"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/kube/instrumentation_ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/kube/runtime_details"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	ctrl "sigs.k8s.io/controller-runtime"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	criwrapper "github.com/odigos-io/odigos/k8sutils/pkg/cri"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(odigosv1.AddToScheme(scheme))
}

type KubeManagerOptions struct {
	Mgr           ctrl.Manager
	EbpfDirectors ebpf.DirectorsMap
	Clientset     *kubernetes.Clientset
	ConfigUpdates chan<- instrumentation.ConfigUpdate[ebpf.K8sConfigGroup]
	CriClient     *criwrapper.CriClient
}

func CreateManager() (ctrl.Manager, error) {
	log.Logger.V(0).Info("Starting reconcileres for runtime details")
	ctrl.SetLogger(log.Logger)
	return manager.New(config.GetConfigOrDie(), manager.Options{
		Scheme: scheme,
		Cache: cache.Options{
			// ManagedFields are removed to save space. This can save a lot of space and recommended in the cache package.
			// running `kubectl get .... --show-managed-fields` will show the managed fields.
			DefaultTransform: cache.TransformStripManagedFields(),
			ByObject: map[client.Object]cache.ByObject{
				&corev1.Pod{}: {
					// only watch and list pods in the current node
					Field: fields.OneTermEqualSelector("spec.nodeName", env.Current.NodeName),
				},
				&corev1.Namespace{}: {
					Label: labels.Set{consts.OdigosInstrumentationLabel: consts.InstrumentationEnabled}.AsSelector(),
				},
			},
		},
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
}

func SetupWithManager(kubeManagerOptions KubeManagerOptions) error {
	err := runtime_details.SetupWithManager(kubeManagerOptions.Mgr, kubeManagerOptions.Clientset, kubeManagerOptions.CriClient)
	if err != nil {
		return err
	}

	err = instrumentation_ebpf.SetupWithManager(kubeManagerOptions.Mgr, kubeManagerOptions.EbpfDirectors, kubeManagerOptions.ConfigUpdates)
	if err != nil {
		return err
	}

	return nil
}
