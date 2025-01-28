package webhookdeviceinjector

import (
	v1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func InjectOdigosInstrumentationDevice(
	podWorkload workload.PodWorkload,
	container *corev1.Container,
	otelSdk common.OtelSdk,
	runtimeDetails *v1alpha1.RuntimeDetailsByContainer,
) error {
	libcType := runtimeDetails.LibCType
	instrumentationDeviceName := common.InstrumentationDeviceName(runtimeDetails.Language, otelSdk, libcType)

	if instrumentationDeviceName == "" {
		return nil
	}

	ensureResourceListsInitialized(container)

	container.Resources.Limits[corev1.ResourceName(instrumentationDeviceName)] = resource.MustParse("1")
	container.Resources.Requests[corev1.ResourceName(instrumentationDeviceName)] = resource.MustParse("1")

	return nil
}

func ensureResourceListsInitialized(container *corev1.Container) {
	if container.Resources.Limits == nil {
		container.Resources.Limits = make(corev1.ResourceList)
	}
	if container.Resources.Requests == nil {
		container.Resources.Requests = make(corev1.ResourceList)
	}
}
