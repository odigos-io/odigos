package runtime_details

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type PodsReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// the clientset is used to interact with the k8s API directly,
	// without pulling in specific objects into the controller runtime cache
	// which can be expensive (memory and CPU)
	Clientset *kubernetes.Clientset
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
	podGeneration, err := GetPodGeneration(ctx, p.Clientset, &pod)
	if err != nil {
		return reconcile.Result{}, err
	}

	// prevent runtime inspection on pods for which we already have the runtime details for this generation
	// if instrumentation config contains unknown language we need to re-inspect the pod
	failedToGetPodGeneration := podGeneration == 0
	isNewPodGeneration := podGeneration > instrumentationConfig.Status.ObservedWorkloadGeneration
	instrumentedConfigContainUnknown := InstrumentationConfigContainsUnknownLanguage(instrumentationConfig)

	shouldSkipDetection := failedToGetPodGeneration || (!isNewPodGeneration && !instrumentedConfigContainUnknown)

	if shouldSkipDetection {
		logger.V(3).Info("skipping redundant runtime details detection since generation is not newer", "name", request.Name, "namespace", request.Namespace, "currentPodGeneration", podGeneration, "observedWorkloadGeneration", instrumentationConfig.Status.ObservedWorkloadGeneration)
		return reconcile.Result{}, nil
	}

	odigosConfig, err := k8sutils.GetCurrentOdigosConfig(ctx, p.Client)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Perform runtime inspection once we know the pod is newer that the latest runtime inspection performed and saved.
	runtimeResults, err := runtimeInspection([]corev1.Pod{pod}, odigosConfig.IgnoredContainers)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = persistRuntimeDetailsToInstrumentationConfig(ctx, p.Client, &instrumentationConfig, odigosv1.InstrumentationConfigStatus{
		RuntimeDetailsByContainer:  runtimeResults,
		ObservedWorkloadGeneration: podGeneration,
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	logger.V(0).Info("Completed runtime details detection for a new running pod", "name", request.Name, "namespace", request.Namespace, "generation", podGeneration)
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

func InstrumentationConfigContainsUnknownLanguage(config odigosv1.InstrumentationConfig) bool {
	for _, containerDetails := range config.Status.RuntimeDetailsByContainer {
		if containerDetails.Language == common.UnknownProgrammingLanguage {
			return true
		}
	}
	return false
}
