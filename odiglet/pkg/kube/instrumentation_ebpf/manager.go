package instrumentation_ebpf

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/odiglet/pkg/ebpf"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
)

type EbpfInstrumentationPredicate struct {
	predicate.Funcs
}

func (i *EbpfInstrumentationPredicate) Create(e event.CreateEvent) bool {
	return true
}

func (i *EbpfInstrumentationPredicate) Update(e event.UpdateEvent) bool {
	return hasEbpfInstrumentationAnnotation(e.ObjectNew) != hasEbpfInstrumentationAnnotation(e.ObjectOld)
}

func (i *EbpfInstrumentationPredicate) Delete(e event.DeleteEvent) bool {
	return true
}

func (i *EbpfInstrumentationPredicate) Generic(e event.GenericEvent) bool {
	return false
}

func SetupWithManager(mgr ctrl.Manager, ebpfDirectors map[common.ProgrammingLanguage]ebpf.Director) error {

	log.Logger.V(0).Info("Starting reconcileres for ebpf instrumentation")

	err := builder.
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
		WithEventFilter(&EbpfInstrumentationPredicate{}).
		Complete(&StatefulSetsReconciler{
			Client:    mgr.GetClient(),
			Scheme:    mgr.GetScheme(),
			Directors: ebpfDirectors,
		})
	if err != nil {
		return err
	}

	return nil
}
