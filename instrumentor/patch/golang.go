package patch

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	golangDeviceName = "instrumentation.odigos.io/go"
)

var golang = &golangPatcher{}

type golangPatcher struct{}

func (g *golangPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) {
	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		if shouldPatch(instrumentation, common.GoProgrammingLanguage, container.Name) {
			if container.Resources.Limits == nil {
				container.Resources.Limits = make(map[v1.ResourceName]resource.Quantity)
			}

			container.Resources.Limits[golangDeviceName] = resource.MustParse("1")
		}

		modifiedContainers = append(modifiedContainers, container)
	}

	podSpec.Spec.Containers = modifiedContainers
}

func (g *golangPatcher) Revert(podSpec *v1.PodTemplateSpec) {
	removeDeviceFromPodSpec(golangDeviceName, podSpec)
}
