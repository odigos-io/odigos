package ebpf

import (
	"context"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	instance "github.com/odigos-io/odigos/k8sutils/pkg/instrumentation_instance"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8sProcessDetails struct {
	pod           *corev1.Pod
	containerName string
	pw            *workload.PodWorkload
}

func (kd K8sProcessDetails) String() string {
	return fmt.Sprintf("Pod: %s.%s, Container: %s, Workload: %s",
		kd.pod.Name, kd.pod.Namespace,
		kd.containerName,
		workload.CalculateWorkloadRuntimeObjectName(kd.pw.Name, kd.pw.Kind),
	)
}

var _ instrumentation.ProcessDetails = K8sProcessDetails{}

type k8sReporter struct {
	client client.Client
}

type K8sConfigGroup struct {
	Pw   workload.PodWorkload
	Lang common.ProgrammingLanguage
}

var _ instrumentation.Reporter[K8sProcessDetails] = &k8sReporter{}

type errRequiredEnvVarNotFound struct {
	envVarName string
}

func (e *errRequiredEnvVarNotFound) Error() string {
	return fmt.Sprintf("required environment variable not found: %s", e.envVarName)
}

var _ error = &errRequiredEnvVarNotFound{}

var (
	errContainerNameNotReported = &errRequiredEnvVarNotFound{envVarName: consts.OdigosEnvVarContainerName}
	errPodNameNotReported       = &errRequiredEnvVarNotFound{envVarName: consts.OdigosEnvVarPodName}
	errPodNameSpaceNotReported  = &errRequiredEnvVarNotFound{envVarName: consts.OdigosEnvVarNamespace}
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

func (r *k8sReporter) OnInit(ctx context.Context, pid int, err error, e K8sProcessDetails) error {
	if err == nil {
		// currently we don't report on successful initialization
		return nil
	}

	return r.updateInstrumentationInstanceStatus(ctx, e, pid, InstrumentationUnhealthy, FailedToInitialize, err.Error())
}

func (r *k8sReporter) OnLoad(ctx context.Context, pid int, err error, e K8sProcessDetails) error {
	if err != nil {
		return r.updateInstrumentationInstanceStatus(ctx, e, pid, InstrumentationUnhealthy, FailedToLoad, err.Error())
	}

	msg := fmt.Sprintf("Successfully loaded eBPF probes to pod: %s container: %s", e.pod.Name, e.containerName)
	return r.updateInstrumentationInstanceStatus(ctx, e, pid, InstrumentationHealthy, LoadedSuccessfully, msg)
}

func (r *k8sReporter) OnRun(ctx context.Context, pid int, err error, e K8sProcessDetails) error {
	if err == nil {
		// finished running successfully
		return nil
	}

	return r.updateInstrumentationInstanceStatus(ctx, e, pid, InstrumentationUnhealthy, FailedToRun, err.Error())
}

func (r *k8sReporter) OnExit(ctx context.Context, pid int, e K8sProcessDetails) error {
	if err := r.client.Delete(ctx, &odigosv1.InstrumentationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.InstrumentationInstanceName(e.pod.Name, pid),
			Namespace: e.pod.Namespace,
		},
	}); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("error deleting instrumentation instance for pod %s pid %d: %w", e.pod.Name, pid, err)
	}
	return nil
}

func (r *k8sReporter) updateInstrumentationInstanceStatus(ctx context.Context, ke K8sProcessDetails, pid int, health InstrumentationHealth, reason InstrumentationStatusReason, msg string) error {
	instrumentedAppName := workload.CalculateWorkloadRuntimeObjectName(ke.pw.Name, ke.pw.Kind)
	healthy := bool(health)
	return instance.UpdateInstrumentationInstanceStatus(ctx, ke.pod, ke.containerName, r.client, instrumentedAppName, pid, r.client.Scheme(),
		instance.WithHealthy(&healthy, string(reason), &msg),
	)
}
