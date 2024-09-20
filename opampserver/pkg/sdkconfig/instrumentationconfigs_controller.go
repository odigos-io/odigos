package sdkconfig

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	ConnectionCache *connection.ConnectionsCache
}

func (i *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instrumentationConfig := &odigosv1.InstrumentationConfig{}
	err := i.Get(ctx, req.NamespacedName, instrumentationConfig)

	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(req.Name)
	if err != nil {
		return ctrl.Result{}, err
	}

	podWorkload := workload.PodWorkload{
		Namespace: req.Namespace,
		Kind:      workload.WorkloadKind(workloadKind),
		Name:      workloadName,
	}

	for _, sdkConfig := range instrumentationConfig.Spec.SdkConfigs {
		i.ConnectionCache.UpdateWorkloadRemoteConfig(podWorkload, &sdkConfig)
	}

	return ctrl.Result{}, nil
}

func (i *InstrumentationConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("opampserver-instrumentationconfig").
		For(&odigosv1.InstrumentationConfig{}).
		Complete(i)
}
