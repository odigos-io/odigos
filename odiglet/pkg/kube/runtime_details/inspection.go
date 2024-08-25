package runtime_details

import (
	"context"
	"errors"

	procdiscovery "github.com/odigos-io/odigos/procdiscovery/pkg/process"

	"github.com/odigos-io/odigos/odiglet/pkg/process"

	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	kubeutils "github.com/odigos-io/odigos/odiglet/pkg/kube/utils"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var errNoPodsFound = errors.New("no pods found")

func ignoreNoPodsFoundError(err error) error {
	if err.Error() == errNoPodsFound.Error() {
		return nil
	}
	return err
}

func inspectRuntimesOfRunningPods(ctx context.Context, logger *logr.Logger, labels map[string]string,
	kubeClient client.Client, scheme *runtime.Scheme, object client.Object) error {
	pods, err := kubeutils.GetRunningPods(ctx, labels, object.GetNamespace(), kubeClient)
	if err != nil {
		logger.Error(err, "error fetching running pods")
		return err
	}

	if len(pods) == 0 {
		return errNoPodsFound
	}

	odigosConfig, err := k8sutils.GetCurrentOdigosConfig(ctx, kubeClient)
	if err != nil {
		logger.Error(err, "failed to get odigos config")
		return err
	}

	runtimeResults, err := runtimeInspection(pods, odigosConfig.IgnoredContainers)
	if err != nil {
		logger.Error(err, "error inspecting pods")
		return err
	}

	err = persistRuntimeResults(ctx, runtimeResults, object, kubeClient, scheme)
	if err != nil {
		logger.Error(err, "error persisting runtime results")
		return err
	}

	return nil
}

func runtimeInspection(pods []corev1.Pod, ignoredContainers []string) ([]odigosv1.RuntimeDetailsByContainer, error) {
	resultsMap := make(map[string]odigosv1.RuntimeDetailsByContainer)
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {

			// Skip ignored containers, but label them as ignored
			if utils.IsItemIgnored(container.Name, ignoredContainers) {
				resultsMap[container.Name] = odigosv1.RuntimeDetailsByContainer{
					ContainerName: container.Name,
					Language:      common.IgnoredProgrammingLanguage,
				}
				continue
			}

			processes, err := process.FindAllInContainer(string(pod.UID), container.Name)
			if err != nil {
				log.Logger.Error(err, "failed to find processes in pod container", "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
				return nil, err
			}
			if len(processes) == 0 {
				log.Logger.V(0).Info("no processes found in pod container", "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
				continue
			}

			programLanguageDetails := common.ProgramLanguageDetails{Language: common.UnknownProgrammingLanguage}
			var inspectProc *procdiscovery.Details
			var detectErr error

			for _, proc := range processes {
				containerURL := kubeutils.GetPodExternalURL(pod.Status.PodIP, container.Ports)
				programLanguageDetails, detectErr = inspectors.DetectLanguage(proc, containerURL)
				if detectErr == nil && programLanguageDetails.Language != common.UnknownProgrammingLanguage {
					inspectProc = &proc
					break
				}
			}

			envs := make([]odigosv1.EnvVar, 0)
			if inspectProc == nil {
				log.Logger.V(0).Info("unable to detect language for any process", "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
				programLanguageDetails.Language = common.UnknownProgrammingLanguage
			} else {
				if len(processes) > 1 {
					log.Logger.V(0).Info("multiple processes found in pod container, only taking the first one with detected language into account", "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
				}

				// Convert map to slice for k8s format
				envs = make([]odigosv1.EnvVar, 0, len(inspectProc.Environments.DetailedEnvs))
				for envName, envValue := range inspectProc.Environments.OverwriteEnvs {
					envs = append(envs, odigosv1.EnvVar{Name: envName, Value: envValue})
				}
			}

			resultsMap[container.Name] = odigosv1.RuntimeDetailsByContainer{
				ContainerName:  container.Name,
				Language:       programLanguageDetails.Language,
				RuntimeVersion: programLanguageDetails.RuntimeVersion,
				EnvVars:        envs,
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
			Name:      workload.CalculateWorkloadRuntimeObjectName(owner.GetName(), owner.GetObjectKind().GroupVersionKind().Kind),
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

func GetRuntimeDetails(ctx context.Context, kubeClient client.Client, podWorkload *workload.PodWorkload) (*odigosv1.InstrumentedApplication, error) {
	instrumentedApplicationName := workload.CalculateWorkloadRuntimeObjectName(podWorkload.Name, podWorkload.Kind)

	var runtimeDetails odigosv1.InstrumentedApplication
	err := kubeClient.Get(ctx, client.ObjectKey{
		Namespace: podWorkload.Namespace,
		Name:      instrumentedApplicationName,
	}, &runtimeDetails)
	if err != nil {
		return nil, err
	}

	return &runtimeDetails, nil
}
