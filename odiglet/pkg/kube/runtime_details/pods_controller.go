package runtime_details

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	criwrapper "github.com/odigos-io/odigos/k8sutils/pkg/cri"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	batchv1 "k8s.io/api/batch/v1"
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
	CriClient *criwrapper.CriClient

	// map where keys are the names of the environment variables that participate in append mechanism
	// they need to be recorded by runtime detection into the runtime info, and this list instruct what to collect.
	RuntimeDetectionEnvs map[string]struct{}
}

func IsCronJobBackedWorkload(ctx context.Context, c client.Client, pw *k8sconsts.PodWorkload) (bool, error) {
	if pw.Kind != k8sconsts.WorkloadKindJob {
		return false, nil
	}

	var job batchv1.Job
	err := c.Get(ctx, client.ObjectKey{Name: pw.Name, Namespace: pw.Namespace}, &job)
	if err != nil {
		return false, fmt.Errorf("failed to get Job: %w", err)
	}

	for _, ownerRef := range job.OwnerReferences {
		if ownerRef.Controller != nil && *ownerRef.Controller && ownerRef.Kind == "CronJob" {
			return true, nil
		}
	}

	return false, nil
}

func GetCronJobOwnerName(ctx context.Context, c client.Client, pw *k8sconsts.PodWorkload) (string, error) {
	// Only care about Jobs
	if pw.Kind != k8sconsts.WorkloadKindJob {
		return "", nil
	}

	// Fetch the Job object
	var job batchv1.Job
	if err := c.Get(ctx, client.ObjectKey{
		Namespace: pw.Namespace,
		Name:      pw.Name,
	}, &job); err != nil {
		return "", fmt.Errorf("failed to get Job %s/%s: %w", pw.Namespace, pw.Name, err)
	}

	// Look for a controlling OwnerReference of kind CronJob
	for _, owner := range job.OwnerReferences {
		if owner.Controller != nil && *owner.Controller && owner.Kind == "CronJob" {
			return owner.Name, nil
		}
	}

	// No CronJob owner found
	return "", nil
}

// We need to apply runtime details detection for a new running pod in the following cases:
// 1. User Instrument the pod for the first time
func (p *PodsReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	var pod corev1.Pod
	err := p.Client.Get(ctx, request.NamespacedName, &pod)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	podWorkload, err := getPodWorkloadObject(&pod)
	if err != nil {
		logger.Error(err, "error getting pod workload object")
		return reconcile.Result{}, err
	}
	if podWorkload == nil {
		// pod is not managed by a workload, no runtime details detection needed
		return reconcile.Result{}, nil
	}

	logger.Info(fmt.Sprintf("Received podWorkload! %+v", podWorkload))

	if podWorkload.Kind == k8sconsts.WorkloadKindJob {
		/* Test if job's owner is cronJob */
		isCronJob, err := IsCronJobBackedWorkload(ctx, p.Client, podWorkload)
		if err != nil {
			logger.Error(err, "Failed to determine of pod owned by CronJob")
			return reconcile.Result{}, err
		}

		if isCronJob {
			logger.Info(fmt.Sprintf("Cron job was detected correctly! %+v", podWorkload))
			jobOwnerName, err := GetCronJobOwnerName(ctx, p.Client, podWorkload)
			if err != nil {
				return reconcile.Result{}, err
			}

			podWorkload = &k8sconsts.PodWorkload{
				Name:      jobOwnerName,
				Kind:      k8sconsts.WorkloadKindCronJob,
				Namespace: podWorkload.Namespace,
			}
		} else {
			/* Regular jobs not supported */
			return reconcile.Result{}, nil
		}
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

	logger.V(0).Info("Completed runtime details detection for a new running pod", "name", request.Name, "namespace", request.Namespace, "runtimeResults", runtimeResults)
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
