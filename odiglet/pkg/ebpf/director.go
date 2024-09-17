package ebpf

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/odigos-io/odigos/procdiscovery/pkg/process"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	inst "github.com/odigos-io/odigos/k8sutils/pkg/instrumentation_instance"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This interface should be implemented by all ebpf sdks
// for example, the go auto instrumentation sdk implements it
type OtelEbpfSdk interface {
	Run(ctx context.Context) error
	Close(ctx context.Context) error
}

type ConfigurableOtelEbpfSdk interface {
	OtelEbpfSdk
	ApplyConfig(ctx context.Context, config *odigosv1.InstrumentationConfig) error
}

// users can use different eBPF otel SDKs by returning them from this function
type InstrumentationFactory[T OtelEbpfSdk] interface {
	CreateEbpfInstrumentation(ctx context.Context, pid int, serviceName string, podWorkload *workload.PodWorkload, containerName string, podName string, loadedIndicator chan struct{}) (T, error)
}

// Director manages the instrumentation for a specific SDK in a specific language
type Director interface {
	Language() common.ProgrammingLanguage
	Instrument(ctx context.Context, pid int, podDetails types.NamespacedName, podWorkload *workload.PodWorkload, appName string, containerName string) error
	Cleanup(podDetails types.NamespacedName)
	Shutdown()
	// TODO: once all our implementation move to this function we can rename it to ApplyInstrumentationConfig,
	// currently that name is reserved for the old API until it is removed.
	ApplyInstrumentationConfiguration(ctx context.Context, workload *workload.PodWorkload, instrumentationConfig *odigosv1.InstrumentationConfig) error
	ShouldInstrument(pid int, details []process.Details) bool
}

type InstrumentedProcess[T OtelEbpfSdk] struct {
	PID  int
	inst T
	// Used to make sure the instrumentation is run Once for the given process
	runOnce sync.Once
}

type podDetails[T OtelEbpfSdk] struct {
	Workload *workload.PodWorkload
	InstrumentedProcesses []*InstrumentedProcess[T]
}

type InstrumentationStatusReason string

const (
	FailedToLoad       InstrumentationStatusReason = "FailedToLoad"
	FailedToInitialize InstrumentationStatusReason = "FailedToInitialize"
	LoadedSuccessfully InstrumentationStatusReason = "LoadedSuccessfully"
)

const CleanupInterval = 10 * time.Second

type instrumentationStatus struct {
	Workload      workload.PodWorkload
	PodName       types.NamespacedName
	ContainerName string
	Healthy       bool
	Message       string
	Reason        InstrumentationStatusReason
	Pid           int
}

type EbpfDirector[T OtelEbpfSdk] struct {
	mux sync.Mutex

	language               common.ProgrammingLanguage
	instrumentationFactory InstrumentationFactory[T]

	// via this map, we can find the workload and pids for a specific pod.
	// sometimes we only have the pod name and namespace, so this map is useful.
	podsToDetails map[types.NamespacedName]podDetails[T]

	// this map can be used when we only have the workload, and need to find the pods to derive pids.
	workloadToPods map[workload.PodWorkload]map[types.NamespacedName]struct{}

	// this channel is used to send the status of the instrumentation SDK after it is created and ran.
	// the status is used to update the status conditions for the instrumentedApplication CR.
	// The status can be either a failure to initialize the SDK, or a failure to load the eBPF probes or a success which
	// means the eBPF probes were loaded successfully.
	// TODO: this channel should probably be buffered, so we don't block the instrumentation goroutine?
	instrumentationStatusChan chan instrumentationStatus

	// k8s client used to update status conditions for the instrumentedApplication CR
	client client.Client
}

func (d *EbpfDirector[T]) ShouldInstrument(pid int, details []process.Details) bool {
	return true
}

type DirectorKey struct {
	Language common.ProgrammingLanguage
	common.OtelSdk
}

type DirectorsMap map[DirectorKey]Director

var _ Director = &EbpfDirector[*GoOtelEbpfSdk]{}

func NewEbpfDirector[T OtelEbpfSdk](ctx context.Context, client client.Client, scheme *runtime.Scheme, language common.ProgrammingLanguage, instrumentationFactory InstrumentationFactory[T]) *EbpfDirector[T] {
	director := &EbpfDirector[T]{
		language:                     language,
		instrumentationFactory:       instrumentationFactory,
		podsToDetails:  make(map[types.NamespacedName]podDetails[T]),
		workloadToPods:               make(map[workload.PodWorkload]map[types.NamespacedName]struct{}),
		instrumentationStatusChan:    make(chan instrumentationStatus),
		client:                       client,
	}

	go director.observeInstrumentations(ctx, scheme)
	go director.periodicCleanup(ctx)

	return director
}

func (d *EbpfDirector[T]) periodicCleanup(ctx context.Context) {
	ticker := time.NewTicker(CleanupInterval)

	isProcessExists := func(pid int) bool {
		p, err := os.FindProcess(pid)
		if err != nil {
			return false
		}
		// To check if the process exists, we send signal 0 to the process
		// this is the standard way to check if a process exists in unix
		err = p.Signal(syscall.Signal(0))
		return err == nil
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.mux.Lock()
			for _, details := range d.podsToDetails {
				for i := range details.InstrumentedProcesses {
					ip := details.InstrumentedProcesses[i]
					if !isProcessExists(ip.PID) {
						log.Logger.V(0).Info("Instrumented process does not exist, cleaning up", "pid", ip.PID)
						err := ip.inst.Close(ctx)
						if err != nil {
							log.Logger.Error(err, "error cleaning up instrumentation for process", "pid", ip.PID)
						}
						details.InstrumentedProcesses = append(details.InstrumentedProcesses[:i], details.InstrumentedProcesses[i+1:]...)
					}
				}
			}
			d.mux.Unlock()
		}
	}
}

func (d *EbpfDirector[T]) ApplyInstrumentationConfiguration(ctx context.Context, workload *workload.PodWorkload, instrumentationConfig *odigosv1.InstrumentationConfig) error {
	var t T
	if _, ok := any(t).(ConfigurableOtelEbpfSdk); !ok {
		log.Logger.V(1).Info("eBPF SDK is not configurable, skip applying configuration", "language", d.Language())
		return nil
	}

	d.mux.Lock()
	defer d.mux.Unlock()

	insts := d.GetWorkloadInstrumentations(workload)

	log.Logger.V(3).Info("Applying config to instrumentations after CRD change", "instrumentationConfig", instrumentationConfig, "workload", workload, "SDKs count", len(insts))

	var retErr []error
	for _, inst := range insts {
		// The type assertion is safe because we made sure the SDK for this director implements the ConfigurableOtelEbpfSdk interface
		err := any(inst).(ConfigurableOtelEbpfSdk).ApplyConfig(ctx, instrumentationConfig)
		if err != nil {
			retErr = append(retErr, err)
		}
	}
	if len(retErr) > 0 {
		return fmt.Errorf("failed to apply config to %d instrumentations", len(retErr))
	}
	return nil
}

func (d *EbpfDirector[T]) observeInstrumentations(ctx context.Context, scheme *runtime.Scheme) {
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

			var pod corev1.Pod
			err := d.client.Get(ctx, status.PodName, &pod)
			if err != nil {
				log.Logger.Error(err, "error getting pod", "workload", status.Workload)
				continue
			}

			instrumentedAppName := workload.CalculateWorkloadRuntimeObjectName(status.Workload.Name, status.Workload.Kind)
			err = inst.UpdateInstrumentationInstanceStatus(ctx, &pod, status.ContainerName, d.client, instrumentedAppName, status.Pid, scheme,
				inst.WithHealthy(&status.Healthy, string(status.Reason), &status.Message),
			)

			if err != nil {
				log.Logger.Error(err, "error updating instrumentation instance status", "workload", status.Workload)
			}
		}
	}
}

func (d *EbpfDirector[T]) Instrument(ctx context.Context, pid int, pod types.NamespacedName, podWorkload *workload.PodWorkload, appName string, containerName string) error {
	log.Logger.V(0).Info("Instrumenting process", "pid", pid, "workload", podWorkload)
	d.mux.Lock()
	defer d.mux.Unlock()

	podDetails, exists := d.podsToDetails[pod]
	ip := InstrumentedProcess[T]{PID: pid}
	if !exists {
		// the first we instrument processes in this pod
		podDetails.Workload = podWorkload
		podDetails.InstrumentedProcesses = []*InstrumentedProcess[T]{&ip}
	} else {
		// check if the process is already instrumented
		for i := range podDetails.InstrumentedProcesses {
			if podDetails.InstrumentedProcesses[i].PID == pid {
				log.Logger.V(5).Info("Process already instrumented", "pid", pid, "pod", pod)
				return nil
			}
		}
		podDetails.InstrumentedProcesses = append(podDetails.InstrumentedProcesses, &ip)
	}

	d.podsToDetails[pod] = podDetails

	if _, exists := d.workloadToPods[*podWorkload]; !exists {
		d.workloadToPods[*podWorkload] = make(map[types.NamespacedName]struct{})
	}
	d.workloadToPods[*podWorkload][pod] = struct{}{}

	ip.runOnce.Do(func() {
		loadedIndicator := make(chan struct{})
		loadedCtx, loadedObserverCancel := context.WithCancel(ctx)
		go func() {
			select {
			case <-loadedCtx.Done():
				return
			case <-loadedIndicator:
				d.instrumentationStatusChan <- instrumentationStatus{
					Healthy:       true,
					Message:       "Successfully loaded eBPF probes to pod: " + pod.String(),
					Workload:      *podWorkload,
					Reason:        LoadedSuccessfully,
					PodName:       pod,
					ContainerName: containerName,
					Pid:           pid,
				}
			}
		}()
	
		go func() {
			// once the instrumentation finished running (either by error or successful exit), we can cancel the 'loaded' observer for this instrumentation
			defer loadedObserverCancel()
			inst, err := d.instrumentationFactory.CreateEbpfInstrumentation(ctx, pid, appName, podWorkload, containerName, pod.Name, loadedIndicator)
			if err != nil {
				d.instrumentationStatusChan <- instrumentationStatus{
					Healthy:       false,
					Message:       err.Error(),
					Workload:      *podWorkload,
					Reason:        FailedToInitialize,
					PodName:       pod,
					ContainerName: containerName,
					Pid:           pid,
				}
				return
			}

			ip.inst = inst
	
			log.Logger.V(0).Info("Running ebpf instrumentation", "workload", podWorkload, "pod", pod, "language", d.language)
	
			if err := inst.Run(ctx); err != nil {
				d.instrumentationStatusChan <- instrumentationStatus{
					Healthy:       false,
					Message:       err.Error(),
					Workload:      *podWorkload,
					Reason:        FailedToLoad,
					PodName:       pod,
					ContainerName: containerName,
					Pid:           pid,
				}
			}
		}()
	})

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

	log.Logger.V(0).Info("Cleaning up ebpf instrumentation for pod", "pod", pod, "language", d.language)
	delete(d.podsToDetails, pod)

	// clear the pod from the workloadToPods map
	workload := details.Workload
	delete(d.workloadToPods[*workload], pod)
	if len(d.workloadToPods[*workload]) == 0 {
		delete(d.workloadToPods, *workload)
	}

	err := d.client.Delete(context.Background(), &odigosv1.InstrumentationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
	})

	// the instrumentation instance might already be deleted at this point if the pod was deleted
	if err != nil && !apierrors.IsNotFound(err) {
		log.Logger.Error(err, "error deleting instrumentation instance", "pod", pod)
	}

	for _, ip := range details.InstrumentedProcesses {
		go func() {
			err := ip.inst.Close(context.Background())
			if err != nil {
				log.Logger.Error(err, "error cleaning up objects for process", "pid", ip.PID)
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

func (d *EbpfDirector[T]) GetWorkloadInstrumentations(workload *workload.PodWorkload) []T {
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

		for _, ip := range details.InstrumentedProcesses {
			insts = append(insts, ip.inst)
		}
	}

	return insts
}
