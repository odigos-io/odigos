package instrumentationconfig

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

// These controllers handle update of the InstrumentationConfig's ServiceName
// whenever there are changes in the associated Source object.
type SourceReconciler struct {
	client.Client
}

func (r *SourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	source := &odigosv1alpha1.Source{}
	err := r.Get(ctx, req.NamespacedName, source)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if source.Spec.Workload.Kind == k8sconsts.WorkloadKindNamespace {
		// Namespace Source does not contain configuration for a specific workload
		return ctrl.Result{}, nil
	}

	// if a source is disabled, an instrumentationConfig should not be present,
	// and we would get a NotFound error here.
	// if the instrumentationConfig is not deleted yet for a disabled source,
	// we would update it to have the service name, and it would be deleted by another controller.

	instConfigName := workload.CalculateWorkloadRuntimeObjectName(source.Spec.Workload.Name, source.Spec.Workload.Kind)
	instConfig := &odigosv1alpha1.InstrumentationConfig{}
	err = r.Get(ctx, types.NamespacedName{Name: instConfigName, Namespace: req.Namespace}, instConfig)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	currentServiceName := source.Spec.OtelServiceName
	if currentServiceName == "" {
		// if the user did not override the service name, we should use the name of the workload
		currentServiceName = source.Spec.Workload.Name
	}

	if instConfig.Spec.ServiceName != currentServiceName {
		instConfig.Spec.ServiceName = currentServiceName
		logger.Info("Updating InstrumentationConfig service name", "instrumentationConfig", instConfigName, "namespace", req.Namespace, "serviceName", source.Spec.OtelServiceName)
		err = r.Update(ctx, instConfig)
		return utils.K8SUpdateErrorHandler(err)
	}

	return reconcile.Result{}, nil
}
