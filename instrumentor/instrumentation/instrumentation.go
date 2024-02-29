package instrumentation

import (
	"fmt"
	"strings"

	"github.com/keyval-dev/odigos/common/envOverwrite"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func ApplyInstrumentationDevicesToPodTemplate(original *v1.PodTemplateSpec, runtimeDetails *odigosv1.InstrumentedApplication, defaultSdks map[common.ProgrammingLanguage]common.OtelSdk) error {

	// delete any existing instrumentation devices.
	// this is necessary for example when migrating from community to enterprise,
	// and we need to cleanup the community device before adding the enterprise one.
	Revert(original)

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

		// Overwrite env var if needed
		for i, envVar := range container.Env {
			if envOverwrite.ShouldOverwrite(envVar.Name) {
				container.Env[i].Value = envOverwrite.Patch(envVar.Name, envVar.Value, otelSdk.SdkType)
			}
		}

		modifiedContainers = append(modifiedContainers, container)
	}

	original.Spec.Containers = modifiedContainers
	return nil
}

func Revert(original *v1.PodTemplateSpec) {
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

		for i, envVar := range container.Env {
			if envOverwrite.ShouldOverwrite(envVar.Name) {
				container.Env[i].Value = envOverwrite.Revert(envVar.Name, envVar.Value)
			}
		}
	}
}

func getLanguageOfContainer(instrumentation *odigosv1.InstrumentedApplication, containerName string) *common.ProgrammingLanguage {
	for _, l := range instrumentation.Spec.Languages {
		if l.ContainerName == containerName {
			return &l.Language
		}
	}

	return nil
}
