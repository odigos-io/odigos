package patch

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/api/v1"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/common/consts"
	"github.com/keyval-dev/odigos/common/utils"
	v1 "k8s.io/api/core/v1"
)

const (
	pythonAgentName         = "edenfed/otel-python-agent:v0.1"
	pythonVolumeName        = "agentdir-python"
	pythonMountPath         = "/agent"
	pythonInitContainerName = "copy-python-agent"
)

var python = &pythonPatcher{}

type pythonPatcher struct{}

func (p *pythonPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) {
	podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, v1.Volume{
		Name: pythonVolumeName,
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{},
		},
	})

	podSpec.Spec.InitContainers = append(podSpec.Spec.InitContainers, v1.Container{
		Name:  pythonInitContainerName,
		Image: pythonAgentName,
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      pythonVolumeName,
				MountPath: pythonMountPath,
			},
		},
	})

	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		if shouldPatch(instrumentation, common.PythonProgrammingLanguage, container.Name) {
			container.Env = append(container.Env, v1.EnvVar{
				Name:  "PYTHONPATH",
				Value: "/agent/deps:/agent/deps/opentelemetry/instrumentation/auto_instrumentation/",
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
				Value: fmt.Sprintf("%s.%s:%d", instrumentation.Spec.CollectorAddr, utils.GetCurrentNamespace(), consts.OTLPPort),
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  "OTEL_EXPORTER_OTLP_INSECURE",
				Value: "True",
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  "OTEL_RESOURCE_ATTRIBUTES",
				Value: fmt.Sprintf("service.name=%s", calculateAppName(podSpec, &container, instrumentation)),
			})

			container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
				MountPath: pythonMountPath,
				Name:      pythonVolumeName,
			})
		}
		modifiedContainers = append(modifiedContainers, container)
	}

	podSpec.Spec.Containers = modifiedContainers
}

func (p *pythonPatcher) IsInstrumented(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) bool {
	// TODO: Deep comparison
	for _, c := range podSpec.Spec.InitContainers {
		if c.Name == pythonInitContainerName {
			return true
		}
	}
	return false
}
