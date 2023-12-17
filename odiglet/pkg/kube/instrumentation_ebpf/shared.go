package instrumentation_ebpf

import (
	"context"
	"errors"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/common/utils"
	"github.com/keyval-dev/odigos/odiglet/pkg/ebpf"
	"github.com/keyval-dev/odigos/odiglet/pkg/process"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func cleanupEbpf(directors map[common.ProgrammingLanguage]ebpf.Director, name types.NamespacedName) {
	// cleanup using all available directors
	// the Cleanup method is idempotent, so no harm in calling it multiple times
	for _, director := range directors {
		director.Cleanup(name)
	}
}

func instrumentPodWithEbpf(ctx context.Context, pod *corev1.Pod, directors map[common.ProgrammingLanguage]ebpf.Director, runtimeDetails *odigosv1.InstrumentedApplication, podWorkload *common.PodWorkload) error {
	logger := log.FromContext(ctx)
	podUid := string(pod.UID)
	for _, container := range runtimeDetails.Spec.Languages {

		director := directors[container.Language]
		if director == nil {
			return errors.New("no director found for language " + string(container.Language))
		}

		appName := container.ContainerName
		if len(runtimeDetails.Spec.Languages) == 1 {
			appName = runtimeDetails.OwnerReferences[0].Name
		}

		details, err := process.FindAllInContainer(podUid, container.ContainerName)
		if err != nil {
			logger.Error(err, "error finding processes")
			return err
		}

		for _, d := range details {
			podDetails := types.NamespacedName{
				Namespace: pod.Namespace,
				Name:      pod.Name,
			}
			err = director.Instrument(ctx, d.ProcessID, podDetails, podWorkload, appName)

			if err != nil {
				logger.Error(err, "error initiating process instrumentation", "pid", d.ProcessID)
				return err
			}
		}
	}
	return nil
}

// TODO: do it for container in pod
func isPodEbpfInstrumented(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if container.Resources.Limits == nil {
			continue
		}

		for resourceName, _ := range container.Resources.Limits {
			if strings.HasPrefix(string(resourceName), common.OdigosResourceNamespace) &&
				strings.Contains(string(resourceName), "ebpf") {
				return true
			}
		}
	}

	return false
}

func getRuntimeDetails(ctx context.Context, kubeClient client.Client, podWorkload *common.PodWorkload) (*odigosv1.InstrumentedApplication, error) {
	instrumentedApplicationName := utils.GetRuntimeObjectName(podWorkload.Name, podWorkload.Kind)

	var runtimeDetails odigosv1.InstrumentedApplication
	err := kubeClient.Get(ctx, client.ObjectKey{
		Namespace: podWorkload.Namespace,
		Name:      instrumentedApplicationName,
	}, &runtimeDetails)
	if err != nil {
		return nil, err
	}

	return &runtimeDetails, nil
}
