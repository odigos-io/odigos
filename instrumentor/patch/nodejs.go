package patch

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/api/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/common/consts"
	v1 "k8s.io/api/core/v1"
)

const (
	nodeAgentImage       = "keyval/otel-nodejs-agent:v0.2"
	nodeVolumeName       = "agentdir-nodejs"
	nodeMountPath        = "/agent-nodejs"
	nodeEnvNodeDebug     = "OTEL_NODEJS_DEBUG"
	nodeEnvTraceExporter = "OTEL_TRACES_EXPORTER"
	nodeEnvEndpoint      = "OTEL_EXPORTER_OTLP_ENDPOINT"
	nodeEnvServiceName   = "OTEL_SERVICE_NAME"
	nodeEnvNodeOptions   = "NODE_OPTIONS"
)

var nodeJs = &nodeJsPatcher{}

type nodeJsPatcher struct{}

func (n *nodeJsPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) {
	podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, v1.Volume{
		Name: nodeVolumeName,
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{},
		},
	})

	podSpec.Spec.InitContainers = append(podSpec.Spec.InitContainers, v1.Container{
		Name:    "copy-nodejs-agent",
		Image:   nodeAgentImage,
		Command: []string{"cp", "-a", "/autoinstrumentation/.", fmt.Sprintf("/%s/", nodeMountPath)},
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      nodeVolumeName,
				MountPath: nodeMountPath,
			},
		},
	})

	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		if shouldPatch(instrumentation, common.JavascriptProgrammingLanguage, container.Name) {
			container.Env = append([]v1.EnvVar{{
				Name: NodeIPEnvName,
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "status.hostIP",
					},
				},
			}}, container.Env...)

			container.Env = append(container.Env, v1.EnvVar{
				Name:  nodeEnvNodeDebug,
				Value: "true",
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  nodeEnvTraceExporter,
				Value: "otlp",
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  nodeEnvEndpoint,
				Value: fmt.Sprintf("%s:%d", HostIPEnvValue, consts.OTLPPort),
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  nodeEnvServiceName,
				Value: calculateAppName(podSpec, &container, instrumentation),
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  nodeEnvNodeOptions,
				Value: fmt.Sprintf("--require /%s/autoinstrumentation.js", nodeMountPath),
			})

			container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
				MountPath: nodeMountPath,
				Name:      nodeVolumeName,
			})
		}
		modifiedContainers = append(modifiedContainers, container)
	}
	podSpec.Spec.Containers = modifiedContainers
}

func (n *nodeJsPatcher) IsInstrumented(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) bool {
	// TODO: Deep comparison
	for _, c := range podSpec.Spec.InitContainers {
		if c.Name == "copy-nodejs-agent" {
			return true
		}
	}
	return false
}
