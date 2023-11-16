package kube

import (
	"context"

	"github.com/go-logr/logr"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/common/consts"
	"github.com/keyval-dev/odigos/common/utils"
	"github.com/keyval-dev/odigos/odiglet/pkg/env"
	"github.com/keyval-dev/odigos/odiglet/pkg/inspectors"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"github.com/keyval-dev/odigos/odiglet/pkg/process"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func inspectRuntimesOfRunningPods(ctx context.Context, logger *logr.Logger, labels map[string]string,
	kubeClient client.Client, scheme *runtime.Scheme, object client.Object) (ctrl.Result, error) {
	pods, err := getRunningPods(ctx, labels, object.GetNamespace(), kubeClient)
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
		for _, c := range pod.Spec.Containers {
			// Skip Go instrumentation container
			if c.Image == consts.GolangInstrumentationImage {
				continue
			}

			processes, err := process.FindAllInContainer(string(pod.UID), c.Name)
			if err != nil {
				log.Logger.Error(err, "Failed to find processes")
				return nil, err
			}
			if processes != nil && len(processes) > 0 {
				processResults, processName := inspectors.DetectLanguage(processes)
				if len(processResults) > 0 {
					resultsMap[c.Name] = common.LanguageByContainer{
						ContainerName: c.Name,
						Language:      processResults[0],
						ProcessName:   processName,
					}
				} else {
					log.Logger.V(0).Info("unrecognized processes", "processes", processes, "pod", pod.Name, "container", c.Name, "namespace", pod.Namespace)
				}
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

func getRunningPods(ctx context.Context, labels map[string]string, ns string, kubeClient client.Client) ([]corev1.Pod, error) {
	var podList corev1.PodList
	err := kubeClient.List(ctx, &podList, client.MatchingLabels(labels), client.InNamespace(ns))

	var filteredPods []corev1.Pod
	for _, pod := range podList.Items {
		if isPodInCurrentNode(&pod) && pod.Status.Phase == corev1.PodRunning {
			filteredPods = append(filteredPods, pod)
		}
	}

	if err != nil {
		return nil, err
	}

	return filteredPods, nil
}

func isPodInCurrentNode(pod *corev1.Pod) bool {
	return pod.Spec.NodeName == env.Current.NodeName
}
