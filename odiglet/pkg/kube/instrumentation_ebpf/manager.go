package instrumentation_ebpf

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/event"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/odiglet/pkg/ebpf"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type EbpfInstrumentationPredicate struct {
	predicate.Funcs
}

func (i *EbpfInstrumentationPredicate) Create(e event.CreateEvent) bool {
	return true
}

func (i *EbpfInstrumentationPredicate) Update(e event.UpdateEvent) bool {

	if e.ObjectOld == nil {
		// log.Error(nil, "Update event has no old object to update", "event", e)
		return false
	}
	if e.ObjectNew == nil {
		// log.Error(nil, "Update event has no new object for update", "event", e)
		return false
	}

	return hasEbpfInstrumentationAnnotation(e.ObjectNew) != hasEbpfInstrumentationAnnotation(e.ObjectOld)
}

func (i *EbpfInstrumentationPredicate) Delete(e event.DeleteEvent) bool {
	return true
}

func (i *EbpfInstrumentationPredicate) Generic(e event.GenericEvent) bool {
	return false
}

var (
	scheme = runtime.NewScheme()
)

func StartReconciling(ctx context.Context, ebpfDirectors map[common.ProgrammingLanguage]ebpf.Director) error {
	log.Logger.V(0).Info("Starting reconcileres for ebpf instrumentation")
	ctrl.SetLogger(log.Logger)
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	if err != nil {
		return err
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
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Owns(&odigosv1.InstrumentedApplication{}).
		WithEventFilter(&EbpfInstrumentationPredicate{}).
		Complete(&DeploymentsReconciler{
			Client:    mgr.GetClient(),
			Scheme:    mgr.GetScheme(),
			Directors: ebpfDirectors,
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&appsv1.DaemonSet{}).
		Owns(&odigosv1.InstrumentedApplication{}).
		WithEventFilter(&EbpfInstrumentationPredicate{}).
		Complete(&DaemonSetsReconciler{
			Client:    mgr.GetClient(),
			Scheme:    mgr.GetScheme(),
			Directors: ebpfDirectors,
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&appsv1.StatefulSet{}).
		Owns(&odigosv1.InstrumentedApplication{}).
		WithEventFilter(&EbpfInstrumentationPredicate{}).
		Complete(&StatefulSetsReconciler{
			Client:    mgr.GetClient(),
			Scheme:    mgr.GetScheme(),
			Directors: ebpfDirectors,
		})
	if err != nil {
		return err
	}

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			log.Logger.Error(err, "error starting instrumentation ebpf manager")
		}
	}()

	return nil
}
