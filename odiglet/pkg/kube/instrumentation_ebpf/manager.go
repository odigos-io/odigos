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

type podPredicate struct {
	predicate.Funcs
}

func (i *podPredicate) Create(e event.CreateEvent) bool {
	// when it is created, it is not running yet
	return false
}

func (i *podPredicate) Update(e event.UpdateEvent) bool {
	// Cast old and new objects to *corev1.Pod
	oldPod, oldOk := e.ObjectOld.(*corev1.Pod)
	newPod, newOk := e.ObjectNew.(*corev1.Pod)

	// Check if both old and new objects are Pods
	if !oldOk || !newOk {
		return false
	}

	// Check if the Pod status has changed from not running to running
	if oldPod.Status.Phase != corev1.PodRunning && newPod.Status.Phase == corev1.PodRunning {
		return true
	}

	return false
}

func (i *podPredicate) Delete(e event.DeleteEvent) bool {
	return true
}

func (i *podPredicate) Generic(e event.GenericEvent) bool {
	return false
}

type workloadPredicate struct {
	predicate.Funcs
}

func (i *workloadPredicate) Create(e event.CreateEvent) bool {
	return true
}

func (i *workloadPredicate) Update(e event.UpdateEvent) bool {
	return hasEbpfInstrumentationAnnotation(e.ObjectNew) != hasEbpfInstrumentationAnnotation(e.ObjectOld)
}

func (i *workloadPredicate) Delete(e event.DeleteEvent) bool {
	return true
}

func (i *workloadPredicate) Generic(e event.GenericEvent) bool {
	return true
}

func SetupWithManager(mgr ctrl.Manager, ebpfDirectors map[common.ProgrammingLanguage]ebpf.Director) error {

	log.Logger.V(0).Info("Starting reconcileres for ebpf instrumentation")

	err := builder.
		ControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithEventFilter(&podPredicate{}).
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
		WithEventFilter(&workloadPredicate{}).
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
		WithEventFilter(&workloadPredicate{}).
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
		WithEventFilter(&workloadPredicate{}).
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
