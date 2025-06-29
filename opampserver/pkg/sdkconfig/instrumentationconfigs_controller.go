package sdkconfig

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// Only handle events where the agent injection is enabled
type agentEnabledPredicate struct{}

func (i *agentEnabledPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}

	ic, ok := e.Object.(*odigosv1.InstrumentationConfig)
	if !ok {
		return false
	}
	return ic.Spec.AgentInjectionEnabled
}

func (i *agentEnabledPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectNew == nil {
		return false
	}

	ic, ok := e.ObjectNew.(*odigosv1.InstrumentationConfig)
	if !ok {
		return false
	}

	return ic.Spec.AgentInjectionEnabled
}

func (i *agentEnabledPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (i *agentEnabledPredicate) Generic(e event.GenericEvent) bool {
	return false
}

var _ predicate.Predicate = &agentEnabledPredicate{}

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
		// if agent is disabled, we don't need to reconcile as all config is disabled.
		// TODO: in the future, notify the agent to stop collection until a rollout is triggered?
		// TODO2: do it also when object is deleted
		// TODO3: refine the condition so it will be triggered even less
		WithEventFilter(&agentEnabledPredicate{}).
		Complete(i)
}
