package runtime_details

import (
	"context"
	"encoding/json"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	kubeutils "github.com/odigos-io/odigos/odiglet/pkg/kube/utils"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
}

func (r *InstrumentationConfigReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {

	logger := log.FromContext(ctx)

	var instrumentationConfig odigosv1.InstrumentationConfig
	err := r.Get(ctx, request.NamespacedName, &instrumentationConfig)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if len(instrumentationConfig.OwnerReferences) != 1 {
		return reconcile.Result{}, fmt.Errorf("InstrumentationConfig %s/%s has %d owner references, expected 1", instrumentationConfig.Namespace, instrumentationConfig.Name, len(instrumentationConfig.OwnerReferences))
	}

	workload, labels, err := getWorkloadAndLabelsfromOwner(ctx, r.Client, instrumentationConfig.Namespace, instrumentationConfig.OwnerReferences[0])
	if err != nil {
		logger.Error(err, "Failed to get workload and labels from owner")
		return reconcile.Result{}, err
	}

	pods, err := kubeutils.GetRunningPods(ctx, labels, workload.GetNamespace(), r.Client)
	if err != nil {
		return reconcile.Result{}, err
	}

	if len(pods) == 0 {
		// when a instrumentation config is created, many nodes may not have any running pods for it
		return reconcile.Result{}, nil
	}

	odigosConfig, err := k8sutils.GetCurrentOdigosConfig(ctx, r.Client)
	if err != nil {
		return reconcile.Result{}, err
	}

	runtimeResults, err := runtimeInspection(pods, odigosConfig.IgnoredContainers)
	if err != nil {
		return reconcile.Result{}, err
	}

	// persist the runtime results into the status of the instrumentation config
	patchStatus := odigosv1.InstrumentationConfig{
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: runtimeResults,
		},
	}
	patchData, err := json.Marshal(patchStatus)
	if err != nil {
		return reconcile.Result{}, err
	}
	err = r.Client.Status().Patch(ctx, &instrumentationConfig, client.RawPatch(types.MergePatchType, patchData), client.FieldOwner("odiglet-runtimedetails"))
	if err != nil {
		return reconcile.Result{}, err
	}

	logger.Info("Completed runtime detection for new instrumentation config", "namespace", request.Namespace, "name", request.Name)

	return reconcile.Result{}, nil
}
