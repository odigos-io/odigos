package runtime_details

import (
	"context"
	"encoding/json"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type podPredicate struct{}

func (p *podPredicate) Create(e event.CreateEvent) bool {
	// this function is called when a new entity is created in the controller-runtime cache.
	// 1. When odiglet restarts, it will receive create events for all running pods.
	// 2. When a new pod is created in k8s, should not have status phase as running, but it does not harm to check.
	pod, ok := e.Object.(*corev1.Pod)
	if !ok {
		return false
	}
	return pod.Status.Phase == corev1.PodRunning
}

func (p *podPredicate) Update(e event.UpdateEvent) bool {
	// this function is called when an entity is updated in the controller-runtime cache.
	// We care about pod once it's status changes from not running to running -
	// this is the point in time we want to apply runtime details detection.
	oldPod, oldOk := e.ObjectOld.(*corev1.Pod)
	newPod, newOk := e.ObjectNew.(*corev1.Pod)

	if !oldOk || !newOk {
		return false
	}

	return oldPod.Status.Phase != corev1.PodRunning && newPod.Status.Phase == corev1.PodRunning
}

func (p *podPredicate) Delete(e event.DeleteEvent) bool {
	// no runtime details detection needed when a pod is deleted
	return false
}

func (p *podPredicate) Generic(e event.GenericEvent) bool {
	// no runtime details detection needed for generic events
	return false
}

type PodsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// We need to apply runtime details detection for a new running pod in the following cases:
// 1. When a new workload generation is applied, the runtime details might be changed (different env, versions, etc).
// 2. When a source is added, but there are no running pods yet. When the first pod starts running, this is chance to apply runtime details detection.
func (p *PodsReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	var pod corev1.Pod
	err := p.Client.Get(ctx, request.NamespacedName, &pod)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	podWorkload, err := p.getPodWorkloadObject(ctx, &pod)
	if err != nil {
		logger.Error(err, "error getting pod workload object")
		return reconcile.Result{}, err
	}
	if podWorkload == nil {
		// pod is not managed by a workload, no runtime details detection needed
		return reconcile.Result{}, nil
	}

	// get instrumentation config for the pod to check if it is instrumented or not
	instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(podWorkload.Name, podWorkload.Kind)
	instrumentationConfig := odigosv1.InstrumentationConfig{}
	err = p.Client.Get(ctx, client.ObjectKey{Name: instrumentationConfigName, Namespace: podWorkload.Namespace}, &instrumentationConfig)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// The reconcile should be called every time the pod is updated and changes from not running to running

	odigosConfig, err := k8sutils.GetCurrentOdigosConfig(ctx, p.Client)
	if err != nil {
		return reconcile.Result{}, err
	}

	runtimeResults, err := runtimeInspection([]corev1.Pod{pod}, odigosConfig.IgnoredContainers)
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
	err = p.Client.Status().Patch(ctx, &instrumentationConfig, client.RawPatch(types.MergePatchType, patchData), client.FieldOwner("odiglet-runtimedetails"))
	if err != nil {
		return reconcile.Result{}, err
	}

	logger.V(0).Info("Completed runtime details detection for a new running pod", "name", request.Name, "namespace", request.Namespace)

	return reconcile.Result{}, nil
}

func (p *PodsReconciler) getPodWorkloadObject(ctx context.Context, pod *corev1.Pod) (*workload.PodWorkload, error) {
	for _, owner := range pod.OwnerReferences {
		workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(owner)
		if err != nil {
			return nil, workload.IgnoreErrorKindNotSupported(err)
		}

		return &workload.PodWorkload{
			Name:      workloadName,
			Kind:      workloadKind,
			Namespace: pod.Namespace,
		}, nil
	}

	// Pod does not necessarily have to be managed by a controller
	return nil, nil
}
