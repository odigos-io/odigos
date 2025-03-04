package instrumentation_ebpf

import (
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/odigos-io/odigos/instrumentation"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
)

func SetupWithManager(mgr ctrl.Manager, configUpdates chan<- instrumentation.ConfigUpdate[ebpf.K8sConfigGroup]) error {
	log.Logger.V(0).Info("Starting reconcileres for ebpf instrumentation")
	var err error

	err = builder.
		ControllerManagedBy(mgr).
		Named("InstrumentationConfigReconciler_ebpf").
		For(&odigosv1.InstrumentationConfig{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(&InstrumentationConfigReconciler{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			ConfigUpdates: configUpdates,
		})
	if err != nil {
		return err
	}

	return nil
}
