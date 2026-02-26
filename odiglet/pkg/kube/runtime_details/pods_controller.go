package runtime_details

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	criwrapper "github.com/odigos-io/odigos/k8sutils/pkg/cri"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type PodsReconciler struct {
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

// We need to apply runtime details detection for a new running pod in the following cases:
// 1. User Instrument the pod for the first time
func (p *PodsReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := commonlogger.FromContext(ctx).With(
		"controller", "odiglet-runtime-details-pods",
		"namespace", request.Namespace,
		"name", request.Name,
	)
	ctx = commonlogger.IntoContext(ctx, logger)

	var pod corev1.Pod
	err := p.Client.Get(ctx, request.NamespacedName, &pod)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	podWorkload, err := workload.PodWorkloadObject(ctx, &pod)
	if err != nil {
		logger.Error("error getting pod workload object", "err", err)
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

	// Perform runtime inspection once we know the pod is newer that the latest runtime inspection performed and saved.
	runtimeResults, err := runtimeInspection(ctx, []corev1.Pod{pod}, p.CriClient, p.RuntimeDetectionEnvs)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = persistRuntimeDetailsToInstrumentationConfig(ctx, p.Client, &instrumentationConfig, runtimeResults)
	if err != nil {
		return reconcile.Result{}, err
	}

	logger.Info("Completed runtime details detection for a new running pod", "name", request.Name, "namespace", request.Namespace, "runtimeResults", runtimeResults)
	return reconcile.Result{}, nil
}

func InstrumentationConfigContainsUnknownLanguage(config odigosv1.InstrumentationConfig) bool {
	for _, containerDetails := range config.Status.RuntimeDetailsByContainer {
		if containerDetails.Language == common.UnknownProgrammingLanguage {
			return true
		}
	}
	return false
}
