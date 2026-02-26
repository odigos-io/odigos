package instrumentation_ebpf

import (
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentation"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
)

func SetupWithManager(
	mgr ctrl.Manager,
	configUpdates chan<- instrumentation.ConfigUpdate[ebpf.K8sConfigGroup],
	InstrumentationRequests chan<- instrumentation.Request[ebpf.K8sProcessGroup, ebpf.K8sConfigGroup, *ebpf.K8sProcessDetails],
	distributionGetter *distros.Getter,
) error {
	commonlogger.Logger().Info("Starting reconcilers for ebpf instrumentation")
	var err error

	err = builder.
		ControllerManagedBy(mgr).
		Named("InstrumentationConfigReconciler_ebpf").
		For(&odigosv1.InstrumentationConfig{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(&InstrumentationConfigReconciler{
			Client:                  mgr.GetClient(),
			Scheme:                  mgr.GetScheme(),
			ConfigUpdates:           configUpdates,
			InstrumentationRequests: InstrumentationRequests,
			DistributionGetter:      distributionGetter,
		})
	if err != nil {
		return err
	}

	return nil
}
