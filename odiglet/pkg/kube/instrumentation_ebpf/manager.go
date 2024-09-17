package instrumentation_ebpf

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
)

type podPredicate struct {
	predicate.Funcs
}

func (i *podPredicate) Create(e event.CreateEvent) bool {
	// when odiglet restart, it will receive create event for all running pods
	// which we need to process to instrument them
	return true
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

	// Sum the restart counts for both oldPod and newPod containers, then compare them.
	// If the newPod has a higher restart count than the oldPod, we need to re-instrument it.
	// This happens because the pod was abruptly killed, which caused an increment in the restart count.
	// This check is required because the pod will remain running during the process kill and re-launch.
	return GetPodSumRestarts(newPod) > GetPodSumRestarts(oldPod)
}

func (i *podPredicate) Delete(e event.DeleteEvent) bool {
	return true
}

func (i *podPredicate) Generic(e event.GenericEvent) bool {
	return false
}

func SetupWithManager(mgr ctrl.Manager, ebpfDirectors ebpf.DirectorsMap) error {

	log.Logger.V(0).Info("Starting reconcileres for ebpf instrumentation")

	err := builder.
		ControllerManagedBy(mgr).
		Named("PodReconciler_ebpf").
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
		Named("InstrumentationConfigReconciler_ebpf").
		For(&odigosv1.InstrumentationConfig{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(&InstrumentationConfigReconciler{
			Client:    mgr.GetClient(),
			Scheme:    mgr.GetScheme(),
			Directors: ebpfDirectors,
		})
	if err != nil {
		return err
	}

	return nil
}
