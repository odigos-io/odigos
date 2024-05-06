package runtime_details

import (
	"context"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/utils"
	kubeutils "github.com/odigos-io/odigos/odiglet/pkg/kube/utils"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"github.com/odigos-io/odigos/odiglet/pkg/process"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func inspectRuntimesOfRunningPods(ctx context.Context, logger *logr.Logger, labels map[string]string,
	kubeClient client.Client, scheme *runtime.Scheme, object client.Object) (ctrl.Result, error) {
	pods, err := kubeutils.GetRunningPods(ctx, labels, object.GetNamespace(), kubeClient)
	if err != nil {
		logger.Error(err, "error fetching running pods")
		return ctrl.Result{}, err
	}

	if len(pods) == 0 {
		return ctrl.Result{}, nil
	}

	runtimeResults, err := runtimeInspection(pods)
	if err != nil {
		logger.Error(err, "error inspecting pods")
		return ctrl.Result{}, err
	}

	if len(runtimeResults) == 0 {
		return ctrl.Result{}, nil
	}

	err = persistRuntimeResults(ctx, runtimeResults, object, kubeClient, scheme)
	if err != nil {
		logger.Error(err, "error persisting runtime results")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func runtimeInspection(pods []corev1.Pod) ([]odigosv1.RuntimeDetailsByContainer, error) {
	resultsMap := make(map[string]odigosv1.RuntimeDetailsByContainer)
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {

			processes, err := process.FindAllInContainer(string(pod.UID), container.Name)
			if err != nil {
				log.Logger.Error(err, "failed to find processes in pod container", "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
				return nil, err
			}
			if len(processes) == 0 {
				log.Logger.V(0).Info("no processes found in pod container", "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
				continue
			}
			if len(processes) > 1 {
				// Currently we don't support multiple processes in the same container, where each one can have a different language
				// We only take the first process into account, when we'll support multiple processes we'll need to change this.
				log.Logger.V(0).Info("multiple processes found in pod container, only taking the first one into account", "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
			}
			process := processes[0]

			lang := inspectors.DetectLanguage(process)
			if lang == nil {
				log.Logger.V(0).Info("no supported language detected for container in pod", "process", process, "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
				continue
			}

			// Convert map to slice for k8s format
			envs := make([]odigosv1.EnvVar, 0, len(process.Envs))
			for envName, envValue := range process.Envs {
				envs = append(envs, odigosv1.EnvVar{Name: envName, Value: envValue})
			}

			resultsMap[container.Name] = odigosv1.RuntimeDetailsByContainer{
				ContainerName: container.Name,
				Language:      *lang,
				EnvVars:       envs,
			}
		}
	}

	results := make([]odigosv1.RuntimeDetailsByContainer, 0, len(resultsMap))
	for _, value := range resultsMap {
		results = append(results, value)
	}

	return results, nil
}

func persistRuntimeResults(ctx context.Context, results []odigosv1.RuntimeDetailsByContainer, owner client.Object, kubeClient client.Client, scheme *runtime.Scheme) error {
	updatedIa := &odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.GetRuntimeObjectName(owner.GetName(), owner.GetObjectKind().GroupVersionKind().Kind),
			Namespace: owner.GetNamespace(),
		},
	}

	err := controllerutil.SetControllerReference(owner, updatedIa, scheme)
	if err != nil {
		log.Logger.Error(err, "Failed to set controller reference")
		return err
	}

	operationResult, err := controllerutil.CreateOrPatch(ctx, kubeClient, updatedIa, func() error {
		updatedIa.Spec.RuntimeDetails = results
		return nil
	})

	if err != nil {
		log.Logger.Error(err, "Failed to update runtime info", "name", owner.GetName(), "kind",
			owner.GetObjectKind().GroupVersionKind().Kind, "namespace", owner.GetNamespace())
	}

	if operationResult != controllerutil.OperationResultNone {
		log.Logger.V(0).Info("updated runtime info", "result", operationResult, "name", owner.GetName(), "kind",
			owner.GetObjectKind().GroupVersionKind().Kind, "namespace", owner.GetNamespace())
	}
	return nil
}
