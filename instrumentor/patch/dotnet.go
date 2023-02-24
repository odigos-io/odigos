package patch

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var dotNet = &dotNetPatcher{}

type dotNetPatcher struct{}

func (d *dotNetPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) {
	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		if shouldPatch(instrumentation, common.DotNetProgrammingLanguage, container.Name) {
			container.Resources.Limits["instrumentation.odigos.io/dotnet"] = resource.MustParse("1")
		}

		modifiedContainers = append(modifiedContainers, container)
	}

	podSpec.Spec.Containers = modifiedContainers
}

func (d *dotNetPatcher) IsInstrumented(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) bool {
	for _, c := range podSpec.Spec.Containers {
		if _, exists := c.Resources.Limits["instrumentation.odigos.io/dotnet"]; exists {
			return true
		}
	}
	return false
}
