package ebpf

import (
	"context"
	"sync"

	_ "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/k8s"
	runtime_details "github.com/odigos-io/odigos/odiglet/pkg/kube/runtime_details"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"k8s.io/apimachinery/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This interface should be implemented by all ebpf sdks
// for example, the go auto instrumentation sdk implements it
type OtelEbpfSdk interface {
	Run(ctx context.Context) error
	Close(ctx context.Context) error
}

// users can use different eBPF otel SDKs by returning them from this function
type InstrumentationFactory[T OtelEbpfSdk] interface {
	CreateEbpfInstrumentation(ctx context.Context, pid int, serviceName string, podWorkload *common.PodWorkload, containerName string, podName string, loadedIndicator chan struct{}) (T, error)
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

type InstrumentationUnHealthyReason string

const (
	FailedToLoad       InstrumentationUnHealthyReason = "FailedToLoad"
	FailedToInitialize InstrumentationUnHealthyReason = "FailedToInitialize"
)

const ebpfSDKConditionRunning = "ebpfSDKRunning"

type instrumentationStatus struct {
	Workload common.PodWorkload
	PodName  types.NamespacedName
	Healthy bool
	Message string
	Reason  InstrumentationUnHealthyReason
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

	// this channel is used to send the status of the instrumentation SDK after it is created and ran.
	// the status is used to update the status conditions for the instrumentedApplication CR.
	// The status can be either a failure to initialize the SDK, or a failure to load the eBPF probes or a success which
	// means the eBPF probes were loaded successfully.
	// TODO: this channel should probably be buffered, so we don't block the instrumentation goroutine?
	instrumentationStatusChan chan instrumentationStatus

	// k8s client used to update status conditions for the instrumentedApplication CR
	client client.Client
}

type DirectorKey struct {
	Language common.ProgrammingLanguage
	common.OtelSdk
}

type DirectorsMap map[DirectorKey]Director

func NewEbpfDirector[T OtelEbpfSdk](ctx context.Context, client client.Client, language common.ProgrammingLanguage, instrumentationFactory InstrumentationFactory[T]) *EbpfDirector[T] {
	director := &EbpfDirector[T]{
		language:                     language,
		instrumentationFactory:       instrumentationFactory,
		pidsToInstrumentation:        make(map[int]T),
		pidsAttemptedInstrumentation: make(map[int]struct{}),
		podsToDetails:                make(map[types.NamespacedName]podDetails),
		workloadToPods:               make(map[common.PodWorkload]map[types.NamespacedName]struct{}),
		instrumentationStatusChan:    make(chan instrumentationStatus),
		client:                       client,
	}

	go director.observeInstrumentations(ctx)

	return director
}

func (d *EbpfDirector[T]) observeInstrumentations(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case status, more := <-d.instrumentationStatusChan:
			if !more {
				return
			}

			if d.client == nil {
				log.Logger.V(0).Info("Client is nil, cannot update status conditions", "workload", status.Workload)
				continue
			}

			runtimeDetails, err := runtime_details.GetRuntimeDetails(ctx, d.client, &status.Workload)
			if err != nil {
				log.Logger.Error(err, "error getting runtime details", "workload", status.Workload)
				continue
			}

			// write the status to the CR. Since we are writing the status to the instrumentedApplication CR,
			// we might overwrite the status of another pod which corresponds to the same workload.
			// this can cause the status to not represent the full state of the workload, in case some of the pods are healthy and some are not for the same workload.
			if !status.Healthy {
				log.Logger.Error(nil, "eBPF instrumentation unhealthy", "reason", status.Reason, "message", status.Reason, "workload", status.Workload)
				msg := "Failed to load eBPF probes to pod: " + status.PodName.String() + ".message: " + status.Message
				err := k8s.UpdateStatusConditions(ctx, d.client, runtimeDetails, &runtimeDetails.Status.Conditions, metav1.ConditionFalse, ebpfSDKConditionRunning, string(status.Reason), msg)
				if err != nil {
					log.Logger.Error(err, "error updating status conditions", "workload", status.Workload)
				}
			} else {
				log.Logger.V(0).Info("eBPF instrumentation healthy", "workload", status.Workload)
				err := k8s.UpdateStatusConditions(ctx, d.client, runtimeDetails, &runtimeDetails.Status.Conditions, metav1.ConditionTrue, ebpfSDKConditionRunning, "Success", "Successfully loaded eBPF probes to pod: " + status.PodName.String())
				if err != nil {
					log.Logger.Error(err, "error updating status conditions", "workload", status.Workload)
				}
			}
		}
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

	loadedIndicator := make(chan struct{})
	loadedCtx, loadedObserverCancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-loadedCtx.Done():
			// TODO: remove this print before merging
			log.Logger.V(0).Info("eBPF instrumentation loaded observer cancelled", "workload", podWorkload, "pod", pod)
			return
		case <-loadedIndicator:
			d.instrumentationStatusChan <- instrumentationStatus{Healthy: true, Workload: *podWorkload, PodName: pod}
		}
	}()

	go func() {
		// once the instrumentation finished running (either by error or successful exit), we can cancel the 'loaded' observer for this instrumentation
		defer loadedObserverCancel()
		inst, err := d.instrumentationFactory.CreateEbpfInstrumentation(ctx, pid, appName, podWorkload, containerName, pod.Name, loadedIndicator)
		if err != nil {
			d.instrumentationStatusChan <- instrumentationStatus{Healthy: false, Message: err.Error(), Workload: *podWorkload, Reason: FailedToInitialize, PodName: pod}
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
			d.instrumentationStatusChan <- instrumentationStatus{Healthy: false, Message: err.Error(), Workload: *podWorkload, Reason: FailedToLoad, PodName: pod}
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
	close(d.instrumentationStatusChan)
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
