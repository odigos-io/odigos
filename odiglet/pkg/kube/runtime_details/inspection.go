package runtime_details

import (
	"context"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/procdiscovery/pkg/libc"

	procdiscovery "github.com/odigos-io/odigos/procdiscovery/pkg/process"

	"github.com/odigos-io/odigos/odiglet/pkg/process"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/envOverwrite"
	criwrapper "github.com/odigos-io/odigos/k8sutils/pkg/cri"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	kubeutils "github.com/odigos-io/odigos/odiglet/pkg/kube/utils"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func runtimeInspection(ctx context.Context, pods []corev1.Pod, criClient *criwrapper.CriClient) ([]odigosv1.RuntimeDetailsByContainer, error) {
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

			programLanguageDetails := common.ProgramLanguageDetails{Language: common.UnknownProgrammingLanguage}
			var inspectProc *procdiscovery.Details
			var detectErr error

			for _, proc := range processes {
				containerURL := kubeutils.GetPodExternalURL(pod.Status.PodIP, container.Ports)
				programLanguageDetails, detectErr = inspectors.DetectLanguage(proc, containerURL, log.Logger)
				if detectErr == nil && programLanguageDetails.Language != common.UnknownProgrammingLanguage {
					inspectProc = &proc
					break
				}
			}

			envs := make([]odigosv1.EnvVar, 0)
			var detectedAgent *odigosv1.OtherAgent
			var libcType *common.LibCType
			var secureExecutionMode *bool

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

				secureExecutionMode = inspectProc.SecureExecutionMode
			}

			var runtimeVersion string
			if programLanguageDetails.RuntimeVersion != nil {
				runtimeVersion = programLanguageDetails.RuntimeVersion.String()
			}

			resultsMap[container.Name] = odigosv1.RuntimeDetailsByContainer{
				ContainerName:       container.Name,
				Language:            programLanguageDetails.Language,
				RuntimeVersion:      runtimeVersion,
				EnvVars:             envs,
				OtherAgent:          detectedAgent,
				LibCType:            libcType,
				SecureExecutionMode: secureExecutionMode,
			}

			if inspectProc != nil {
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

	envVarNames = append(envVarNames, consts.LdPreloadEnvVarName)

	// Verify if environment variables already exist in the container manifest.
	// If they exist, set the RuntimeUpdateState as ProcessingStateSkipped.
	if envsExistsInManifest := checkEnvVarsInContainerManifest(container, envVarNames); envsExistsInManifest {
		runtimeDetailsByContainer := (*resultsMap)[container.Name]
		state := odigosv1.ProcessingStateSkipped
		runtimeDetailsByContainer.RuntimeUpdateState = &state
		(*resultsMap)[container.Name] = runtimeDetailsByContainer
	}

	// Environment variables do not exist in the manifest; fetch them from the container's Image
	fetchAndSetEnvFromContainerRuntime(ctx, criClient, pod, container, envVarNames, resultsMap, procEnvVars)
}

// fetchAndSetEnvFromContainerRuntime retrieves environment variables from the container's Image and updates the runtime details.
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

func persistRuntimeDetailsToInstrumentationConfig(ctx context.Context, kubeclient client.Client, instrumentationConfig *odigosv1.InstrumentationConfig, newRuntimeDetials []odigosv1.RuntimeDetailsByContainer) error {

	// fetch a fresh copy of instrumentation config.
	// TODO: is this necessary? can we do it with the existing object?
	currentConfig := &odigosv1.InstrumentationConfig{}
	err := kubeclient.Get(ctx, client.ObjectKeyFromObject(instrumentationConfig), currentConfig)
	if err != nil {
		return fmt.Errorf("failed to retrieve current InstrumentationConfig: %w", err)
	}

	// Verify if the RuntimeDetailsByContainer already set.
	// If it has, skip updating the RuntimeDetails to ensure the new runtime detection is performed only once.
	// In some cases we would like to update the existing RuntimeDetailsByContainer:
	// 1. LD_PRELOAD is identified in EnvVars [/proc/pid/environ]
	// 2. LD_PRELOAD is identified in EnvFromContainerRuntime [DockerFile]
	// 3. SecureExecutionMode is set to true.
	// 4. RuntimeVersion changes
	if len(currentConfig.Status.RuntimeDetailsByContainer) > 0 {
		updated := false
		for _, newDetail := range newRuntimeDetials {
			for j := range currentConfig.Status.RuntimeDetailsByContainer {
				existingDetail := &currentConfig.Status.RuntimeDetailsByContainer[j]
				if newDetail.ContainerName == existingDetail.ContainerName {
					if mergeRuntimeDetails(existingDetail, newDetail) {
						updated = true
					}
				}
			}
		}
		// Do not overwrite existing details if no updates are needed
		if !updated {
			return nil
		}
	} else {
		// First time setting the values
		currentConfig.Status.RuntimeDetailsByContainer = newRuntimeDetials
	}

	meta.SetStatusCondition(&currentConfig.Status.Conditions, metav1.Condition{
		Type:    odigosv1.RuntimeDetectionStatusConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  string(odigosv1.RuntimeDetectionReasonDetectedSuccessfully),
		Message: "runtime detection completed successfully",
	})

	err = kubeclient.Status().Update(ctx, currentConfig)
	if err != nil {
		return err
	}

	return nil
}

func GetRuntimeDetails(ctx context.Context, kubeClient client.Client, podWorkload *k8sconsts.PodWorkload) (*odigosv1.InstrumentationConfig, error) {
	instrumentedApplicationName := workload.CalculateWorkloadRuntimeObjectName(podWorkload.Name, podWorkload.Kind)

	var runtimeDetails odigosv1.InstrumentationConfig
	err := kubeClient.Get(ctx, client.ObjectKey{
		Namespace: podWorkload.Namespace,
		Name:      instrumentedApplicationName,
	}, &runtimeDetails)
	if err != nil {
		return nil, err
	}

	return &runtimeDetails, nil
}

func mergeRuntimeDetails(existing *odigosv1.RuntimeDetailsByContainer, new odigosv1.RuntimeDetailsByContainer) bool {
	updated := false

	// 1. Merge LD_PRELOAD from EnvVars [/proc/pid/environ]
	odigosStr := "odigos"
	updated = mergeLdPreloadEnvVars(new.EnvVars, &existing.EnvVars, &odigosStr)

	// 2. Merge LD_PRELOAD from EnvFromContainerRuntime [DockerFile]
	updated = mergeLdPreloadEnvVars(new.EnvFromContainerRuntime, &existing.EnvFromContainerRuntime, nil)

	// 3. Update SecureExecutionMode if needed
	if existing.SecureExecutionMode == nil && new.SecureExecutionMode != nil {
		existing.SecureExecutionMode = new.SecureExecutionMode
		updated = true
	}

	// 4. Update RuntimeVersion if different
	if new.RuntimeVersion != "" && new.RuntimeVersion != existing.RuntimeVersion {
		existing.RuntimeVersion = new.RuntimeVersion
		updated = true
	}

	return updated
}

func mergeLdPreloadEnvVars(
	newEnvs []odigosv1.EnvVar,
	existingEnvs *[]odigosv1.EnvVar,
	skipIfContains *string,
) bool {
	// Step 1: Check if LD_PRELOAD already exists in the existing envs
	for _, existingEnv := range *existingEnvs {
		if existingEnv.Name == consts.LdPreloadEnvVarName {
			return false // Already present, nothing to do
		}
	}

	// Step 2: Try to add it from new envs
	for _, newEnv := range newEnvs {
		if newEnv.Name == consts.LdPreloadEnvVarName {
			if skipIfContains == nil || !strings.Contains(newEnv.Value, *skipIfContains) {
				*existingEnvs = append(*existingEnvs, newEnv)
				return true // Add LD_PRELOAD and return
			}
		}
	}
	return false // No LD_PRELOAD found, nothing to do
}
