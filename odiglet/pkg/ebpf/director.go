package ebpf

import (
	"context"
	"sync"

	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
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
	CreateEbpfInstrumentation(ctx context.Context, pid int, serviceName string, podWorkload *common.PodWorkload) (T, error)
}

// Director manages the instrumentation for a specific SDK in a specific language
type Director interface {
	Language() common.ProgrammingLanguage
	Instrument(ctx context.Context, pid int, podDetails types.NamespacedName, podWorkload *common.PodWorkload, appName string) error
	Cleanup(podDetails types.NamespacedName)
	Shutdown()
}

type pidDetails[T OtelEbpfSdk] struct {

	// The instrumentation manager object
	// if IsRunning is false, this value is not valid and should not be used
	Instrumentation T

	// this indicates if we called "Run" on the instrumentation.
	// since T is OtelEbpfSdk which is an interface, we could have used nil
	// to denote that, but go compiler gets confused and can't always tell if T is an interface
	//
	// if IsRunning is false, the instrumentation is in the process of being created.
	IsRunning bool

	ShouldBeInstrumented bool
}

type podDetails struct {
	Workload *common.PodWorkload

	// pids we instrumented for this pod, or we are in the process of instrumenting async.
	// every pid for which we store pidDetails, should be in this list.
	Pids []int
}

type EbpfDirector[T OtelEbpfSdk] struct {
	mux sync.Mutex

	language common.ProgrammingLanguage

	// this is used to create the instrumentation manager for each pid managed by this director
	instrumentationFactory InstrumentationFactory[T]

	// we store details on each pid we instrumented, or are in the process of instrumenting
	pidsDetails map[int]pidDetails[T]

	// via this map, we can find the workload and pids for a specific pod.
	// sometimes we only have the pod name and namespace, so this map is useful.
	podsToDetails map[types.NamespacedName]podDetails

	// this map can be used when we only have the workload, and need to find the pods to derive pids.
	workloadToPods map[common.PodWorkload]map[types.NamespacedName]struct{}
}

func NewEbpfDirector[T OtelEbpfSdk](language common.ProgrammingLanguage, instrumentationFactory InstrumentationFactory[T]) *EbpfDirector[T] {
	return &EbpfDirector[T]{
		language:               language,
		instrumentationFactory: instrumentationFactory,
		pidsDetails:            make(map[int]pidDetails[T]),
		podsToDetails:          make(map[types.NamespacedName]podDetails),
		workloadToPods:         make(map[common.PodWorkload]map[types.NamespacedName]struct{}),
	}
}

func (d *EbpfDirector[T]) Instrument(ctx context.Context, pid int, pod types.NamespacedName, podWorkload *common.PodWorkload, serviceName string) error {
	log.Logger.V(0).Info("Instrumenting process", "pid", pid, "workload", podWorkload)
	d.mux.Lock()
	defer d.mux.Unlock()

	// check if we already instrumented this process
	if pidDetails, exists := d.pidsDetails[pid]; exists {
		// for some reason, the compiler is not able to infer that T is an interface
		if pidDetails.IsRunning {
			// we already instrumented this process
			log.Logger.V(5).Info("Process already instrumented", "pid", pid)
		} else {
			log.Logger.V(5).Info("pid is in the process of setting up instrumentation", "pid", pid)
		}
		return nil
	}

	// mark that this pid is in the process of being instrumented
	d.pidsDetails[pid] = pidDetails[T]{IsRunning: false, ShouldBeInstrumented: true}

	// upsert the pod details, and add the pid to the list of pids for this pod
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

	// update workload info
	if _, exists := d.workloadToPods[*podWorkload]; !exists {
		d.workloadToPods[*podWorkload] = make(map[types.NamespacedName]struct{})
	}
	d.workloadToPods[*podWorkload][pod] = struct{}{}

	go d.createPidInstrumentationAsync(ctx, pid, pod, podWorkload, serviceName)

	return nil
}

func (d *EbpfDirector[T]) Language() common.ProgrammingLanguage {
	return d.language
}

func (d *EbpfDirector[T]) Cleanup(pod types.NamespacedName) {
	d.mux.Lock()
	defer d.mux.Unlock()

	// cleanup pod details
	details, exists := d.podsToDetails[pod]
	if !exists {
		log.Logger.V(5).Info("No processes to cleanup for pod", "pod", pod)
		return
	}

	// cleanup pids details and Close the instrumentation manager
	uncleanedPids := []int{}
	for _, pid := range details.Pids {

		pidDetails, exists := d.pidsDetails[pid]
		if !exists {
			log.Logger.V(5).Info("No objects to cleanup for process", "pid", pid)
			continue
		}

		if !pidDetails.IsRunning {
			log.Logger.V(0).Info("Attempting to cleanup instrumentation which is not yet running. Flagging to be done later", "pid", pid)
			uncleanedPids = append(uncleanedPids, pid)
			continue
		}

		err := pidDetails.Instrumentation.Close(context.Background())
		if err != nil {
			log.Logger.Error(err, "error cleaning up objects for process", "pid", pid)
		}
	}

	if len(uncleanedPids) == 0 {
		d.removePodFromDirector(pod)
	} else {
		details.Pids = uncleanedPids
		d.podsToDetails[pod] = details
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
			pidDetails, ok := d.pidsDetails[pid]
			if !ok || !pidDetails.IsRunning {
				continue
			}

			insts = append(insts, pidDetails.Instrumentation)
		}
	}

	return insts
}

func (d *EbpfDirector[T]) createPidInstrumentationAsync(ctx context.Context, pid int, pod types.NamespacedName, podWorkload *common.PodWorkload, serviceName string) {

	// this operation might take some time, so we should do it async and do not lock
	inst, err := d.instrumentationFactory.CreateEbpfInstrumentation(ctx, pid, serviceName, podWorkload)
	if err != nil {
		// TODO: should we remove it from the map?
		log.Logger.Error(err, "instrumentation setup failed", "workload", podWorkload, "pod", pod)
		return
	}

	d.mux.Lock()
	pidDetails := d.pidsDetails[pid]

	// it is possible that while we were setting up the instrumentation, the pod was deleted or uninstrumented
	// in which case, we should undo the instrumentation we just created and cleanup
	if !pidDetails.ShouldBeInstrumented {
		inst.Close(ctx)

		// update pid details
		delete(d.pidsDetails, pid)

		// update pod details
		podDetails := d.podsToDetails[pod]
		for i, p := range podDetails.Pids {
			if p == pid {
				podDetails.Pids = append(podDetails.Pids[:i], podDetails.Pids[i+1:]...)
				break
			}
		}
		if len(podDetails.Pids) == 0 {
			d.removePodFromDirector(pod)
		} else {
			d.podsToDetails[pod] = podDetails
		}
		d.mux.Unlock()

		return
	}

	// update the pid details before running the instrumentation manager
	pidDetails.Instrumentation = inst
	pidDetails.IsRunning = true
	d.pidsDetails[pid] = pidDetails

	d.mux.Unlock()

	log.Logger.V(0).Info("Running ebpf instrumentation", "workload", podWorkload, "pod", pod, "language", d.language)

	// Run is blocking
	if err := inst.Run(ctx); err != nil {
		log.Logger.Error(err, "instrumentation crashed after running")
	}
}

// cleanup pod details from the director.
// this function should be called when the mutex is locked
func (d *EbpfDirector[T]) removePodFromDirector(pod types.NamespacedName) {
	podDetails, exists := d.podsToDetails[pod]
	if !exists {
		return
	}
	podWorkload := *podDetails.Workload

	// cleanup pod details
	delete(d.podsToDetails, pod)

	// cleanup workload details
	delete(d.workloadToPods[podWorkload], pod)
	if len(d.workloadToPods[podWorkload]) == 0 {
		delete(d.workloadToPods, podWorkload)
	}
}
