package patch

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/instrumentor/api/v1"
	"github.com/keyval-dev/odigos/instrumentor/utils"

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
		if shouldPatch(instrumentation, odigosv1.DotNetProgrammingLanguage, container.Name) {
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
				Value: fmt.Sprintf("http://%s.%s:9411/api/v2/spans", instrumentation.Spec.CollectorAddr, utils.GetCurrentNamespace()),
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
