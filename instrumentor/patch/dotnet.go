package patch

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	dotNetDeviceName = "instrumentation.odigos.io/dotnet"
)

var dotNet = &dotNetPatcher{}

type dotNetPatcher struct{}

func (d *dotNetPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) {
	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		if shouldPatch(instrumentation, common.DotNetProgrammingLanguage, container.Name) {
			if container.Resources.Limits == nil {
				container.Resources.Limits = make(map[v1.ResourceName]resource.Quantity)
			}

			container.Resources.Limits[dotNetDeviceName] = resource.MustParse("1")
		}

		modifiedContainers = append(modifiedContainers, container)
	}

	podSpec.Spec.Containers = modifiedContainers
}

func (d *dotNetPatcher) Revert(podSpec *v1.PodTemplateSpec) {
	removeDeviceFromPodSpec(dotNetDeviceName, podSpec)
}
