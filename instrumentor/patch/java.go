package patch

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/instrumentor/api/v1"
	"github.com/keyval-dev/odigos/instrumentor/consts"
	"github.com/keyval-dev/odigos/instrumentor/utils"

	v1 "k8s.io/api/core/v1"
)

const (
	javaAgentImage               = "keyval/otel-java-agent:v0.3"
	javaVolumeName               = "agentdir-java"
	javaMountPath                = "/agent"
	otelResourceAttributesEnvVar = "OTEL_RESOURCE_ATTRIBUTES"
	otelResourceAttrPatteern     = "service.name=%s"
	javaToolOptionsEnvVar        = "JAVA_OPTS"
	javaToolOptionsPattern       = "-javaagent:/agent/opentelemetry-javaagent-all.jar " +
		"-Dotel.metrics.exporter=none -Dotel.traces.sampler=always_on -Dotel.exporter.otlp.endpoint=http://%s.%s:%d"
)

var java = &javaPatcher{}

type javaPatcher struct{}

func (j *javaPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) {
	podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, v1.Volume{
		Name: javaVolumeName,
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{},
		},
	})

	podSpec.Spec.InitContainers = append(podSpec.Spec.InitContainers, v1.Container{
		Name:    "copy-java-agent",
		Image:   javaAgentImage,
		Command: []string{"cp", "/javaagent.jar", "/agent/opentelemetry-javaagent-all.jar"},
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      javaVolumeName,
				MountPath: javaMountPath,
			},
		},
	})

	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		if shouldPatch(instrumentation, odigosv1.JavaProgrammingLanguage, container.Name) {
			idx := getIndexOfEnv(container.Env, javaToolOptionsEnvVar)
			if idx == -1 {
				container.Env = append(container.Env, v1.EnvVar{
					Name:  javaToolOptionsEnvVar,
					Value: fmt.Sprintf(javaToolOptionsPattern, instrumentation.Spec.CollectorAddr, utils.GetCurrentNamespace(), consts.OTLPPort),
				})
			} else {
				container.Env[idx].Value = container.Env[idx].Value + " " + fmt.Sprintf(javaToolOptionsPattern, instrumentation.Spec.CollectorAddr,
					utils.GetCurrentNamespace(), consts.OTLPPort)
			}

			container.Env = append(container.Env, v1.EnvVar{
				Name:  otelResourceAttributesEnvVar,
				Value: fmt.Sprintf(otelResourceAttrPatteern, calculateAppName(podSpec, &container, instrumentation)),
			})
			container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
				MountPath: javaMountPath,
				Name:      javaVolumeName,
			})
		}

		modifiedContainers = append(modifiedContainers, container)
	}

	podSpec.Spec.Containers = modifiedContainers
}
