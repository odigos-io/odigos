package runtime_details

import (
	"context"
	"errors"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	criwrapper "github.com/odigos-io/odigos/k8sutils/pkg/cri"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	k8scontainer "github.com/odigos-io/odigos/k8sutils/pkg/container"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
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

	// the clientset is used to interact with the k8s API directly,
	// without pulling in specific objects into the controller runtime cache
	// which can be expensive (memory and CPU)
	Clientset *kubernetes.Clientset
	CriClient *criwrapper.CriClient
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

	// find pods that are managed by the workload,
	// filter out pods that are being deleted or not ready,
	// note that the controller-runtime cache is assumed here to only contain pods in the same node as the odiglet
	var podList corev1.PodList
	err = r.List(ctx, &podList, client.InNamespace(instrumentationConfig.Namespace))
	if err != nil {
		return reconcile.Result{}, err
	}

	var selectedPods []corev1.Pod
	for _, pod := range podList.Items {
		// skip pods that are being deleted or not ready
		if pod.DeletionTimestamp != nil || !k8scontainer.AllContainersReady(&pod) {
			continue
		}
		podWorkload, err := getPodWorkloadObject(&pod)
		if errors.Is(err, workload.ErrKindNotSupported) {
			continue
		}
		if podWorkload == nil {
			// pod is not managed by a workload, no runtime details detection needed
			continue
		}

		// get instrumentation config name for the pod
		instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(podWorkload.Name, podWorkload.Kind)
		if instrumentationConfigName == instrumentationConfig.Name {
			selectedPods = append(selectedPods, pod)
		}
	}

	odigosConfig, err := k8sutils.GetCurrentOdigosConfig(ctx, r.Client)
	if err != nil {
		return k8sutils.K8SNoEffectiveConfigErrorHandler(err)
	}

	runtimeResults, err := runtimeInspection(ctx, selectedPods, odigosConfig.IgnoredContainers, r.CriClient)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = persistRuntimeDetailsToInstrumentationConfig(ctx, r.Client, &instrumentationConfig, odigosv1.InstrumentationConfigStatus{
		RuntimeDetailsByContainer: runtimeResults,
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	logger.Info("Completed runtime detection for new instrumentation config", "namespace", request.Namespace, "name", request.Name)
	return reconcile.Result{}, nil
}

