package instrumentation_ebpf

import (
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
)

func SetupWithManager(mgr ctrl.Manager, ebpfDirectors ebpf.DirectorsMap, configUpdateFunc ebpf.ConfigUpdateFunc) error {
	log.Logger.V(0).Info("Starting reconcileres for ebpf instrumentation")
	var err error

	// TODO: once we fully move to the new approach of triggering instrumentations based on the
	// process events, we can remove the PodReconciler entirely.
	if ebpfDirectors != nil {
		err = builder.
			ControllerManagedBy(mgr).
			Named("PodReconciler_ebpf").
			For(&corev1.Pod{}).
			// trigger the reconcile when either:
			// 1. A Create event is accepted for a pod with all containers ready (this is relevant when Odiglet is restarted)
			// 2. All containers become ready in a running pod
			// 3. Pod is deleted
			WithEventFilter(predicate.Or(&odigospredicate.AllContainersReadyPredicate{}, &odigospredicate.DeletionPredicate{})).
			Complete(&PodsReconciler{
				Client:    mgr.GetClient(),
				Scheme:    mgr.GetScheme(),
				Directors: ebpfDirectors,
			})
		if err != nil {
			return err
		}
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
			OnUpdate:  configUpdateFunc,
		})
	if err != nil {
		return err
	}

	return nil
}
