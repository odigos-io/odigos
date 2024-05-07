package instrumentlang

import (
	"fmt"

	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/consts"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	enableProfilingEnvVar = "CORECLR_ENABLE_PROFILING"
	profilerEndVar        = "CORECLR_PROFILER"
	profilerId            = "{918728DD-259F-4A6A-AC2B-B85E1B658318}"
	profilerPathEnv       = "CORECLR_PROFILER_PATH"
	profilerPath          = "/var/odigos/dotnet/OpenTelemetry.AutoInstrumentation.ClrProfiler.Native.so"
	serviceNameEnv        = "OTEL_SERVICE_NAME"
	collectorUrlEnv       = "OTEL_EXPORTER_OTLP_ENDPOINT"
	tracerHomeEnv         = "OTEL_DOTNET_AUTO_HOME"
	exportTypeEnv         = "OTEL_TRACES_EXPORTER"
	tracerHome            = "/var/odigos/dotnet"
	resourceAttrEnv       = "OTEL_RESOURCE_ATTRIBUTES"
	startupHookEnv        = "DOTNET_STARTUP_HOOKS"
	startupHook           = "/var/odigos/dotnet/net/OpenTelemetry.AutoInstrumentation.StartupHook.dll"
	additonalDepsEnv      = "DOTNET_ADDITIONAL_DEPS"
	additonalDeps         = "/var/odigos/dotnet/AdditionalDeps"
	sharedStoreEnv        = "DOTNET_SHARED_STORE"
	sharedStore           = "/var/odigos/dotnet/store"
)

func DotNet(deviceId string) *v1beta1.ContainerAllocateResponse {
	return &v1beta1.ContainerAllocateResponse{
		Envs: map[string]string{
			enableProfilingEnvVar: "1",
			profilerEndVar:        profilerId,
			profilerPathEnv:       profilerPath,
			tracerHomeEnv:         tracerHome,
			collectorUrlEnv:       fmt.Sprintf("http://%s:%d", env.Current.NodeIP, consts.OTLPHttpPort),
			serviceNameEnv:        deviceId,
			exportTypeEnv:         "otlp",
			resourceAttrEnv:       "odigos.device=dotnet",
			startupHookEnv:        startupHook,
			additonalDepsEnv:      additonalDeps,
			sharedStoreEnv:        sharedStore,
		},
		Mounts: []*v1beta1.Mount{
			{
				ContainerPath: "/var/odigos/dotnet",
				HostPath:      "/var/odigos/dotnet",
				ReadOnly:      true,
			},
		},
	}
}
