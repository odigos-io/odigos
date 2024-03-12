package runtime_details

import (
	"context"

	"github.com/go-logr/logr"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/common/utils"
	kubeutils "github.com/keyval-dev/odigos/odiglet/pkg/kube/utils"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"github.com/keyval-dev/odigos/odiglet/pkg/process"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/inspectors"
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

func runtimeInspection(pods []corev1.Pod) ([]common.LanguageByContainer, error) {
	resultsMap := make(map[string]common.LanguageByContainer)
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

			detectionResults := inspectors.DetectLanguage(processes)
			if len(detectionResults) == 0 {
				log.Logger.V(0).Info("no supported language detected for container in pod", "processes", processes, "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
				continue
			}

			if len(detectionResults) > 1 {
				log.Logger.V(0).Info("multiple languages detected for pod container processes, selecting first one", "processes", processes, "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
			}

			resultsMap[container.Name] = common.LanguageByContainer{
				ContainerName: container.Name,
				Language:      detectionResults[0].Language,
			}
		}
	}

	results := make([]common.LanguageByContainer, 0, len(resultsMap))
	for _, value := range resultsMap {
		results = append(results, value)
	}

	return results, nil
}

func persistRuntimeResults(ctx context.Context, results []common.LanguageByContainer, owner client.Object, kubeClient client.Client, scheme *runtime.Scheme) error {
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
		updatedIa.Spec.Languages = results
		return nil
	})

	if operationResult != controllerutil.OperationResultNone {
		log.Logger.V(0).Info("updated runtime info", "result", operationResult, "name", owner.GetName(), "kind",
			owner.GetObjectKind().GroupVersionKind().Kind, "namespace", owner.GetNamespace())
	}
	return nil
}
