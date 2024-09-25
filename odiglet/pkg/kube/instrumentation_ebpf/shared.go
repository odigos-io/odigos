package instrumentation_ebpf

import (
	"context"
	"errors"

	"github.com/odigos-io/odigos/common"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/container"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/process"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func cleanupEbpf(directors ebpf.DirectorsMap, name types.NamespacedName) {
	// cleanup using all available directors
	// the Cleanup method is idempotent, so no harm in calling it multiple times
	for _, director := range directors {
		director.Cleanup(name)
	}
}

func instrumentPodWithEbpf(ctx context.Context, pod *corev1.Pod, directors ebpf.DirectorsMap, runtimeDetails *odigosv1.InstrumentedApplication, podWorkload *workload.PodWorkload) (error, bool) {
	logger := log.FromContext(ctx)
	podUid := string(pod.UID)
	instrumentedEbpf := false
	ignoredContainers := getIgnoredContainers(runtimeDetails)

	for _, container := range pod.Spec.Containers {
		// Ignored containers should not have a device but double check just in case
		if _, ignored := ignoredContainers[container.Name]; ignored {
			continue
		}

		language, sdk, found := odgiosK8s.GetLanguageAndOtelSdk(container)
		if !found {
			continue
		}

		director := directors[ebpf.DirectorKey{Language: language, OtelSdk: sdk}]
		if director == nil {
			continue
		}

		details, err := process.FindAllInContainer(podUid, container.Name)
		if err != nil {
			logger.Error(err, "error finding processes")
			return err, instrumentedEbpf
		}

		var errs []error
		for _, d := range details {
			podDetails := types.NamespacedName{
				Namespace: pod.Namespace,
				Name:      pod.Name,
			}

			// Hack - Nginx instrumentation needs to be on Master Pid which is the min Pid in the container
			if !director.ShouldInstrument(d.ProcessID, details) {
				continue
			}

			err = director.Instrument(ctx, d.ProcessID, podDetails, podWorkload, podWorkload.Name, container.Name)

			if err != nil {
				logger.Error(err, "error initiating process instrumentation", "pid", d.ProcessID)
				errs = append(errs, err)
				continue
			}
			instrumentedEbpf = true
		}

		// Failed to instrument all processes in the container
		if len(errs) > 0 && len(errs) == len(details) {
			return errors.Join(errs...), instrumentedEbpf
		}
	}
	return nil, instrumentedEbpf
}

func getIgnoredContainers(instApp *odigosv1.InstrumentedApplication) map[string]struct{} {
	ignoredContainers := make(map[string]struct{})
	for _, container := range instApp.Spec.RuntimeDetails {
		if container.Language == common.IgnoredProgrammingLanguage {
			ignoredContainers[container.ContainerName] = struct{}{}
		}
	}
	return ignoredContainers
}
