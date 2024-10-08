package ebpf

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
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
	// Run starts the eBPF instrumentation.
	// It should block until the instrumentation is stopped or the context is canceled or an error occurs.
	Run(ctx context.Context) error
	// Close cleans up the resources associated with the eBPF instrumentation.
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
	closed  atomic.Bool
}

type podDetails[T OtelEbpfSdk] struct {
	Workload              *workload.PodWorkload
	InstrumentedProcesses []*InstrumentedProcess[T]
}

type InstrumentationStatusReason string

const (
	FailedToLoad       InstrumentationStatusReason = "FailedToLoad"
	FailedToInitialize InstrumentationStatusReason = "FailedToInitialize"
	LoadedSuccessfully InstrumentationStatusReason = "LoadedSuccessfully"
)

// CleanupInterval is the interval in which the director will check if the instrumented processes are still running
// and clean up the resources associated to the ones that are not.
// It is not const for testing purposes.
var CleanupInterval = 30 * time.Second

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
	podsToDetails map[types.NamespacedName]*podDetails[T]

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

func NewEbpfDirector[T OtelEbpfSdk](ctx context.Context, client client.Client, scheme *runtime.Scheme, language common.ProgrammingLanguage, instrumentationFactory InstrumentationFactory[T]) *EbpfDirector[T] {
	director := &EbpfDirector[T]{
		language:                  language,
		instrumentationFactory:    instrumentationFactory,
		podsToDetails:             make(map[types.NamespacedName]*podDetails[T]),
		workloadToPods:            make(map[workload.PodWorkload]map[types.NamespacedName]struct{}),
		instrumentationStatusChan: make(chan instrumentationStatus),
		client:                    client,
	}

	go director.observeInstrumentations(ctx, scheme)
	go director.periodicCleanup(ctx)

	return director
}

// defining this function here allows mocking it in tests
var IsProcessExists = func(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// To check if the process exists, we send signal 0 to the process
	// this is the standard way to check if a process exists in unix
	err = p.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}

	if errors.Is(err, os.ErrProcessDone) {
		return false
	}

	errno, ok := err.(syscall.Errno)
	if !ok {
		return false
	}

	if errno == syscall.EPERM {
		// we don't have permission to send signal 0 to the process
		// so we assume the process exists, to avoid removing the instrumentation
		return true
	}

	return false
}

// Since OtelEbpfSdk is a generic type, we can't simply check it is nil with inst == nil
func isNil[T OtelEbpfSdk](inst T) bool {
	return reflect.ValueOf(&inst).Elem().IsZero()
}

func (d *EbpfDirector[T]) periodicCleanup(ctx context.Context) {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.mux.Lock()
			for pod, details := range d.podsToDetails {
				newInstrumentedProcesses := make([]*InstrumentedProcess[T], 0, len(details.InstrumentedProcesses))
				for i := range details.InstrumentedProcesses {
					ip := details.InstrumentedProcesses[i]
					// if the process does not exist, we should make sure we clean the instrumentation resources.
					// Also making sure the instrumentation itself is not nil to avoid closing it here.
					// This can happen if the process exits while the instrumentation is initializing.
					if !IsProcessExists(ip.PID) && !isNil(ip.inst) {
						log.Logger.V(0).Info("Instrumented process does not exist, cleaning up", "pid", ip.PID)
						d.cleanProcess(ctx, pod, ip)
					} else {
						newInstrumentedProcesses = append(newInstrumentedProcesses, ip)
					}
				}
				details.InstrumentedProcesses = newInstrumentedProcesses
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

	pd, exists := d.podsToDetails[pod]
	ip := InstrumentedProcess[T]{PID: pid}
	if !exists {
		// the first we instrument processes in this pod
		d.podsToDetails[pod] = &podDetails[T]{
			Workload:              podWorkload,
			InstrumentedProcesses: []*InstrumentedProcess[T]{&ip},
		}
	} else {
		// check if the process is already instrumented
		for i := range pd.InstrumentedProcesses {
			if pd.InstrumentedProcesses[i].PID == pid {
				log.Logger.V(5).Info("Process already instrumented", "pid", pid, "pod", pod)
				return nil
			}
		}
		// New process to instrument in the same pod
		pd.InstrumentedProcesses = append(pd.InstrumentedProcesses, &ip)
	}

	if _, exists := d.workloadToPods[*podWorkload]; !exists {
		d.workloadToPods[*podWorkload] = make(map[types.NamespacedName]struct{})
	}
	d.workloadToPods[*podWorkload][pod] = struct{}{}

	ip.runOnce.Do(func() {
		loadedIndicator := make(chan struct{})
		loadedCtx, loadedObserverCancel := context.WithCancel(ctx)
		// launch an observer for successful loading of the eBPF probes
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

			if ip.closed.Load() {
				log.Logger.Info("Instrumentation already closed before running, stopping instrumentation", "pid", pid)
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

// Cleanup cleans up the resources associated with the given pod including all the instrumented processes.
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

	for _, ip := range details.InstrumentedProcesses {
		d.cleanProcess(context.Background(), pod, ip)
	}
}

// cleanProcess cleans up the resources associated with the given instrumented process in the given pod.
func (d *EbpfDirector[T]) cleanProcess(ctx context.Context, pod types.NamespacedName, ip *InstrumentedProcess[T]) {
	err := ip.inst.Close(ctx)
	if err != nil {
		log.Logger.Error(err, "error cleaning up objects for process", "pid", ip.PID)
	}
	ip.closed.Store(true)

	if err = d.client.Delete(ctx, &odigosv1.InstrumentationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inst.InstrumentationInstanceName(pod.Name, ip.PID),
			Namespace: pod.Namespace,
		},
	}); err != nil && !apierrors.IsNotFound(err) {
		log.Logger.Error(err, "error deleting instrumentation instance", "workload", pod)
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
			if !isNil(ip.inst) {
				insts = append(insts, ip.inst)
			}
		}
	}

	return insts
}
