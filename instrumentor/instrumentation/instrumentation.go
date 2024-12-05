package instrumentation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/envoverwrite"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	ErrNoDefaultSDK = errors.New("no default sdks found")
	ErrPatchEnvVars = errors.New("failed to patch env vars")
)

func ApplyInstrumentationDevicesToPodTemplate(original *corev1.PodTemplateSpec, instConfig *odigosv1.InstrumentationConfig, defaultSdks map[common.ProgrammingLanguage]common.OtelSdk, targetObj client.Object,
	logger logr.Logger) (error, bool, bool) {
	// delete any existing instrumentation devices.
	// this is necessary for example when migrating from community to enterprise,
	// and we need to cleanup the community device before adding the enterprise one.
	RevertInstrumentationDevices(original)

	// don't use the runtime detection if it's from an older generation
	// as it might inject irrelevant env values into the workload manifest.
	if instConfig.Status.ObservedWorkloadGeneration != targetObj.GetGeneration() {
		logger.Info("Skipping applying instrumentation devices to workload manifest due to generation mismatch", "observedGeneration", instConfig.Status.ObservedWorkloadGeneration, "currentGeneration", targetObj.GetGeneration())
		return nil, false, false
	}

	deviceApplied := false
	deviceSkippedDueToOtherAgent := false
	var modifiedContainers []corev1.Container

	manifestEnvOriginal, err := envoverwrite.NewOrigWorkloadEnvValues(targetObj.GetAnnotations())
	if err != nil {
		return err, deviceApplied, deviceSkippedDueToOtherAgent
	}

	for _, container := range original.Spec.Containers {
		containerLanguage := getLanguageOfContainer(instConfig, container.Name)
		containerHaveOtherAgent := getContainerOtherAgents(instConfig, container.Name)

		// In case there is another agent in the container, we should not apply the instrumentation device.
		if containerLanguage == common.PythonProgrammingLanguage && containerHaveOtherAgent != nil {
			logger.Info("Python container has other agent, skip applying instrumentation device", "agent", containerHaveOtherAgent.Name, "container", container.Name)

			// Not actually modifying the container, but we need to append it to the list.
			modifiedContainers = append(modifiedContainers, container)
			deviceSkippedDueToOtherAgent = true
			continue

		}
		// handle containers with unknown language or ignored language
		if containerLanguage == common.UnknownProgrammingLanguage || containerLanguage == common.IgnoredProgrammingLanguage || containerLanguage == common.NginxProgrammingLanguage {
			// always patch the env vars, even if the language is unknown or ignored.
			// this is necessary to sync the existing envs with the missing language if changed for any reason.
			err = patchEnvVarsForContainer(instConfig, &container, nil, containerLanguage, manifestEnvOriginal)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrPatchEnvVars, err), deviceApplied, deviceSkippedDueToOtherAgent
			}
			modifiedContainers = append(modifiedContainers, container)
			continue
		}

		// Find and apply the appropriate SDK for the container language.
		otelSdk, found := defaultSdks[containerLanguage]
		if !found {
			return fmt.Errorf("%w for language: %s, container:%s", ErrNoDefaultSDK, containerLanguage, container.Name), deviceApplied, deviceSkippedDueToOtherAgent
		}

		instrumentationDeviceName := common.InstrumentationDeviceName(containerLanguage, otelSdk)
		if container.Resources.Limits == nil {
			container.Resources.Limits = make(map[corev1.ResourceName]resource.Quantity)
		}
		container.Resources.Limits[corev1.ResourceName(instrumentationDeviceName)] = resource.MustParse("1")
		deviceApplied = true

		err = patchEnvVarsForContainer(instConfig, &container, &otelSdk, containerLanguage, manifestEnvOriginal)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrPatchEnvVars, err), deviceApplied, deviceSkippedDueToOtherAgent
		}

		modifiedContainers = append(modifiedContainers, container)
	}

	if modifiedContainers != nil {
		original.Spec.Containers = modifiedContainers
	}

	// persist the original values if changed
	manifestEnvOriginal.SerializeToAnnotation(targetObj)

	// if non of the devices were applied due to the presence of another agent, return an error.
	if !deviceApplied && deviceSkippedDueToOtherAgent {
		return fmt.Errorf("device not added to any container due to the presence of another agent"), false, deviceSkippedDueToOtherAgent
	}

	// devicePartiallyApplied is used to indicate that the instrumentation device was partially applied for some of the containers.
	devicePartiallyApplied := deviceSkippedDueToOtherAgent && deviceApplied

	return nil, deviceApplied, devicePartiallyApplied
}

// this function restores a workload manifest env vars to their original values.
// it is used when the instrumentation is removed from the workload.
// the original values are read from the annotation which was saved when the instrumentation was applied.
func RevertEnvOverwrites(obj client.Object, podSpec *corev1.PodTemplateSpec) (bool, error) {
	manifestEnvOriginal, err := envoverwrite.NewOrigWorkloadEnvValues(obj.GetAnnotations())
	if err != nil {
		return false, err
	}

	changed := false
	for iContainer, c := range podSpec.Spec.Containers {
		containerOriginalEnv := manifestEnvOriginal.GetContainerStoredEnvs(c.Name)
		newContainerEnvs := make([]corev1.EnvVar, 0, len(c.Env))
		for _, envVar := range c.Env {
			if origValue, found := containerOriginalEnv[envVar.Name]; found {
				// revert the env var to its original value
				if origValue != nil {
					newContainerEnvs = append(newContainerEnvs, corev1.EnvVar{
						Name:  envVar.Name,
						Value: *containerOriginalEnv[envVar.Name],
					})
				} else {
					// if the value is nil, the env var was not set by the user to begin with.
					// we will simply not append it to the new envs to achieve the same effect.
				}
				changed = true
			} else {
				newContainerEnvs = append(newContainerEnvs, envVar)
			}
		}
		podSpec.Spec.Containers[iContainer].Env = newContainerEnvs
	}

	annotationRemoved := manifestEnvOriginal.DeleteFromObj(obj)

	return changed || annotationRemoved, nil
}

func RevertInstrumentationDevices(original *corev1.PodTemplateSpec) bool {
	changed := false
	for _, container := range original.Spec.Containers {
		for resourceName := range container.Resources.Limits {
			if strings.HasPrefix(string(resourceName), common.OdigosResourceNamespace) {
				delete(container.Resources.Limits, resourceName)
				changed = true
			}
		}
		// Is it needed?
		for resourceName := range container.Resources.Requests {
			if strings.HasPrefix(string(resourceName), common.OdigosResourceNamespace) {
				delete(container.Resources.Requests, resourceName)
				changed = true
			}
		}
	}
	return changed
}

func getLanguageOfContainer(instrumentation *odigosv1.InstrumentationConfig, containerName string) common.ProgrammingLanguage {
	for _, l := range instrumentation.Status.RuntimeDetailsByContainer {
		if l.ContainerName == containerName {
			return l.Language
		}
	}

	return common.UnknownProgrammingLanguage
}

func getContainerOtherAgents(instrumentation *odigosv1.InstrumentationConfig, containerName string) *odigosv1.OtherAgent {
	for _, l := range instrumentation.Status.RuntimeDetailsByContainer {
		if l.ContainerName == containerName {
			if l.OtherAgent != nil && *l.OtherAgent != (odigosv1.OtherAgent{}) {
				return l.OtherAgent
			}
		}
	}
	return nil
}

// getEnvVarsOfContainer returns the env vars which are defined for the given container and are used for instrumentation purposes.
// This function also returns env vars which are declared in the container build.
func getEnvVarsOfContainer(instrumentation *odigosv1.InstrumentationConfig, containerName string) map[string]string {
	envVars := make(map[string]string)

	for _, l := range instrumentation.Status.RuntimeDetailsByContainer {
		if l.ContainerName == containerName {
			for _, env := range l.EnvVars {
				envVars[env.Name] = env.Value
			}
			return envVars
		}
	}

	return envVars
}

// when otelsdk is nil, it means that the container is not instrumented.
// this will trigger reverting of any existing env vars which were set by odigos before.
func patchEnvVarsForContainer(runtimeDetails *odigosv1.InstrumentationConfig, container *corev1.Container, sdk *common.OtelSdk, programmingLanguage common.ProgrammingLanguage, manifestEnvOriginal *envoverwrite.OrigWorkloadEnvValues) error {

	observedEnvs := getEnvVarsOfContainer(runtimeDetails, container.Name)

	// Step 1: check existing environment on the manifest and update them if needed
	newEnvs := make([]corev1.EnvVar, 0, len(container.Env))
	for _, envVar := range container.Env {

		// extract the observed value for this env var, which might be empty if not currently exists
		observedEnvValue := observedEnvs[envVar.Name]

		desiredEnvValue := envOverwrite.GetPatchedEnvValue(envVar.Name, observedEnvValue, sdk, programmingLanguage)

		if desiredEnvValue == nil {
			// no need to patch this env var, so make sure it is reverted to its original value
			origValue, found := manifestEnvOriginal.RemoveOriginalValue(container.Name, envVar.Name)
			if !found {
				newEnvs = append(newEnvs, envVar)
			} else { // found, we need to update the env var to it's original value
				if origValue != nil {
					// this case reverts back the env var to it's original value
					newEnvs = append(newEnvs, corev1.EnvVar{
						Name:  envVar.Name,
						Value: *origValue,
					})
				} else {
					// if the original value was nil, then it was not set by the user.
					// we will simply not append it to the new envs to achieve the same effect.
				}
			}
		} else { // there is a desired value to inject
			// if it's the first time we patch this env var, save the original value
			manifestEnvOriginal.InsertOriginalValue(container.Name, envVar.Name, &envVar.Value)
			// update the env var to it's desired value
			newEnvs = append(newEnvs, corev1.EnvVar{
				Name:  envVar.Name,
				Value: *desiredEnvValue,
			})
		}

		// If an env var is defined both in the container build and in the container spec, the value in the container spec will be used.
		delete(observedEnvs, envVar.Name)
	}

	// Step 2: add the new env vars which odigos might patch, but which are not defined in the manifest
	if sdk != nil {
		for envName, envValue := range observedEnvs {
			desiredEnvValue := envOverwrite.GetPatchedEnvValue(envName, envValue, sdk, programmingLanguage)
			if desiredEnvValue != nil {
				// store that it was empty to begin with
				manifestEnvOriginal.InsertOriginalValue(container.Name, envName, nil)
				// and add this new env var to the manifest
				newEnvs = append(newEnvs, corev1.EnvVar{
					Name:  envName,
					Value: *desiredEnvValue,
				})
			}
		}
	}

	// Step 3: update the container with the new env vars
	container.Env = newEnvs

	return nil
}

func SetInjectInstrumentationLabel(original *corev1.PodTemplateSpec) {
	odigosTier := env.GetOdigosTierFromEnv()

	// inject the instrumentation annotation for oss tier only
	if odigosTier == common.CommunityOdigosTier {
		if original.Labels == nil {
			original.Labels = make(map[string]string)
		}
		original.Labels["odigos.io/inject-instrumentation"] = "true"
	}
}

// RemoveInjectInstrumentationLabel removes the "odigos.io/inject-instrumentation" label if it exists.
func RemoveInjectInstrumentationLabel(original *corev1.PodTemplateSpec) bool {
	if original.Labels != nil {
		if _, ok := original.Labels["odigos.io/inject-instrumentation"]; ok {
			delete(original.Labels, "odigos.io/inject-instrumentation")
			return true
		}
	}
	return false
}
