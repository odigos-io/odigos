package instrumentation

import (
	"errors"
	"fmt"
	"strings"

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

func ApplyInstrumentationDevicesToPodTemplate(original *corev1.PodTemplateSpec, runtimeDetails *odigosv1.InstrumentedApplication, defaultSdks map[common.ProgrammingLanguage]common.OtelSdk, targetObj client.Object) error {
	// delete any existing instrumentation devices.
	// this is necessary for example when migrating from community to enterprise,
	// and we need to cleanup the community device before adding the enterprise one.
	RevertInstrumentationDevices(original)

	manifestEnvOriginal, err := envoverwrite.NewOrigWorkloadEnvValues(targetObj.GetAnnotations())
	if err != nil {
		return err
	}

	var modifiedContainers []corev1.Container
	for _, container := range original.Spec.Containers {
		containerLanguage := getLanguageOfContainer(runtimeDetails, container.Name)
		if containerLanguage == nil || *containerLanguage == common.UnknownProgrammingLanguage || *containerLanguage == common.IgnoredProgrammingLanguage || *containerLanguage == common.NginxProgrammingLanguage {
			// always patch the env vars, even if the language is unknown or ignored.
			// this is necessary to sync the existing envs with the missing language if changed for any reason.
			err = patchEnvVarsForContainer(runtimeDetails, &container, nil, *containerLanguage, manifestEnvOriginal)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrPatchEnvVars, err)
			}
			modifiedContainers = append(modifiedContainers, container)
			continue
		}

		otelSdk, found := defaultSdks[*containerLanguage]
		if !found {
			return fmt.Errorf("%w for language: %s, container:%s", ErrNoDefaultSDK, *containerLanguage, container.Name)
		}

		instrumentationDeviceName := common.InstrumentationDeviceName(*containerLanguage, otelSdk)

		if container.Resources.Limits == nil {
			container.Resources.Limits = make(map[corev1.ResourceName]resource.Quantity)
		}
		container.Resources.Limits[corev1.ResourceName(instrumentationDeviceName)] = resource.MustParse("1")

		err = patchEnvVarsForContainer(runtimeDetails, &container, &otelSdk, *containerLanguage, manifestEnvOriginal)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrPatchEnvVars, err)
		}

		modifiedContainers = append(modifiedContainers, container)
	}

	original.Spec.Containers = modifiedContainers

	// persist the original values if changed
	manifestEnvOriginal.SerializeToAnnotation(targetObj)
	return nil
}

// this function restores a workload manifest env vars to their original values.
// it is used when the instrumentation is removed from the workload.
// the original values are read from the annotation which was saved when the instrumentation was applied.
func RevertEnvOverwrites(obj client.Object, podSpec *corev1.PodTemplateSpec) error {
	manifestEnvOriginal, err := envoverwrite.NewOrigWorkloadEnvValues(obj.GetAnnotations())
	if err != nil {
		return err
	}

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
			} else {
				newContainerEnvs = append(newContainerEnvs, envVar)
			}
		}
		podSpec.Spec.Containers[iContainer].Env = newContainerEnvs
	}

	manifestEnvOriginal.DeleteFromObj(obj)

	return nil
}

func RevertInstrumentationDevices(original *corev1.PodTemplateSpec) {
	for _, container := range original.Spec.Containers {
		for resourceName := range container.Resources.Limits {
			if strings.HasPrefix(string(resourceName), common.OdigosResourceNamespace) {
				delete(container.Resources.Limits, resourceName)
			}
		}
		// Is it needed?
		for resourceName := range container.Resources.Requests {
			if strings.HasPrefix(string(resourceName), common.OdigosResourceNamespace) {
				delete(container.Resources.Requests, resourceName)
			}
		}
	}
}

func getLanguageOfContainer(instrumentation *odigosv1.InstrumentedApplication, containerName string) *common.ProgrammingLanguage {
	for _, l := range instrumentation.Spec.RuntimeDetails {
		if l.ContainerName == containerName {
			return &l.Language
		}
	}

	return nil
}

// getEnvVarsOfContainer returns the env vars which are defined for the given container and are used for instrumentation purposes.
// This function also returns env vars which are declared in the container build.
func getEnvVarsOfContainer(instrumentation *odigosv1.InstrumentedApplication, containerName string) map[string]string {
	envVars := make(map[string]string)

	for _, l := range instrumentation.Spec.RuntimeDetails {
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
func patchEnvVarsForContainer(runtimeDetails *odigosv1.InstrumentedApplication, container *corev1.Container, sdk *common.OtelSdk, programmingLanguage common.ProgrammingLanguage, manifestEnvOriginal *envoverwrite.OrigWorkloadEnvValues) error {

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
func RemoveInjectInstrumentationLabel(original *corev1.PodTemplateSpec) {
	if original.Labels != nil {
		delete(original.Labels, "odigos.io/inject-instrumentation")
	}
}
