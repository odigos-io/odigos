package ebpf

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/container"
	instance "github.com/odigos-io/odigos/k8sutils/pkg/instrumentation_instance"
	workloadUtils "github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/detector"
	"github.com/odigos-io/odigos/odiglet/pkg/kube/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrContainerNotInPodSpec    = errors.New("container not found in pod spec")
	ErrContainerNameNotReported = errors.New("container name not reported in environment variables")
	ErrDeviceNotDetected        = errors.New("device not detected")
	ErrNoInstrumentationFactory = errors.New("no ebpf factory found")
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

type Manager struct {
	ProcEvents   chan detector.ProcessEvent
	Client       client.Client
	Factories    map[FactoryID]Factory
	Logger       logr.Logger
	DetailsByPid map[int]InstrumentationDetails

	done chan struct{}
	stop context.CancelFunc
}

func NewManager(client client.Client, logger logr.Logger, factories map[FactoryID]Factory, procEvents chan detector.ProcessEvent) *Manager {
	return &Manager{
		ProcEvents:   procEvents,
		Client:       client,
		Factories:    factories,
		Logger:       logger,
		DetailsByPid: make(map[int]InstrumentationDetails),
		done:         make(chan struct{}),
	}
}

func (m *Manager) Run(ctx context.Context) {
	defer close(m.done)

	runLoopCtx, stop := context.WithCancel(ctx)
	m.stop = stop

	for {
		select {
		case <-runLoopCtx.Done():
			m.Logger.Info("stopping Odiglet instrumentation manager")
			for pid, details := range m.DetailsByPid {
				err := details.Inst.Close(ctx)
				if err != nil {
					m.Logger.Error(err, "failed to close instrumentation", "pid", pid)
				}
				delete(m.DetailsByPid, pid)
				// probably shouldn't remove instrumentation instance here
				// as this flow is happening when Odiglet is shutting down
			}
			return
		case e := <-m.ProcEvents:
			switch e.EventType {
			case detector.ProcessExecEvent:
				m.Logger.Info("detected new process", "pid", e.PID, "cmd", e.ExecDetails.CmdLine)
				err := m.handleProcessExecEvent(runLoopCtx, e)
				// ignore the error if no instrumentation factory is found,
				// as this is expected for some language and sdk combinations
				if err != nil && !errors.Is(err, ErrNoInstrumentationFactory) {
					m.Logger.Error(err, "failed to handle process exec event")
				}
			case detector.ProcessExitEvent:
				m.cleanInstrumentation(runLoopCtx, e.PID)
			}
		}
	}
}

func (m *Manager) Stop() {
	m.stop()
	<-m.done
}

func (m *Manager) cleanInstrumentation(ctx context.Context, pid int) {
	details, found := m.DetailsByPid[pid]
	if !found {
		m.Logger.V(3).Info("no instrumentation found for exiting pid", "pid", pid)
		return
	}

	m.Logger.Info("cleaning instrumentation resources", "pid", pid)

	err := details.Inst.Close(ctx)
	if err != nil {
		m.Logger.Error(err, "failed to close instrumentation")
	}

	// remove instrumentation instance
	if err = m.Client.Delete(ctx, &odigosv1.InstrumentationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.InstrumentationInstanceName(details.Pod.Name, pid),
			Namespace: details.Pod.Namespace,
		},
	}); err != nil && !apierrors.IsNotFound(err) {
		m.Logger.Error(err, "error deleting instrumentation instance", "pod", details.Pod.Name, "pid", pid)
	}

	delete(m.DetailsByPid, pid)
}

func (m *Manager) handleProcessExecEvent(ctx context.Context, e detector.ProcessEvent) error {
	if _, found := m.DetailsByPid[e.PID]; found {
		// this can happen if we have multiple exec events for the same pid (chain loading)
		// TODO: better handle this?
		// this can be done by first closing the existing instrumentation,
		// and then creating a new one
		m.Logger.Info("instrumentation already exists for pid", "pid", e.PID)
		return nil
	}

	// get the corresponding pod object for this process event
	pod, err := m.podFromProcEvent(e)
	if err != nil {
		return fmt.Errorf("failed to get pod from process event: %w", err)
	}

	containerName, found := containerNameFromProcEvent(e)
	if !found {
		return ErrContainerNameNotReported
	}

	// get the language and sdk for this process event
	// based on the pod spec and the container name from the process event
	// TODO: We should have all the required information in the process event
	// to determine the language - hence in the future we can improve this
	lang, sdk, err := m.languageSdk(pod, containerName)
	if err != nil {
		return fmt.Errorf("failed to get language and sdk: %w", err)
	}

	factory, found := m.Factories[FactoryID{Language: lang, OtelSdk: sdk}]
	if !found {
		return ErrNoInstrumentationFactory
	}

	workload, err := workloadUtils.PodWorkloadObject(ctx, pod)
	if err != nil {
		return fmt.Errorf("failed to get workload object: %w", err)
	}

	settings := Settings{
		ServiceName:        workload.Name,
		ResourceAttributes: utils.GetResourceAttributes(workload, pod.Name),
	}
	inst, err := factory.CreateInstrumentation(ctx, e.PID, settings)
	if err != nil {
		m.Logger.Error(err, "failed to initialize instrumentation")

		// write instrumentation instance CR with error status
		err = m.updateInstrumentationInstanceStatus(ctx, pod, containerName, workload, e.PID, InstrumentationUnhealthy, FailedToInitialize, err.Error())
		// TODO: should we return here the initialize error? or the instance write error? or both?
		return err
	}

	err = inst.Load(ctx)
	if err != nil {
		m.Logger.Error(err, "failed to load instrumentation")

		// write instrumentation instance CR with error status
		err = m.updateInstrumentationInstanceStatus(ctx, pod, containerName, workload, e.PID, InstrumentationUnhealthy, FailedToLoad, err.Error())
		// TODO: should we return here the load error? or the instance write error? or both?
		return err
	}

	m.DetailsByPid[e.PID] = InstrumentationDetails{
		Inst: inst,
		Pod:  types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name},
	}

	m.Logger.Info("instrumentation loaded", "pid", e.PID, "pod", pod.Name)

	// write instrumentation instance CR with success status
	msg := fmt.Sprintf("Successfully loaded eBPF probes to pod: %s container: %s", pod.Name, containerName)
	err = m.updateInstrumentationInstanceStatus(ctx, pod, containerName, workload, e.PID, InstrumentationHealthy, LoadedSuccessfully, msg)
	if err != nil {
		m.Logger.Error(err, "failed to update instrumentation instance for successful load")
	}

	go func() {
		err := inst.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			m.Logger.Error(err, "failed to run instrumentation")
			// these errors occur after the instrumentation is loaded
			// write instrumentation instance CR with error status
			err = m.updateInstrumentationInstanceStatus(ctx, pod, containerName, workload, e.PID, InstrumentationUnhealthy, FailedToRun, err.Error())
			if err != nil {
				m.Logger.Error(err, "failed to update instrumentation instance for failed instrumentation run")
			}
		}
	}()

	return nil
}

func (m *Manager) updateInstrumentationInstanceStatus(ctx context.Context, pod *corev1.Pod, containerName string, w *workloadUtils.PodWorkload, pid int, health InstrumentationHealth, reason InstrumentationStatusReason, msg string) error {
	instrumentedAppName := workloadUtils.CalculateWorkloadRuntimeObjectName(w.Name, w.Kind)
	healthy := bool(health)
	return instance.UpdateInstrumentationInstanceStatus(ctx, pod, containerName, m.Client, instrumentedAppName, pid, m.Client.Scheme(),
		instance.WithHealthy(&healthy, string(reason), &msg),
	)
}

func (m *Manager) podFromProcEvent(event detector.ProcessEvent) (*corev1.Pod, error) {
	eventEnvs := event.ExecDetails.Environments

	podName, ok := eventEnvs[consts.OdigosEnvVarPodName]
	if !ok {
		return nil, fmt.Errorf("missing %s in environment variables", consts.OdigosEnvVarPodName)
	}

	podNamespace, ok := eventEnvs[consts.OdigosEnvVarNamespace]
	if !ok {
		return nil, fmt.Errorf("missing %s in environment variables", consts.OdigosEnvVarNamespace)
	}

	pod := corev1.Pod{}
	err := m.Client.Get(context.Background(), client.ObjectKey{Namespace: podNamespace, Name: podName}, &pod)
	if err != nil {
		return nil, fmt.Errorf("error fetching pod object: %w", err)
	}

	return &pod, nil
}

func containerNameFromProcEvent(event detector.ProcessEvent) (string, bool) {
	containerName, ok := event.ExecDetails.Environments[consts.OdigosEnvVarContainerName]
	return containerName, ok
}

func (m *Manager) languageSdk(pod *corev1.Pod, containerName string) (common.ProgrammingLanguage, common.OtelSdk, error) {
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			language, sdk, found := odgiosK8s.GetLanguageAndOtelSdk(container)
			if !found {
				return common.UnknownProgrammingLanguage, common.OtelSdk{}, ErrDeviceNotDetected
			}

			return language, sdk, nil
		}
	}

	return common.UnknownProgrammingLanguage, common.OtelSdk{}, ErrContainerNotInPodSpec
}
