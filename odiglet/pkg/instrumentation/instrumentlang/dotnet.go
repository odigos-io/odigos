package instrumentlang

import (
	"fmt"
	"github.com/keyval-dev/odigos/odiglet/pkg/env"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	enableProfilingEnvVar = "CORECLR_ENABLE_PROFILING"
	profilerEndVar        = "CORECLR_PROFILER"
	profilerId            = "{918728DD-259F-4A6A-AC2B-B85E1B658318}"
	profilerPathEnv       = "CORECLR_PROFILER_PATH"
	profilerPath          = "/odigos/dotnet/OpenTelemetry.AutoInstrumentation.ClrProfiler.Native.so"
	intergationEnv        = "OTEL_INTEGRATIONS"
	intergations          = "/odigos/dotnet/integrations.json"
	conventionsEnv        = "OTEL_CONVENTION"
	serviceNameEnv        = "OTEL_SERVICE"
	convetions            = "OpenTelemetry"
	collectorUrlEnv       = "OTEL_TRACE_AGENT_URL"
	tracerHomeEnv         = "OTEL_DOTNET_TRACER_HOME"
	exportTypeEnv         = "OTEL_EXPORTER"
	tracerHome            = "/odigos/dotnet"
)

func DotNet(deviceId string) *v1beta1.ContainerAllocateResponse {
	return &v1beta1.ContainerAllocateResponse{
		Envs: map[string]string{
			enableProfilingEnvVar: "1",
			profilerEndVar:        profilerId,
			profilerPathEnv:       profilerPath,
			intergationEnv:        intergations,
			tracerHomeEnv:         tracerHome,
			conventionsEnv:        convetions,
			collectorUrlEnv:       fmt.Sprintf("http://%s:9411/api/v2/spans", env.Current.NodeIP),
			serviceNameEnv:        deviceId,
			exportTypeEnv:         "Zipkin",
			//resourceAttrEnv:       "odigos.device=dotnet",
		},
		Mounts: []*v1beta1.Mount{
			{
				ContainerPath: "/odigos/dotnet",
				HostPath:      "/odigos/dotnet",
			},
		},
	}
}
