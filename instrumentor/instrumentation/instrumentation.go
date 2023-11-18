package instrumentation

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type CollectorInfo struct {
	Hostname string
	Port     int
}

func ModifyObject(original *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) error {
	var modifiedContainers []v1.Container
	for _, container := range original.Spec.Containers {
		containerLanguage := getLanguageOfContainer(instrumentation, container.Name)
		if containerLanguage == nil {
			continue
		}

		instrumentationDeviceName := common.ProgrammingLanguageToInstrumentationDevice(*containerLanguage)
		if instrumentationDeviceName == "" {
			// should not happen, only for safety
			continue
		}

		if container.Resources.Limits == nil {
			container.Resources.Limits = make(map[v1.ResourceName]resource.Quantity)
		}
		container.Resources.Limits[v1.ResourceName(instrumentationDeviceName)] = resource.MustParse("1")

		modifiedContainers = append(modifiedContainers, container)
	}

	original.Spec.Containers = modifiedContainers
	return nil
}

func Revert(original *v1.PodTemplateSpec) {
	for _, instrumentationDevice := range common.InstrumentationDevices {
		removeDeviceFromPodSpec(instrumentationDevice, original)
	}
}

func removeDeviceFromPodSpec(deviceName common.OdigosInstrumentationDevice, podSpec *v1.PodTemplateSpec) {
	for _, container := range podSpec.Spec.Containers {
		delete(container.Resources.Limits, v1.ResourceName(deviceName))
		delete(container.Resources.Requests, v1.ResourceName(deviceName))
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
