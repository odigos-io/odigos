package ebpf

import (
	"context"
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/instrumentation/detector"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
)

// K8sProcessDetails holds Kubernetes-specific details about a process.
// It can be used to group processes by their associated workloads and to determine
// the appropriate OpenTelemetry distribution for instrumentation.
type K8sProcessDetails struct {
	Pod           *corev1.Pod
	ContainerName string
	Distro        *distro.OtelDistro
	Pw            *k8sconsts.PodWorkload
	ProcEvent     detector.ProcessEvent
}

func (kd K8sProcessDetails) String() string {
	return fmt.Sprintf("Pod: %s.%s, Container: %s, Workload: %s",
		kd.Pod.Name, kd.Pod.Namespace,
		kd.ContainerName,
		workload.CalculateWorkloadRuntimeObjectName(kd.Pw.Name, kd.Pw.Kind),
	)
}

var _ instrumentation.ProcessDetails[K8sProcessGroup, K8sConfigGroup] = &K8sProcessDetails{}

// K8sProcessGroup is the k8s specific ProcessGroup that used to group all the instrumented
// processes of a given user "workload"
type K8sProcessGroup struct {
	Pw k8sconsts.PodWorkload
}

// K8sConfigGroup is the k8s specific ConfigGroup used to group config changes requests.
// Currently the InstrumentationConfig CRD groups the configuration for a given Source in the sdkConfigs field
// in which the configuration is indexed by programming language.
//
// In the InstrumentationConfig CRD we have the Containers slice which groups configuration by containers.
// This is the preferred approach and we should migrate away from the sdkConfigs since grouping by containers allows for more flexibility and cleaner code.
// Once the migration is done - this config group can change internally to replace the language field with a container field.
// For each container, we save its resolved distribution in the InstrumentationConfig - thus we can have access to the language as well.
type K8sConfigGroup struct {
	Pw   k8sconsts.PodWorkload
	Lang common.ProgrammingLanguage
}

func (kd *K8sProcessDetails) ConfigGroup(ctx context.Context) (K8sConfigGroup, error) {
	if kd.Pw == nil {
		return K8sConfigGroup{}, errors.New("podWorkload is not provided, cannot resolve config group")
	}
	if kd.Distro == nil {
		return K8sConfigGroup{}, errors.New("distribution is not provided, cannot resolve config group")
	}
	return K8sConfigGroup{
		Pw:   *kd.Pw,
		Lang: kd.Distro.Language,
	}, nil
}

func (kd *K8sProcessDetails) Distribution(ctx context.Context) (*distro.OtelDistro, error) {
	if kd.Distro == nil {
		return nil, errors.New("distribution is not provided, cannot resolve config group")
	}
	return kd.Distro, nil
}

func (kd *K8sProcessDetails) ProcessGroup(ctx context.Context) (K8sProcessGroup, error) {
	if kd.Pw == nil {
		return K8sProcessGroup{}, errors.New("podWorkload is not provided, cannot resolve config group")
	}
	return K8sProcessGroup{Pw: *kd.Pw}, nil
}
