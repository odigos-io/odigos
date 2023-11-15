package kube

import (
	"context"

	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/common/consts"
	"github.com/keyval-dev/odigos/odiglet/pkg/ebpf"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(odigosv1.AddToScheme(scheme))
}

func StartReconciling(ebpfDirectors map[common.ProgrammingLanguage]ebpf.Director) (context.Context, error) {
	log.Logger.V(0).Info("Starting reconcileres")
	ctrl.SetLogger(log.Logger)
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	if err != nil {
		return nil, err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Owns(&odigosv1.InstrumentedApplication{}).
		Complete(&DeploymentsReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return nil, err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&appsv1.StatefulSet{}).
		Owns(&odigosv1.InstrumentedApplication{}).
		Complete(&StatefulSetsReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return nil, err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&appsv1.DaemonSet{}).
		Owns(&odigosv1.InstrumentedApplication{}).
		Complete(&DaemonSetsReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return nil, err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			return isObjectLabeled(obj)
		})).
		Owns(&odigosv1.InstrumentedApplication{}).
		Complete(&NamespacesReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return nil, err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(&PodsReconciler{
			Client:    mgr.GetClient(),
			Scheme:    mgr.GetScheme(),
			Directors: ebpfDirectors,
		})
	if err != nil {
		return nil, err
	}

	ctx := signals.SetupSignalHandler()
	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			log.Logger.Error(err, "error starting manager")
		}
	}()
	return ctx, nil
}

func isObjectLabeled(obj client.Object) bool {
	labels := obj.GetLabels()
	if labels != nil {
		val, exists := labels[consts.OdigosInstrumentationLabel]
		if exists && val == consts.InstrumentationEnabled {
			return true
		}
	}

	return false
}

func isInstrumentationDisabledExplicitly(obj client.Object) bool {
	labels := obj.GetLabels()
	if labels != nil {
		val, exists := labels[consts.OdigosInstrumentationLabel]
		if exists && val == consts.InstrumentationDisabled {
			return true
		}
	}

	return false
}

func isNamespaceLabeled(ctx context.Context, obj client.Object, c client.Client) bool {
	var ns corev1.Namespace
	err := c.Get(ctx, client.ObjectKey{Name: obj.GetNamespace()}, &ns)
	if err != nil {
		log.Logger.Error(err, "error fetching namespace object")
		return false
	}

	return isObjectLabeled(&ns)
}
