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
	serviceNameEnv        = "OTEL_SERVICE_NAME"
	collectorUrlEnv       = "OTEL_EXPORTER_OTLP_ENDPOINT"
	tracerHomeEnv         = "OTEL_DOTNET_AUTO_HOME"
	exportTypeEnv         = "OTEL_TRACES_EXPORTER"
	tracerHome            = "/odigos/dotnet"
	resourceAttrEnv       = "OTEL_RESOURCE_ATTRIBUTES"
	startupHookEnv        = "DOTNET_STARTUP_HOOKS"
	startupHook           = "/odigos/dotnet/OpenTelemetry.AutoInstrumentation.StartupHook.dll"
	additonalDepsEnv      = "DOTNET_ADDITIONAL_DEPS"
	additonalDeps         = "/odigos/dotnet/AdditionalDeps"
	sharedStoreEnv        = "DOTNET_SHARED_STORE"
	sharedStore           = "/odigos/dotnet/store"
)

func DotNet(deviceId string) *v1beta1.ContainerAllocateResponse {
	return &v1beta1.ContainerAllocateResponse{
		Envs: map[string]string{
			enableProfilingEnvVar: "1",
			profilerEndVar:        profilerId,
			profilerPathEnv:       profilerPath,
			tracerHomeEnv:         tracerHome,
			collectorUrlEnv:       fmt.Sprintf("http://%s:9411/api/v2/spans", env.Current.NodeIP),
			serviceNameEnv:        deviceId,
			exportTypeEnv:         "otlp",
			resourceAttrEnv:       "odigos.device=dotnet",
			startupHookEnv:        startupHook,
			additonalDepsEnv:      additonalDeps,
			sharedStoreEnv:        sharedStore,
		},
		Mounts: []*v1beta1.Mount{
			{
				ContainerPath: "/odigos/dotnet",
				HostPath:      "/odigos/dotnet",
			},
		},
	}
}
