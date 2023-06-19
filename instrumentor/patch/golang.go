package patch

import (
	"fmt"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/common/consts"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	golangDeviceName            = "instrumentation.odigos.io/go"
	golangKernelDebugVolumeName = "kernel-debug"
	golangKernelDebugHostPath   = "/sys/kernel/debug"
	golangExporterEndpoint      = "OTEL_EXPORTER_OTLP_ENDPOINT"
	golangServiceNameEnv        = "OTEL_SERVICE_NAME"
	golangTargetExeEnv          = "OTEL_GO_AUTO_TARGET_EXE"
)

var GolangSidecarInstrumentation bool

var golang = &golangPatcher{}

type golangPatcher struct{}

func (g *golangPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) {
	if GolangSidecarInstrumentation {
		g.patchWithSidecar(podSpec, instrumentation)
		return
	}

	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		if shouldPatch(instrumentation, common.GoProgrammingLanguage, container.Name) {
			if container.Resources.Limits == nil {
				container.Resources.Limits = make(map[v1.ResourceName]resource.Quantity)
			}

			container.Resources.Limits[golangDeviceName] = resource.MustParse("1")
		}

		modifiedContainers = append(modifiedContainers, container)
	}

	podSpec.Spec.Containers = modifiedContainers
}

func (g *golangPatcher) Revert(podSpec *v1.PodTemplateSpec) {
	if GolangSidecarInstrumentation {
		g.revertWithSidecar(podSpec)
		return
	}

	removeDeviceFromPodSpec(golangDeviceName, podSpec)
}

func (g *golangPatcher) patchWithSidecar(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) {
	modifiedContainers := podSpec.Spec.Containers

	for _, l := range instrumentation.Spec.Languages {
		if shouldPatch(instrumentation, common.GoProgrammingLanguage, l.ContainerName) {
			if l.ProcessName == "" {
				ctrl.Log.V(0).Info("could not find binary path for golang application",
					"container", l.ContainerName)
				continue
			}

			appName := l.ContainerName
			if len(instrumentation.Spec.Languages) == 1 && len(instrumentation.OwnerReferences) > 0 {
				appName = instrumentation.OwnerReferences[0].Name
			}

			containerName := fmt.Sprintf("%s-instrumentation", l.ContainerName)
			if g.isContainerExists(podSpec, containerName) {
				continue
			}

			bpfContainer := v1.Container{
				Name:  containerName,
				Image: consts.GolangInstrumentationImage,
				Env: []v1.EnvVar{
					{
						Name: NodeIPEnvName,
						ValueFrom: &v1.EnvVarSource{
							FieldRef: &v1.ObjectFieldSelector{
								FieldPath: "status.hostIP",
							},
						},
					},
					{
						Name:  golangExporterEndpoint,
						Value: fmt.Sprintf("http://%s:%d", HostIPEnvValue, consts.OTLPPort),
					},
					{
						Name:  golangServiceNameEnv,
						Value: appName,
					},
					{
						Name:  golangTargetExeEnv,
						Value: l.ProcessName,
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

	podSpec.Spec.Containers = modifiedContainers
	podSpec.Spec.ShareProcessNamespace = boolPtr(true)

	if !g.isVolumeExists(podSpec, golangKernelDebugVolumeName) {
		podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, v1.Volume{
			Name: golangKernelDebugVolumeName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: golangKernelDebugHostPath,
				},
			},
		})
	}
}

func (g *golangPatcher) revertWithSidecar(podSpec *v1.PodTemplateSpec) {
	for i, c := range podSpec.Spec.Containers {
		if c.Image == consts.GolangInstrumentationImage {
			podSpec.Spec.Containers = append(podSpec.Spec.Containers[:i], podSpec.Spec.Containers[i+1:]...)
			break
		}
	}

	if podSpec.Spec.ShareProcessNamespace != nil && *podSpec.Spec.ShareProcessNamespace {
		podSpec.Spec.ShareProcessNamespace = nil
	}

	for i, v := range podSpec.Spec.Volumes {
		if v.Name == golangKernelDebugVolumeName {
			podSpec.Spec.Volumes = append(podSpec.Spec.Volumes[:i], podSpec.Spec.Volumes[i+1:]...)
			break
		}
	}
}

func (g *golangPatcher) isContainerExists(podSpec *v1.PodTemplateSpec, containerName string) bool {
	for _, c := range podSpec.Spec.Containers {
		if c.Name == containerName {
			return true
		}
	}

	return false
}

func (g *golangPatcher) isVolumeExists(podSpec *v1.PodTemplateSpec, volumeName string) bool {
	for _, v := range podSpec.Spec.Volumes {
		if v.Name == volumeName {
			return true
		}
	}

	return false
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(n int64) *int64 {
	return &n
}
