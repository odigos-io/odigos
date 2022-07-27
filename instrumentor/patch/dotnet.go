package patch

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/api/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	v1 "k8s.io/api/core/v1"
)

const (
	dotnetAgentName       = "edenfed/otel-dotnet-agent:v0.1"
	enableProfilingEnvVar = "CORECLR_ENABLE_PROFILING"
	profilerEndVar        = "CORECLR_PROFILER"
	profilerId            = "{918728DD-259F-4A6A-AC2B-B85E1B658318}"
	profilerPathEnv       = "CORECLR_PROFILER_PATH"
	profilerPath          = "/agent/OpenTelemetry.AutoInstrumentation.ClrProfiler.Native.so"
	intergationEnv        = "OTEL_INTEGRATIONS"
	intergations          = "/agent/integrations.json"
	conventionsEnv        = "OTEL_CONVENTION"
	serviceNameEnv        = "OTEL_SERVICE"
	convetions            = "OpenTelemetry"
	collectorUrlEnv       = "OTEL_TRACE_AGENT_URL"
	tracerHomeEnv         = "OTEL_DOTNET_TRACER_HOME"
	exportTypeEnv         = "OTEL_EXPORTER"
	tracerHome            = "/agent"
	dotnetVolumeName      = "agentdir-dotnet"
)

var dotNet = &dotNetPatcher{}

type dotNetPatcher struct{}

func (d *dotNetPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) {
	podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, v1.Volume{
		Name: dotnetVolumeName,
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{},
		},
	})

	podSpec.Spec.InitContainers = append(podSpec.Spec.InitContainers, v1.Container{
		Name:  "copy-dotnet-agent",
		Image: dotnetAgentName,
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      dotnetVolumeName,
				MountPath: tracerHome,
			},
		},
	})

	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		if shouldPatch(instrumentation, common.DotNetProgrammingLanguage, container.Name) {
			container.Env = append([]v1.EnvVar{{
				Name: NodeIPEnvName,
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "status.hostIP",
					},
				},
			}}, container.Env...)

			container.Env = append(container.Env, v1.EnvVar{
				Name:  enableProfilingEnvVar,
				Value: "1",
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  profilerEndVar,
				Value: profilerId,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  profilerPathEnv,
				Value: profilerPath,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  intergationEnv,
				Value: intergations,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  tracerHomeEnv,
				Value: tracerHome,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  conventionsEnv,
				Value: convetions,
			})

			// Currently .NET instrumentation only support zipkin format, we should move to OTLP when support is added
			container.Env = append(container.Env, v1.EnvVar{
				Name:  collectorUrlEnv,
				Value: fmt.Sprintf("http://%s:9411/api/v2/spans", HostIPEnvValue),
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  serviceNameEnv,
				Value: calculateAppName(podSpec, &container, instrumentation),
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  exportTypeEnv,
				Value: "Zipkin",
			})

			container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
				MountPath: tracerHome,
				Name:      dotnetVolumeName,
			})
		}

		modifiedContainers = append(modifiedContainers, container)
	}

	podSpec.Spec.Containers = modifiedContainers
}

func (d *dotNetPatcher) IsInstrumented(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) bool {
	// TODO: Deep comparison
	for _, c := range podSpec.Spec.InitContainers {
		if c.Name == "copy-dotnet-agent" {
			return true
		}
	}
	return false
}
