package ebpf

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentation"
	instance "github.com/odigos-io/odigos/k8sutils/pkg/instrumentation_instance"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type k8sReporter struct {
	client client.Client
}

var _ instrumentation.Reporter[K8sProcessGroup, K8sConfigGroup, *K8sProcessDetails] = &k8sReporter{}

type errRequiredEnvVarNotFound struct {
	envVarName string
}

func (e *errRequiredEnvVarNotFound) Error() string {
	return fmt.Sprintf("required environment variable not found: %s", e.envVarName)
}

var _ error = &errRequiredEnvVarNotFound{}

var (
	errContainerNameNotReported = &errRequiredEnvVarNotFound{envVarName: k8sconsts.OdigosEnvVarContainerName}
	errPodNameNotReported       = &errRequiredEnvVarNotFound{envVarName: k8sconsts.OdigosEnvVarPodName}
	errPodNameSpaceNotReported  = &errRequiredEnvVarNotFound{envVarName: k8sconsts.OdigosEnvVarNamespace}
)

type InstrumentationStatusReason string

const (
	FailedToLoad       InstrumentationStatusReason = "FailedToLoad"
	FailedToInitialize InstrumentationStatusReason = "FailedToInitialize"
	LoadedSuccessfully InstrumentationStatusReason = "LoadedSuccessfully"
	FailedToRun        InstrumentationStatusReason = "FailedToRun"
)

type InstrumentationHealth bool

const (
	InstrumentationHealthy   InstrumentationHealth = true
	InstrumentationUnhealthy InstrumentationHealth = false
)

func (r *k8sReporter) OnInit(ctx context.Context, pid int, err error, e *K8sProcessDetails) error {
	if err == nil {
		// currently we don't report on successful initialization
		return nil
	}

	return r.updateInstrumentationInstanceStatus(ctx, e, pid, InstrumentationUnhealthy, FailedToInitialize, err.Error(), instrumentation.Status{})
}

func (r *k8sReporter) OnLoad(ctx context.Context, pid int, err error, e *K8sProcessDetails, status instrumentation.Status) error {
	if err != nil {
		return r.updateInstrumentationInstanceStatus(ctx, e, pid, InstrumentationUnhealthy, FailedToLoad, err.Error(), status)
	}

	msg := fmt.Sprintf("Successfully loaded eBPF probes to pod: %s container: %s", e.Pod.Name, e.ContainerName)
	return r.updateInstrumentationInstanceStatus(ctx, e, pid, InstrumentationHealthy, LoadedSuccessfully, msg, status)
}

func (r *k8sReporter) OnRun(ctx context.Context, pid int, err error, e *K8sProcessDetails) error {
	if err == nil {
		// finished running successfully
		return nil
	}

	return r.updateInstrumentationInstanceStatus(ctx, e, pid, InstrumentationUnhealthy, FailedToRun, err.Error(), instrumentation.Status{})
}

func (r *k8sReporter) OnExit(ctx context.Context, pid int, e *K8sProcessDetails) error {
	if err := r.client.Delete(ctx, &odigosv1.InstrumentationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.InstrumentationInstanceName(e.Pod.Name, pid),
			Namespace: e.Pod.Namespace,
		},
	}); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("error deleting instrumentation instance for pod %s pid %d: %w", e.Pod.Name, pid, err)
	}
	return nil
}

func (r *k8sReporter) updateInstrumentationInstanceStatus(ctx context.Context, ke *K8sProcessDetails, pid int, health InstrumentationHealth, reason InstrumentationStatusReason, msg string, status instrumentation.Status) error {
	instrumentedAppName := workload.CalculateWorkloadRuntimeObjectName(ke.Pw.Name, ke.Pw.Kind)
	healthy := bool(health)
	components := make([]odigosv1.InstrumentationLibraryStatus, 0, len(status.Components))
	for name, componentErr := range status.Components {
		componentHealthy := componentErr == nil
		componentStatus := odigosv1.InstrumentationLibraryStatus{
			Name:           name,
			Type:           odigosv1.InstrumentationLibraryTypeInstrumentation,
			LastStatusTime: metav1.Now(),
			Healthy:        &componentHealthy,
		}
		if componentErr != nil {
			componentStatus.Message = componentErr.Error()
		}
		components = append(components, componentStatus)
	}
	return instance.UpdateInstrumentationInstanceStatus(ctx, ke.Pod, ke.ContainerName, r.client, instrumentedAppName, pid, r.client.Scheme(),
		instance.WithHealthy(&healthy, string(reason), &msg),
		instance.WithComponents(components),
	)
}
