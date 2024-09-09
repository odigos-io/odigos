package instrumentation_ebpf

import (
	"context"
	"errors"
	"fmt"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/container"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	kubeutils "github.com/odigos-io/odigos/odiglet/pkg/kube/utils"
	"github.com/odigos-io/odigos/odiglet/pkg/process"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors"
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

	for _, container := range pod.Spec.Containers {
		language, sdk, found := odgiosK8s.GetLanguageAndOtelSdk(container)

		if !found {
			continue
		}

		director := directors[ebpf.DirectorKey{Language: language, OtelSdk: sdk}]
		if director == nil {
			continue
		}

		// if we instrument multiple containers in the same pod,
		// we want to give each one a unique service.name attribute to differentiate them
		containerName := container.Name
		serviceName := containerName
		if len(runtimeDetails.Spec.RuntimeDetails) == 1 {
			serviceName = podWorkload.Name
		}
		fmt.Printf("@@@@ Instrumenting pod %v:%s container %s with service name %s\n", pod.UID, pod.Name, containerName, serviceName)
		if podWorkload.Name == "postgres" {
			// sleep for 10 seconds to allow the postgres container to start
			fmt.Printf("@@@@ Sleeping for 10 seconds to allow the postgres container to start\n")
			time.Sleep(10 * time.Second)
		}
		// test
		fmt.Printf("@@@@ FindAllInContainer:%v, pod: %v, container: %v\n", pod.UID, pod.Name, container.Name)
		processes, err := process.FindAllInContainer(podUid, containerName)
		if err != nil {
			logger.Error(err, "error finding processes")
			return nil, instrumentedEbpf
		}
		programLanguageDetails := common.ProgramLanguageDetails{Language: common.UnknownProgrammingLanguage}
		var detectErr error
		i := 0
		for _, proc := range processes {
			i++
			containerURL := kubeutils.GetPodExternalURL(pod.Status.PodIP, container.Ports)
			programLanguageDetails, detectErr = inspectors.DetectLanguage(proc, containerURL)
			if detectErr == nil && programLanguageDetails.Language != common.UnknownProgrammingLanguage {
				fmt.Printf("@@@@ [%d] DetectLanguage:%v, Proc: %v pod: %v, container: %v\n", i, programLanguageDetails.Language, proc, pod.Name, container.Name)
			}
		}
		// test end
		details, err := process.FindAllInContainer(podUid, containerName)
		if err != nil {
			logger.Error(err, "error finding processes")
			return err, instrumentedEbpf
		}
		// print the details
		for _, d := range details {
			fmt.Printf("@@@@ $$$ Found process %v in container %s\n", d.ProcessID, containerName)
		}
		var errs []error
		for _, d := range details {
			podDetails := types.NamespacedName{
				Namespace: pod.Namespace,
				Name:      pod.Name,
			}
			err = director.Instrument(ctx, d.ProcessID, podDetails, podWorkload, serviceName, containerName)

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
