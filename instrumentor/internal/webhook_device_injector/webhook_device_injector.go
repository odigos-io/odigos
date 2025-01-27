package webhookdeviceinjector

import (
	"context"

	"github.com/go-logr/logr"
	v1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func InjectOdigosInstrumentationDevice(ctx context.Context, p client.Client, logger logr.Logger, podWorkload workload.PodWorkload, container *corev1.Container, pl common.ProgrammingLanguage, otelSdk common.OtelSdk) {

	libcType := getLibCTypeOfContainer(ctx, p, logger, podWorkload, container.Name)
	instrumentationDeviceName := common.InstrumentationDeviceName(pl, otelSdk, libcType)

	if container.Resources.Limits == nil {
		container.Resources.Limits = make(corev1.ResourceList)
	}
	if container.Resources.Requests == nil {
		container.Resources.Requests = make(corev1.ResourceList)
	}

	container.Resources.Limits[corev1.ResourceName(instrumentationDeviceName)] = resource.MustParse("1")
	container.Resources.Requests[corev1.ResourceName(instrumentationDeviceName)] = resource.MustParse("1")

}

func getLibCTypeOfContainer(ctx context.Context, p client.Client, logger logr.Logger, podWorkload workload.PodWorkload, containerName string) *common.LibCType {

	var instConfig v1alpha1.InstrumentationConfig
	err := p.Get(ctx, client.ObjectKey{Namespace: podWorkload.Namespace, Name: workload.CalculateWorkloadRuntimeObjectName(podWorkload.Name, podWorkload.Kind)}, &instConfig)
	if err != nil {
		logger.Error(err, "Failed to get instrumentationConfig", "podWorkload", podWorkload)
		return nil
	}

	// Find the matching container runtime details
	for _, rd := range instConfig.Status.RuntimeDetailsByContainer {
		if rd.ContainerName == containerName {
			return rd.LibCType
		}
	}

	return nil
}
