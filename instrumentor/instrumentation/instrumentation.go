package instrumentation

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"sigs.k8s.io/controller-runtime/pkg/client"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func ApplyInstrumentationDevicesToPodTemplate(original *v1.PodTemplateSpec, runtimeDetails *odigosv1.InstrumentedApplication, defaultSdks map[common.ProgrammingLanguage]common.OtelSdk, targetObj client.Object) error {

	// delete any existing instrumentation devices.
	// this is necessary for example when migrating from community to enterprise,
	// and we need to cleanup the community device before adding the enterprise one.
	Revert(original, targetObj)

	var modifiedContainers []v1.Container
	for _, container := range original.Spec.Containers {
		containerLanguage := getLanguageOfContainer(runtimeDetails, container.Name)
		if containerLanguage == nil {
			modifiedContainers = append(modifiedContainers, container)
			continue
		}

		otelSdk, found := defaultSdks[*containerLanguage]
		if !found {
			return fmt.Errorf("default sdk not found for language %s", *containerLanguage)
		}

		instrumentationDeviceName := common.InstrumentationDeviceName(*containerLanguage, otelSdk)

		if container.Resources.Limits == nil {
			container.Resources.Limits = make(map[v1.ResourceName]resource.Quantity)
		}
		container.Resources.Limits[v1.ResourceName(instrumentationDeviceName)] = resource.MustParse("1")

		err := patchEnvVars(runtimeDetails, &container, targetObj)
		if err != nil {
			return fmt.Errorf("failed to patch env vars: %v", err)
		}

		modifiedContainers = append(modifiedContainers, container)
	}

	original.Spec.Containers = modifiedContainers
	return nil
}

func Revert(original *v1.PodTemplateSpec, targetObj client.Object) {
	// read the original env vars (of the manifest) from the annotation
	var origManifestEnv map[string]map[string]string
	annotations := targetObj.GetAnnotations()
	if annotations != nil {
		manifestEnvAnnotation, ok := annotations[consts.ManifestEnvOriginalValAnnotation]
		if ok {
			err := json.Unmarshal([]byte(manifestEnvAnnotation), &origManifestEnv)
			if err != nil {
				fmt.Printf("failed to unmarshal manifest env original annotation in Revert: %v", err)
			}
		}
	}

	for iContainer, container := range original.Spec.Containers {
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

		containerOriginalEnv := origManifestEnv[container.Name]
		revertedEnvVars := make([]v1.EnvVar, 0, len(container.Env))

		for _, envVar := range container.Env {
			if envOverwrite.ShouldRevert(envVar.Name, envVar.Value) {
				if origVal, ok := containerOriginalEnv[envVar.Name]; ok {
					// Revert the env var to its original value
					revertedEnvVars = append(revertedEnvVars, v1.EnvVar{
						Name:  envVar.Name,
						Value: origVal,
					})
				}
				// If the original value is not found, we are not going to add the env var back
			} else {
				revertedEnvVars = append(revertedEnvVars, envVar)
			}
		}
		original.Spec.Containers[iContainer].Env = revertedEnvVars
	}

	// Remove the annotation
	delete(annotations, consts.ManifestEnvOriginalValAnnotation)
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

func patchEnvVars(runtimeDetails *odigosv1.InstrumentedApplication, container *v1.Container, obj client.Object) error {
	envs := getEnvVarsOfContainer(runtimeDetails, container.Name)

	var manifestEnvOriginal map[string]map[string]string

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	if currentEnvAnnotation, ok := annotations[consts.ManifestEnvOriginalValAnnotation]; ok {
		// The annotation is already present, unmarshal it
		err := json.Unmarshal([]byte(currentEnvAnnotation), &manifestEnvOriginal)
		if err != nil {
			return fmt.Errorf("failed to unmarshal manifest env original annotation: %v", err)
		}
	} else {
		manifestEnvOriginal = make(map[string]map[string]string)
	}

	if _, ok := manifestEnvOriginal[container.Name]; !ok {
		manifestEnvOriginal[container.Name] = make(map[string]string)
	}

	savedEnvVar := false

	// Overwrite env var if needed
	for i, envVar := range container.Env {
		if envOverwrite.ShouldPatch(envVar.Name, envVar.Value) {
			// We are about to patch this env var, check if we need to save the original value
			// If the original value is not saved, save it to the annotation.
			if _, ok := manifestEnvOriginal[container.Name][envVar.Name]; !ok {
				savedEnvVar = true
				manifestEnvOriginal[container.Name][envVar.Name] = envVar.Value
				container.Env[i].Value = envOverwrite.Patch(envVar.Name, envVar.Value)
			}
		}
		// If an env var is defined both in the container build and in the container spec, the value in the container spec will be used.
		delete(envs, envVar.Name)
	}

	// Add the remaining env vars (which are not defined in a manifest)
	for envName, envValue := range envs {
		if envOverwrite.ShouldPatch(envName, envValue) {
			container.Env = append(container.Env, v1.EnvVar{
				Name:  envName,
				Value: envOverwrite.Patch(envName, envValue),
			})
		}
	}

	// Update the annotation with the original values from the manifest (if there are any to save)
	if savedEnvVar {
		updatedAnnotation, err := json.Marshal(manifestEnvOriginal)
		if err != nil {
			return fmt.Errorf("failed to marshal manifest env original annotation: %v", err)
		}
		annotations[consts.ManifestEnvOriginalValAnnotation] = string(updatedAnnotation)
		obj.SetAnnotations(annotations)
	}

	return nil
}
