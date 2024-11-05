package ebpf

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/container"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/detector"
	"github.com/odigos-io/odigos/odiglet/pkg/kube/utils"
	"go.opentelemetry.io/otel/attribute"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrContainerNotInPodSpec    = errors.New("container not found in pod spec")
	ErrContainerNameNotReported = errors.New("container name not reported in environment variables")
	ErrDeviceNotDetected        = errors.New("device not detected")
	ErrNoInstrumentationFactory = errors.New("no ebpf factory found")
)

type Settings struct {
	ServiceName        string
	ResourceAttributes []attribute.KeyValue
	LoadedIndicator    chan struct{}
}

type Factory interface {
	CreateInstrumentation(ctx context.Context, pid int, settings Settings) (OtelEbpfSdk, error)
}

type FactoryID struct {
	Language common.ProgrammingLanguage
	OtelSdk  common.OtelSdk
}

type InstrumentationDetails struct {
	Inst OtelEbpfSdk
	Pod  types.NamespacedName
}

type Manager struct {
	ProcEvents   chan detector.ProcessEvent
	Client       client.Client
	Factories    map[FactoryID]Factory
	Logger       logr.Logger
	DetailsByPid map[int]InstrumentationDetails

	done   chan struct{}
	stop   context.CancelFunc
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
		case <-ctx.Done():
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
				err := m.handleProcessExecEvent(e)
				// ignore the error if no instrumentation factory is found,
				// as this is expected for some language and sdk combinations
				if err != nil && !errors.Is(err, ErrNoInstrumentationFactory) {
					m.Logger.Error(err, "failed to handle process exec event")
				}
			case detector.ProcessExitEvent:
				m.Logger.Info("detected process exit", "pid", e.PID)

				details, found := m.DetailsByPid[e.PID]
				if !found {
					// TODO: move this to debug level
					m.Logger.Info("no instrumentation found for exiting pid", "pid", e.PID)
					continue
				}

				err := details.Inst.Close(runLoopCtx)
				if err != nil {
					m.Logger.Error(err, "failed to close instrumentation")
				}

				delete(m.DetailsByPid, e.PID)

				// TODO: remove instrumentation instance
			}
		}
	}
}

func (m *Manager) Stop() {
	m.stop()
	<-m.done
}

func (m *Manager) handleProcessExecEvent(e detector.ProcessEvent) error {
	// TODO: create context from Run context
	ctx := context.Background()

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

	// get the language and sdk for this process event
	// based on the pod spec and the container name from the process event
	// TODO: We should have all the required information in the process event
	// to determine the language - hence in the future we can improve this
	lang, sdk, err := m.languageSdk(pod, e)
	if err != nil {
		return fmt.Errorf("failed to get language and sdk: %w", err)
	}

	factory, found := m.Factories[FactoryID{Language: lang, OtelSdk: sdk}]
	if !found {
		return ErrNoInstrumentationFactory
	}

	workload, err := workload.PodWorkloadObject(ctx, pod)
	if err != nil {
		return fmt.Errorf("failed to get workload object: %w", err)
	}

	settings := Settings{
		ServiceName:        workload.Name,
		ResourceAttributes: utils.GetResourceAttributes(workload, pod.Name),
		LoadedIndicator:    make(chan struct{}),
	}
	inst, err := factory.CreateInstrumentation(ctx, e.PID, settings)
	if err != nil {
		return fmt.Errorf("failed to create instrumentation: %w", err)
	}

	runError := make(chan error)

	go func() {
		defer close(runError)
		err := inst.Run(ctx)
		runError <- err
	}()

	select {
	case <-settings.LoadedIndicator:
		m.Logger.Info("instrumentation loaded", "pid", e.PID)
		// successfully created and loaded instrumentation, track it by pid
		m.DetailsByPid[e.PID] = InstrumentationDetails{
			Inst: inst,
			Pod:  types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name},
		}
		// TODO: write instrumentation instance CR with healthy status
	case err := <-runError:
		if err != nil {
			m.Logger.Error(err, "failed to run instrumentation")
			// TODO: write instrumentation instance CR with error status
		}
	}

	return nil
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

func (m *Manager) languageSdk(pod *corev1.Pod, event detector.ProcessEvent) (common.ProgrammingLanguage, common.OtelSdk, error) {
	containerName, ok := event.ExecDetails.Environments[consts.OdigosEnvVarContainerName]
	if !ok {
		return common.UnknownProgrammingLanguage, common.OtelSdk{}, ErrContainerNameNotReported
	}

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
