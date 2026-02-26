package runtime_details

import (
	"context"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	criwrapper "github.com/odigos-io/odigos/k8sutils/pkg/cri"
	kubecommon "github.com/odigos-io/odigos/odiglet/pkg/kube/common"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type instrumentationConfigPredicate struct{}

func (p *instrumentationConfigPredicate) Create(e event.CreateEvent) bool {
	obj, ok := e.Object.(*odigosv1.InstrumentationConfig)
	if !ok {
		return false
	}
	// we only care about new InstrumentationConfig objects
	// the event will be triggered also when odiglet starts, and the instrumentation config is created
	// in controller-runtime cache
	// checking the RuntimeDetailsByContainer map should filter cases where we do not want to trigger the reconcile
	// for workload which has already been reconciled previously
	return len(obj.Status.RuntimeDetailsByContainer) == 0
}

func (p *instrumentationConfigPredicate) Update(e event.UpdateEvent) bool {
	return false
}

func (p *instrumentationConfigPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (p *instrumentationConfigPredicate) Generic(e event.GenericEvent) bool {
	return false
}

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// the clientset is used to interact with the k8s API directly,
	// without pulling in specific objects into the controller runtime cache
	// which can be expensive (memory and CPU)
	Clientset *kubernetes.Clientset
	CriClient *criwrapper.CriClient

	// map where keys are the names of the environment variables that participate in append mechanism
	// they need to be recorded by runtime detection into the runtime info, and this list instruct what to collect.
	RuntimeDetectionEnvs map[string]struct{}
}

func (r *InstrumentationConfigReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := commonlogger.FromContext(ctx).With(
		"controller", "odiglet-runtime-details-instrumentationconfig",
		"namespace", request.Namespace,
		"name", request.Name,
	)
	ctx = commonlogger.IntoContext(ctx, logger)

	var instrumentationConfig odigosv1.InstrumentationConfig
	err := r.Get(ctx, request.NamespacedName, &instrumentationConfig)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if len(instrumentationConfig.OwnerReferences) != 1 {
		return reconcile.Result{}, fmt.Errorf("InstrumentationConfig %s/%s has %d owner references, expected 1", instrumentationConfig.Namespace, instrumentationConfig.Name, len(instrumentationConfig.OwnerReferences))
	}

	selectedPods, err := kubecommon.WorkloadPodsOnCurrentNode(r.Client, ctx, &instrumentationConfig)
	if err != nil {
		return reconcile.Result{}, err
	}

	if len(selectedPods) == 0 {
		// this node is not running any pods managed by the workload, so nothing to do
		return reconcile.Result{}, nil
	}

	runtimeResults, err := runtimeInspection(ctx, selectedPods, r.CriClient, r.RuntimeDetectionEnvs)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = persistRuntimeDetailsToInstrumentationConfig(ctx, r.Client, &instrumentationConfig, runtimeResults)
	if err != nil {
		return reconcile.Result{}, err
	}

	logger.Info("Completed runtime detection for new instrumentation config", "namespace", request.Namespace, "name", request.Name, "runtimeResults", runtimeResults)
	return reconcile.Result{}, nil
}
