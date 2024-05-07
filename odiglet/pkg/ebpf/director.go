package ebpf

import (
	"context"
	"sync"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"k8s.io/apimachinery/pkg/types"
)

// This interface should be implemented by all ebpf sdks
// for example, the go auto instrumentation sdk implements it
type OtelEbpfSdk interface {
	Run(ctx context.Context) error
	Close(ctx context.Context) error
}

// users can use different eBPF otel SDKs by returning them from this function
type InstrumentationFactory[T OtelEbpfSdk] interface {
	CreateEbpfInstrumentation(ctx context.Context, pid int, serviceName string, podWorkload *common.PodWorkload, containerName string, podName string) (T, error)
}

// Director manages the instrumentation for a specific SDK in a specific language
type Director interface {
	Language() common.ProgrammingLanguage
	Instrument(ctx context.Context, pid int, podDetails types.NamespacedName, podWorkload *common.PodWorkload, appName string, containerName string) error
	Cleanup(podDetails types.NamespacedName)
	Shutdown()
}

type podDetails struct {
	Workload *common.PodWorkload
	Pids     []int
}

type EbpfDirector[T OtelEbpfSdk] struct {
	mux sync.Mutex

	language               common.ProgrammingLanguage
	instrumentationFactory InstrumentationFactory[T]

	// this map holds the instrumentation object which is used to close the instrumentation
	// the map is filled only after the instrumentation is actually created
	// which is an asyn process that might take some time
	pidsToInstrumentation map[int]T

	// this map is used to make sure we do not attempt to instrument the same process twice.
	// it keeps track of which processes we already attempted to instrument,
	// so we can avoid attempting to instrument them again.
	pidsAttemptedInstrumentation map[int]struct{}

	// via this map, we can find the workload and pids for a specific pod.
	// sometimes we only have the pod name and namespace, so this map is useful.
	podsToDetails map[types.NamespacedName]podDetails

	// this map can be used when we only have the workload, and need to find the pods to derive pids.
	workloadToPods map[common.PodWorkload]map[types.NamespacedName]struct{}
}

type DirectorKey struct {
	Language common.ProgrammingLanguage
	common.OtelSdk
}

type DirectorsMap map[DirectorKey]Director

func NewEbpfDirector[T OtelEbpfSdk](language common.ProgrammingLanguage, instrumentationFactory InstrumentationFactory[T]) *EbpfDirector[T] {
	return &EbpfDirector[T]{
		language:                     language,
		instrumentationFactory:       instrumentationFactory,
		pidsToInstrumentation:        make(map[int]T),
		pidsAttemptedInstrumentation: make(map[int]struct{}),
		podsToDetails:                make(map[types.NamespacedName]podDetails),
		workloadToPods:               make(map[common.PodWorkload]map[types.NamespacedName]struct{}),
	}
}

func (d *EbpfDirector[T]) Instrument(ctx context.Context, pid int, pod types.NamespacedName, podWorkload *common.PodWorkload, appName string, containerName string) error {
	log.Logger.V(0).Info("Instrumenting process", "pid", pid, "workload", podWorkload)
	d.mux.Lock()
	defer d.mux.Unlock()
	if _, exists := d.pidsAttemptedInstrumentation[pid]; exists {
		log.Logger.V(5).Info("Process already instrumented", "pid", pid)
		return nil
	}

	details, exists := d.podsToDetails[pod]
	if !exists {
		details = podDetails{
			Workload: podWorkload,
			Pids:     []int{},
		}
		d.podsToDetails[pod] = details
	}
	details.Pids = append(details.Pids, pid)
	d.podsToDetails[pod] = details

	d.pidsAttemptedInstrumentation[pid] = struct{}{}

	if _, exists := d.workloadToPods[*podWorkload]; !exists {
		d.workloadToPods[*podWorkload] = make(map[types.NamespacedName]struct{})
	}
	d.workloadToPods[*podWorkload][pod] = struct{}{}

	go func() {
		inst, err := d.instrumentationFactory.CreateEbpfInstrumentation(ctx, pid, appName, podWorkload, containerName, pod.Name)
		if err != nil {
			log.Logger.Error(err, "instrumentation setup failed", "workload", podWorkload, "pod", pod)
			return
		}

		d.mux.Lock()
		_, stillExists := d.pidsAttemptedInstrumentation[pid]
		if stillExists {
			d.pidsToInstrumentation[pid] = inst
			d.mux.Unlock()
		} else {
			d.mux.Unlock()
			// we attempted to instrument this process, but it was already cleaned up
			// so we need to clean up the instrumentation we just created
			err = inst.Close(ctx)
			if err != nil {
				log.Logger.Error(err, "error cleaning up instrumentation for process", "pid", pid)
			}
			return
		}

		log.Logger.V(0).Info("Running ebpf instrumentation", "workload", podWorkload, "pod", pod, "language", d.language)

		if err := inst.Run(context.Background()); err != nil {
			log.Logger.Error(err, "instrumentation crashed after running")
		}
	}()

	return nil
}

func (d *EbpfDirector[T]) Language() common.ProgrammingLanguage {
	return d.language
}

func (d *EbpfDirector[T]) Cleanup(pod types.NamespacedName) {
	d.mux.Lock()
	defer d.mux.Unlock()
	details, exists := d.podsToDetails[pod]
	if !exists {
		log.Logger.V(5).Info("No processes to cleanup for pod", "pod", pod)
		return
	}

	log.Logger.V(0).Info("Cleaning up ebpf go instrumentation for pod", "pod", pod)
	delete(d.podsToDetails, pod)

	// clear the pod from the workloadToPods map
	workload := details.Workload
	delete(d.workloadToPods[*workload], pod)
	if len(d.workloadToPods[*workload]) == 0 {
		delete(d.workloadToPods, *workload)
	}

	for _, pid := range details.Pids {
		delete(d.pidsAttemptedInstrumentation, pid)

		inst, exists := d.pidsToInstrumentation[pid]
		if !exists {
			log.Logger.V(5).Info("No objects to cleanup for process", "pid", pid)
			continue
		}

		delete(d.pidsToInstrumentation, pid)
		go func() {
			err := inst.Close(context.Background())
			if err != nil {
				log.Logger.Error(err, "error cleaning up objects for process", "pid", pid)
			}
		}()
	}
}

func (d *EbpfDirector[T]) Shutdown() {
	log.Logger.V(0).Info("Shutting down instrumentation director")
	for details := range d.podsToDetails {
		d.Cleanup(details)
	}
}

func (d *EbpfDirector[T]) GetWorkloadInstrumentations(workload *common.PodWorkload) []T {
	d.mux.Lock()
	defer d.mux.Unlock()

	pods, ok := d.workloadToPods[*workload]
	if !ok {
		return nil
	}

	var insts []T
	for pod := range pods {
		details, ok := d.podsToDetails[pod]
		if !ok {
			continue
		}

		for _, pid := range details.Pids {
			inst, ok := d.pidsToInstrumentation[pid]
			if !ok {
				continue
			}

			insts = append(insts, inst)
		}
	}

	return insts
}
