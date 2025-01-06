package runtime_details

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/procdiscovery/pkg/libc"

	procdiscovery "github.com/odigos-io/odigos/procdiscovery/pkg/process"

	"github.com/odigos-io/odigos/odiglet/pkg/process"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"github.com/odigos-io/odigos/common/utils"
	criwrapper "github.com/odigos-io/odigos/k8sutils/pkg/cri"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	kubeutils "github.com/odigos-io/odigos/odiglet/pkg/kube/utils"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

	runtimeResults, err := runtimeInspection(ctx, pods, odigosConfig.IgnoredContainers, nil)
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

func runtimeInspection(ctx context.Context, pods []corev1.Pod, ignoredContainers []string, criClient *criwrapper.CriClient) ([]odigosv1.RuntimeDetailsByContainer, error) {
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
			var detectedAgent *odigosv1.OtherAgent
			var libcType *common.LibCType

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

				// Languages that can be detected using environment variables, e.g Python<>newrelic
				for envName := range inspectProc.Environments.DetailedEnvs {
					if otherAgentName, exists := procdiscovery.OtherAgentEnvs[envName]; exists {
						detectedAgent = &odigosv1.OtherAgent{Name: otherAgentName}
					}
				}
				// Languages that can be detected using command line Substrings, e.g. Java<>newrelic
				for otherAgentCmdSubstring, otherAgentName := range procdiscovery.OtherAgentCmdSubString {
					if strings.Contains(inspectProc.CmdLine, otherAgentCmdSubstring) {
						detectedAgent = &odigosv1.OtherAgent{Name: otherAgentName}
					}
				}

				// Inspecting libc type is expensive and not relevant for all languages
				if libc.ShouldInspectForLanguage(programLanguageDetails.Language) {
					typeFound, err := libc.InspectType(inspectProc)
					if err == nil {
						libcType = typeFound
					} else {
						log.Logger.Error(err, "error inspecting libc type", "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
					}
				}
			}

			var runtimeVersion string
			if programLanguageDetails.RuntimeVersion != nil {
				runtimeVersion = programLanguageDetails.RuntimeVersion.String()
			}

			resultsMap[container.Name] = odigosv1.RuntimeDetailsByContainer{
				ContainerName:  container.Name,
				Language:       programLanguageDetails.Language,
				RuntimeVersion: runtimeVersion,
				EnvVars:        envs,
				OtherAgent:     detectedAgent,
				LibCType:       libcType,
			}

			if criClient != nil && inspectProc != nil { // CriClient passed as nil in cases that will be deprecated in the future [InstrumentedApplication]
				procEnvVars := inspectProc.Environments.OverwriteEnvs
				updateRuntimeDetailsWithContainerRuntimeEnvs(ctx, *criClient, pod, container, programLanguageDetails, &resultsMap, procEnvVars)
			}

		}
	}

	results := make([]odigosv1.RuntimeDetailsByContainer, 0, len(resultsMap))
	for _, value := range resultsMap {
		results = append(results, value)
	}

	return results, nil
}

// updateRuntimeDetailsWithContainerRuntimeEnvs checks if relevant environment variables are set in the Runtime
// and updates the RuntimeDetailsByContainer accordingly.
func updateRuntimeDetailsWithContainerRuntimeEnvs(ctx context.Context, criClient criwrapper.CriClient, pod corev1.Pod, container corev1.Container,
	programLanguageDetails common.ProgramLanguageDetails, resultsMap *map[string]odigosv1.RuntimeDetailsByContainer, procEnvVars map[string]string) {
	// Retrieve environment variable names for the specified language
	envVarNames, exists := envOverwrite.EnvVarsForLanguage[programLanguageDetails.Language]
	if !exists {
		return
	}

	// Verify if environment variables already exist in the container manifest.
	// If they exist, set the RuntimeUpdateState as ProcessingStateSkipped.
	// there's no need to fetch them from the Container Runtime, and we will just append our additions in the webhook.
	if envsExistsInManifest := checkEnvVarsInContainerManifest(container, envVarNames); envsExistsInManifest {
		runtimeDetailsByContainer := (*resultsMap)[container.Name]
		state := odigosv1.ProcessingStateSkipped
		runtimeDetailsByContainer.RuntimeUpdateState = &state
		(*resultsMap)[container.Name] = runtimeDetailsByContainer
		return
	}

	// Environment variables do not exist in the manifest; fetch them from the container's Runtime
	fetchAndSetEnvFromContainerRuntime(ctx, criClient, pod, container, envVarNames, resultsMap, procEnvVars)
}

// fetchAndSetEnvFromContainerRuntime retrieves environment variables from the container's runtime and updates the runtime details.
func fetchAndSetEnvFromContainerRuntime(ctx context.Context, criClient criwrapper.CriClient, pod corev1.Pod, container corev1.Container,
	envVarKeys []string, resultsMap *map[string]odigosv1.RuntimeDetailsByContainer, procEnvVars map[string]string) {
	containerID := getContainerID(pod.Status.ContainerStatuses, container.Name)
	if containerID == "" {
		log.Logger.V(0).Info("containerID not found for container", "container", container.Name, "pod", pod.Name, "namespace", pod.Namespace)
		return
	}
	envVars, err := criClient.GetContainerEnvVarsList(ctx, envVarKeys, containerID)
	runtimeDetailsByContainer := (*resultsMap)[container.Name]

	var state odigosv1.ProcessingState

	if err != nil {
		// If the CRI request fails, we can still attempt to check the /proc/<pid>/environ file.
		// This is only applicable if the relevant value in /proc is EMPTY and we are certain it wasn't present in the manifest (as indicated by reaching this point in the code).
		// In such cases, we can mark the state as `ProcessingStateSucceeded` and proceed without setting any environment variables.
		for _, envVarKey := range envVarKeys {
			procEnvVarValue, exists := procEnvVars[envVarKey]
			if !exists || procEnvVarValue == "" {
				state = odigosv1.ProcessingStateSucceeded
			} else {
				state = odigosv1.ProcessingStateFailed
				// In Java, there are two potential relevant environment variables. If either of them exists or is not nil, we cannot consider the process as succeeded.
				break
			}
		}

		log.Logger.Error(err, "failed to get relevant env var per language from CRI", "container", container.Name, "pod", pod.Name, "namespace", pod.Namespace)
		errMessage := fmt.Sprintf("CRI communication error for container %s in pod %s/%s",
			container.Name, pod.Namespace, pod.Name)

		runtimeDetailsByContainer.CriErrorMessage = &errMessage

	} else {
		state = odigosv1.ProcessingStateSucceeded
		runtimeDetailsByContainer.EnvFromContainerRuntime = envVars
	}

	runtimeDetailsByContainer.RuntimeUpdateState = &state

	// Update the results map with the modified runtime details
	(*resultsMap)[container.Name] = runtimeDetailsByContainer
}

// getContainerID retrieves the container ID for a given container name from the list of container statuses.
func getContainerID(containerStatuses []corev1.ContainerStatus, containerName string) string {
	for _, containerStatus := range containerStatuses {
		if containerStatus.Name == containerName {
			return containerStatus.ContainerID
		}
	}
	return ""
}

func checkEnvVarsInContainerManifest(container corev1.Container, envVarNames []string) bool {
	// Create a map for quick lookup of envVar names
	envVarSet := make(map[string]struct{})
	for _, name := range envVarNames {
		envVarSet[name] = struct{}{}
	}

	// Iterate over the container's environment variables
	for _, containerEnvVar := range container.Env {
		if _, exists := envVarSet[containerEnvVar.Name]; exists {
			return true
		}
	}
	return false
}

func persistRuntimeDetailsToInstrumentationConfig(ctx context.Context, kubeclient client.Client, instrumentationConfig *odigosv1.InstrumentationConfig, newStatus odigosv1.InstrumentationConfigStatus) error {
	// This come to make sure we're updating instrumentationConfig only once (at the first time)
	currentConfig := &odigosv1.InstrumentationConfig{}
	err := kubeclient.Get(ctx, client.ObjectKeyFromObject(instrumentationConfig), currentConfig)
	if err != nil {
		return fmt.Errorf("failed to retrieve current InstrumentationConfig: %w", err)
	}

	// Verify if the RuntimeDetailsByContainer already set.
	// If it has, skip updating the RuntimeDetails to ensure the new runtime detection is performed only once.
	if len(currentConfig.Status.RuntimeDetailsByContainer) > 0 {
		return nil
	}

	// persist the runtime results into the status of the instrumentation config
	patchStatus := odigosv1.InstrumentationConfig{
		Status: newStatus,
	}
	patchData, err := json.Marshal(patchStatus)
	if err != nil {
		return err
	}
	err = kubeclient.Status().Patch(ctx, instrumentationConfig, client.RawPatch(types.MergePatchType, patchData), client.FieldOwner("odiglet-runtimedetails"))
	if err != nil {
		return err
	}

	return nil
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
