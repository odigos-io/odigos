package runtime_details

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/procdiscovery/pkg/libc"

	procdiscovery "github.com/odigos-io/odigos/procdiscovery/pkg/process"

	"github.com/odigos-io/odigos/odiglet/pkg/process"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/envOverwrite"
	criwrapper "github.com/odigos-io/odigos/k8sutils/pkg/cri"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	kubecommon "github.com/odigos-io/odigos/odiglet/pkg/kube/common"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var errNoKnownLanguageDetected = errors.New("no known programming language detected in the container")

// relevantProcessesDetailsInContainer filters the processes in a container to find those relevant:
// a relevant process is one that matches the selected programming language for the container.
// The selected programming language is determined based on the known languages detected in the container's processes.
// If multiple languages are detected, specific rules are applied to select the main language.
// Some combinations of detected languages result in an error, as they indicate ambiguity in determining the main language.
func relevantProcessesDetailsInContainer(knownLangByPid map[int]common.ProgramLanguageDetails, processes []procdiscovery.Details) ([]procdiscovery.Details, common.ProgramLanguageDetails, error) {
	uniqueLangs := make(map[common.ProgrammingLanguage]struct{})
	for _, langDetails := range knownLangByPid {
		uniqueLangs[langDetails.Language] = struct{}{}
	}

	uniqueLangsSlice := make([]common.ProgrammingLanguage, 0, len(uniqueLangs))
	for lang := range uniqueLangs {
		if lang != common.UnknownProgrammingLanguage {
			uniqueLangsSlice = append(uniqueLangsSlice, lang)
		}
	}

	// resolve the language detected for the container
	// depending on the number of unique languages detected and their types
	selectedLangDetails := common.ProgramLanguageDetails{Language: common.UnknownProgrammingLanguage}
	switch len(uniqueLangsSlice) {
	case 0:
		return nil, selectedLangDetails, errNoKnownLanguageDetected
	case 1:
		selectedLangDetails.Language = uniqueLangsSlice[0]
	case 2:
		switch {
		// c++ can be a wrapper of script etc.
		// we want to detect the "later" language to get the real application.
		// but we also want to detect c++ if it is the only language detected.
		// hence if c++ is detected with another language, we select the other language.
		case uniqueLangsSlice[0] == common.CPlusPlusProgrammingLanguage:
			selectedLangDetails.Language = uniqueLangsSlice[1]
		case uniqueLangsSlice[1] == common.CPlusPlusProgrammingLanguage:
			selectedLangDetails.Language = uniqueLangsSlice[0]
		default:
			return nil, selectedLangDetails, fmt.Errorf("two different programming languages detected in the same container, cannot determine the main language: %v", uniqueLangsSlice)
		}
	default:
		return nil, selectedLangDetails, fmt.Errorf("more than two programming languages detected in the same container, cannot determine the main language: %v", uniqueLangsSlice)
	}

	// construct the list of relevant processes and determine runtime version if possible
	// relevant processes are those that match the selected programming language
	relevantProcesses := make([]procdiscovery.Details, 0)
	uniqueRuntimeVersions := make(map[string]struct{})
	for _, proc := range processes {
		if langDetails, exists := knownLangByPid[proc.ProcessID]; exists && langDetails.Language == selectedLangDetails.Language {
			relevantProcesses = append(relevantProcesses, proc)
			if langDetails.RuntimeVersion != "" {
				uniqueRuntimeVersions[langDetails.RuntimeVersion] = struct{}{}
			}
		}
	}

	if len(uniqueRuntimeVersions) == 1 {
		for version := range uniqueRuntimeVersions {
			selectedLangDetails.RuntimeVersion = version
		}
	}

	return relevantProcesses, selectedLangDetails, nil
}

func runtimeInspection(ctx context.Context, pods []corev1.Pod, criClient *criwrapper.CriClient, runtimeDetectionEnvs map[string]struct{}) ([]odigosv1.RuntimeDetailsByContainer, error) {
	resultsMap := make(map[string]odigosv1.RuntimeDetailsByContainer)
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			processes, err := process.FindAllInContainer(workload.PodUID(&pod), container.Name, runtimeDetectionEnvs)
			if err != nil {
				log.Logger.Error(err, "failed to find processes in pod container", "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
				return nil, err
			}
			if len(processes) == 0 {
				log.Logger.V(0).Info("no processes found in pod container", "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
				continue
			}

			// map of known programming languages detected by pid in this container
			knownLangsByPid := make(map[int]common.ProgramLanguageDetails)

			for _, proc := range processes {
				containerURL := kubecommon.GetPodExternalURL(pod.Status.PodIP, container.Ports)
				langDetails, detectErr := inspectors.DetectLanguage(proc, containerURL, log.Logger)
				if detectErr == nil && langDetails.Language != common.UnknownProgrammingLanguage {
					knownLangsByPid[proc.ProcessID] = langDetails
				}
			}

			// resolve relevant processes and main language for the container
			relevantProcesses, langDetails, err := relevantProcessesDetailsInContainer(knownLangsByPid, processes)
			if err != nil {
				switch {
				case errors.Is(err, errNoKnownLanguageDetected):
					log.Logger.V(0).Info("unable to detect language for any process", "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace, "processes", processes)
				default:
					log.Logger.Error(err, "error determining relevant processes and main language", "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
				}
				langDetails.Language = common.UnknownProgrammingLanguage
			}

			envs := make([]odigosv1.EnvVar, 0)
			var detectedAgent *odigosv1.OtherAgent
			var libcType *common.LibCType
			var secureExecutionMode *bool
			var inspectProc *procdiscovery.Details

			if len(relevantProcesses) == 0 || langDetails.Language == common.UnknownProgrammingLanguage {
				log.Logger.V(0).Info("unable to detect language for any process", "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace, "processes", processes)
				langDetails.Language = common.UnknownProgrammingLanguage
			} else {
				if len(relevantProcesses) > 1 {
					log.Logger.V(0).Info("multiple processes found in pod container, only taking the first one with detected language into account", "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
				}

				// Convert map to slice for k8s format
				inspectProc = &relevantProcesses[0]
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

				// Agent that can be detected using environment variables
				val, ok := inspectProc.Environments.OverwriteEnvs[consts.LdPreloadEnvVarName]
				if ok && strings.Contains(val, procdiscovery.DynatraceFullStackEnvValuePrefix) {
					detectedAgent = &odigosv1.OtherAgent{Name: procdiscovery.DynatraceAgentName}
				}

				// Inspecting libc type is expensive and not relevant for all languages
				if libc.ShouldInspectForLanguage(langDetails.Language) {
					typeFound, err := libc.InspectType(inspectProc)
					if err == nil {
						libcType = typeFound
					} else {
						log.Logger.Error(err, "error inspecting libc type", "pod", pod.Name, "container", container.Name, "namespace", pod.Namespace)
					}
				}

				secureExecutionMode = inspectProc.SecureExecutionMode
			}

			resultsMap[container.Name] = odigosv1.RuntimeDetailsByContainer{
				ContainerName:       container.Name,
				Language:            langDetails.Language,
				RuntimeVersion:      langDetails.RuntimeVersion,
				EnvVars:             envs,
				OtherAgent:          detectedAgent,
				LibCType:            libcType,
				SecureExecutionMode: secureExecutionMode,
			}

			if inspectProc != nil {
				procEnvVars := inspectProc.Environments.OverwriteEnvs
				updateRuntimeDetailsWithContainerRuntimeEnvs(ctx, *criClient, pod, container, langDetails, &resultsMap, procEnvVars)
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
		var criErrorMessage *string
		// If the CRI request fails, we can still attempt to check the /proc/<pid>/environ file.
		// This is only applicable if the relevant value in /proc is EMPTY and we are certain it wasn't present in the manifest (as indicated by reaching this point in the code).
		// In such cases, we can mark the state as `ProcessingStateSucceeded` and proceed without setting any environment variables.
		for _, envVarKey := range envVarKeys {
			procEnvVarValue, exists := procEnvVars[envVarKey]
			if !exists || procEnvVarValue == "" {
				state = odigosv1.ProcessingStateSucceeded
			} else {
				state = odigosv1.ProcessingStateFailed
				errMessage := fmt.Sprintf("CRI communication error for container %s in pod %s/%s",
					container.Name, pod.Namespace, pod.Name)
				criErrorMessage = &errMessage
				break
			}
		}

		log.Logger.Error(err, "failed to get relevant env var per language from CRI", "container", container.Name, "pod", pod.Name, "namespace", pod.Namespace)

		runtimeDetailsByContainer.CriErrorMessage = criErrorMessage

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
					podKey := strings.Join([]string{currentConfig.Namespace, currentConfig.Name}, "/")
					if mergeRuntimeDetails(existingDetail, newDetail, podKey) {
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

func mergeRuntimeDetails(existing *odigosv1.RuntimeDetailsByContainer, new odigosv1.RuntimeDetailsByContainer, podIdentintifier string) bool {

	// Skip merging if languages are different, except when updating from unknown to known language.
	if new.Language != existing.Language &&
		!(new.Language != common.UnknownProgrammingLanguage && existing.Language == common.UnknownProgrammingLanguage) {
		log.Logger.V(0).Info("detected different language, skipping merge runtime details", "pod_identifier", podIdentintifier, "container_name", new.ContainerName, "new.Language", new.Language, "existing.Language", existing.Language)
		return false
	}

	// Overwrite the existing env vars. they always reflect the current state of the container.
	// 1. Merge LD_PRELOAD from EnvVars [/proc/pid/environ]
	odigosStr := "odigos"
	mergedEnvVars, updatedEnviron := mergeLdPreloadEnvVars(new.EnvVars, existing.EnvVars, &odigosStr)
	existing.EnvVars = mergedEnvVars

	// 2. Merge LD_PRELOAD from EnvFromContainerRuntime [DockerFile]
	mergedEnvFromContainerRuntime, updatedDocker := mergeLdPreloadEnvVars(new.EnvFromContainerRuntime, existing.EnvFromContainerRuntime, nil)
	existing.EnvFromContainerRuntime = mergedEnvFromContainerRuntime

	updated := updatedEnviron || updatedDocker

	// 3. Update SecureExecutionMode if needed
	existingSecureExecution := existing.SecureExecutionMode != nil && *existing.SecureExecutionMode
	newSecureExecution := new.SecureExecutionMode != nil && *new.SecureExecutionMode
	if !existingSecureExecution && newSecureExecution {
		existing.SecureExecutionMode = new.SecureExecutionMode
		updated = true
	}

	// 4. Update RuntimeVersion if different
	if new.RuntimeVersion != "" && new.RuntimeVersion != existing.RuntimeVersion {
		existing.RuntimeVersion = new.RuntimeVersion
		updated = true
	}

	// 5. Update Language if different
	if new.Language != existing.Language {
		existing.Language = new.Language
		updated = true
	}

	// 6. Update OtherAgent if there is any difference between the existing and new values.
	// This includes three cases:
	// 1. existing.OtherAgent is nil but new.OtherAgent is not (addition),
	// 2. existing.OtherAgent is not nil but new.OtherAgent is nil (removal),
	// 3. both are non-nil but their .Name fields differ (modification).
	if (existing.OtherAgent == nil && new.OtherAgent != nil) ||
		(existing.OtherAgent != nil && new.OtherAgent == nil) ||
		(existing.OtherAgent != nil && new.OtherAgent != nil && existing.OtherAgent.Name != new.OtherAgent.Name) {
		existing.OtherAgent = new.OtherAgent
		updated = true
	}

	return updated
}

func mergeLdPreloadEnvVars(
	newEnvs []odigosv1.EnvVar,
	existingEnvs []odigosv1.EnvVar,
	skipIfContains *string,
) ([]odigosv1.EnvVar, bool) {

	newLdPreloadValue, newHasLdPreload := env.FindLdPreloadInEnvs(newEnvs)
	_, existingHasLdPreload := env.FindLdPreloadInEnvs(existingEnvs)

	if newHasLdPreload && existingHasLdPreload {
		// Already present, nothing to do.
		// Amir 01/07/2025: do we need to update the existing envs value if it changes?
		return existingEnvs, false
	}

	if newHasLdPreload && !existingHasLdPreload {
		// Avoid adding LD_PRELOAD if it contains odigos value.
		// Amir 01/07/2025: the consumer (agentenabled controllers) already checks for this value.
		// can we simply log what we have and let downstream filter it or not depending on the usecase?
		if skipIfContains != nil && strings.Contains(newLdPreloadValue, *skipIfContains) {
			return existingEnvs, false
		}
		// New LD_PRELOAD is set, add it to the existing envs.
		envsWithLdPreload := append(existingEnvs, odigosv1.EnvVar{Name: consts.LdPreloadEnvVarName, Value: newLdPreloadValue})
		return envsWithLdPreload, true
	}

	if !newHasLdPreload && existingHasLdPreload {
		// at this point, if we have an existing LD_PRELOAD, we never remove it.
		// this is to prevent loops where the value jitters and we end up
		// enabling and disabling the agent and causing a lot of noise and rollout.
		// the downside is that if a user has LD_PRELOAD and removes it,
		// odigos will falsly show this as if LD_PRELOAD is still set.
		// in this case, user currently has no other way then to uninstrument and re-instrument.
		// TODO: this is a bad UX to the user, consider how to update this value live.
		return existingEnvs, true
	}

	// At this point, we have no new nor existing LD_PRELOAD, so nothing to do.
	return existingEnvs, false
}
