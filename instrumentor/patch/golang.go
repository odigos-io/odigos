package patch

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/instrumentor/api/v1"
	"github.com/keyval-dev/odigos/instrumentor/consts"
	"github.com/keyval-dev/odigos/instrumentor/utils"

	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	golangKernelDebugVolumeName = "kernel-debug"
	golangKernelDebugHostPath   = "/sys/kernel/debug"
	golangAgentName             = "keyval/otel-go-agent:v0.5.3"
	golangExporterEndpoint      = "OTEL_EXPORTER_OTLP_ENDPOINT"
	golangServiceNameEnv        = "OTEL_SERVICE_NAME"
	golangTargetExeEnv          = "OTEL_TARGET_EXE"
)

var golang = &golangPatcher{}

type golangPatcher struct{}

func (g *golangPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) {
	modifiedContainers := podSpec.Spec.Containers
	patchedAnyContainer := false

	for _, container := range podSpec.Spec.Containers {
		if shouldPatch(instrumentation, odigosv1.GoProgrammingLanguage, container.Name) {
			exePath := g.getExe(container.Name, instrumentation)
			if exePath == "" {
				ctrl.Log.V(0).Info("could not find binary path for golang application",
					"container", container.Name)
				continue
			}

			patchedAnyContainer = true
			bpfContainer := v1.Container{
				Name:  fmt.Sprintf("%s-instrumentation", container.Name),
				Image: golangAgentName,
				Env: []v1.EnvVar{
					{
						Name:  golangExporterEndpoint,
						Value: fmt.Sprintf("%s.%s:%d", instrumentation.Spec.CollectorAddr, utils.GetCurrentNamespace(), consts.OTLPPort),
					},
					{
						Name:  golangServiceNameEnv,
						Value: calculateAppName(podSpec, &container, instrumentation),
					},
					{
						Name:  golangTargetExeEnv,
						Value: exePath,
					},
				},
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      golangKernelDebugVolumeName,
						MountPath: golangKernelDebugHostPath,
					},
				},
				SecurityContext: &v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{
							"SYS_PTRACE",
						},
					},
					Privileged: boolPtr(true),
					RunAsUser:  intPtr(0),
				},
			}

			modifiedContainers = append(modifiedContainers, bpfContainer)
		}
	}

	if !patchedAnyContainer {
		return
	}

	podSpec.Spec.Containers = modifiedContainers
	// TODO: if explicitly set to false, fallback to hostPID
	podSpec.Spec.ShareProcessNamespace = boolPtr(true)

	podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, v1.Volume{
		Name: golangKernelDebugVolumeName,
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: golangKernelDebugHostPath,
			},
		},
	})
}

func (g *golangPatcher) getExe(containerName string, instrumentation *odigosv1.InstrumentedApplication) string {
	for _, l := range instrumentation.Spec.Languages {
		if l.ContainerName == containerName && l.Language == odigosv1.GoProgrammingLanguage {
			return l.ProcessName
		}
	}
	return ""
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(n int64) *int64 {
	return &n
}
