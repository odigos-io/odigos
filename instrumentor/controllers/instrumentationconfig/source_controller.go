package instrumentationconfig

import (
	"context"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// These controllers handle update of the InstrumentationConfig's ServiceName
// whenever there are changes in the associated workloads (Deployments, DaemonSets, StatefulSets).
type SourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *SourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Source object", "name", req.Name, "namespace", req.Namespace)
	source := &odigosv1alpha1.Source{}
	err := r.Get(ctx, req.NamespacedName, source)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if source.Spec.Workload.Kind == workload.WorkloadKindNamespace {
		// Namespace Source does not contain configuration for a specific workload
		return ctrl.Result{}, nil
	}

	if odigosv1alpha1.IsDisabledSource(source) {
		// Exclude this workload from instrumentation, no need to update InstrumentationConfig
		return ctrl.Result{}, nil
	}

	instConfigName := workload.CalculateWorkloadRuntimeObjectName(source.Spec.Workload.Name, source.Spec.Workload.Kind)
	return r.updateInstrumentationConfigServiceName(ctx, instConfigName, req.Namespace, source.Spec.ReportedName)
}

func (r *SourceReconciler) updateInstrumentationConfigServiceName(ctx context.Context, instConfigName, namespace string, serviceName string) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	instConfig := &odigosv1alpha1.InstrumentationConfig{}
	err := r.Get(ctx, types.NamespacedName{Name: instConfigName, Namespace: namespace}, instConfig)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if instConfig.Spec.ServiceName != serviceName {
		instConfig.Spec.ServiceName = serviceName
		logger.Info("Updating InstrumentationConfig service name", "instrumentationConfig", instConfigName, "namespace", namespace, "serviceName", serviceName)
		err = r.Update(ctx, instConfig)
		return utils.K8SUpdateErrorHandler(err)
	}

	return reconcile.Result{}, nil
}
