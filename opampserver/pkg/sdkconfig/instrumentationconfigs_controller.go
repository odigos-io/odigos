package sdkconfig

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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
		// TODO: signal the agent to stop collection?
		// it should be restarted after some time, but until then it can be nice to have it disabled
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	podWorkload, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(req.Name, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	i.ConnectionCache.UpdateWorkloadRemoteConfig(podWorkload, instrumentationConfig.Spec.SdkConfigs, instrumentationConfig.Spec.Containers)

	return ctrl.Result{}, nil
}

func (i *InstrumentationConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("opampserver-instrumentationconfig").
		For(&odigosv1.InstrumentationConfig{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(i)
}
