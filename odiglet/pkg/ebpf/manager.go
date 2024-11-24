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

type configUpdate struct {
	workloadKey types.NamespacedName
	config      *odigosv1.InstrumentationConfig
}

type ConfigUpdateFunc func(ctx context.Context, workloadKey types.NamespacedName, config *odigosv1.InstrumentationConfig) error

type instrumentationDetails struct {
	inst              Instrumentation
	pod               types.NamespacedName
	lang              common.ProgrammingLanguage
	workloadName      string
	workloadNamespace string
}

type Manager struct {
	procEvents    <-chan detector.ProcessEvent
	client        client.Client
	factories     map[FactoryID]Factory
	logger        logr.Logger
	detailsByPid  map[int]instrumentationDetails
	configUpdates chan configUpdate

	done chan struct{}
	stop context.CancelFunc
}

func NewManager(client client.Client, logger logr.Logger, factories map[FactoryID]Factory, procEvents <-chan detector.ProcessEvent) *Manager {
	return &Manager{
		procEvents:    procEvents,
		client:        client,
		factories:     factories,
		logger:        logger.WithName("ebpf-instrumentation-manager"),
		detailsByPid:  make(map[int]instrumentationDetails),
		configUpdates: make(chan configUpdate),
		done:          make(chan struct{}),
	}
}

func (m *Manager) UpdateConfig(ctx context.Context, workloadKey types.NamespacedName, config *odigosv1.InstrumentationConfig) error {
	// send a config update event for the specified workload
	select {
	case m.configUpdates <- configUpdate{workloadKey: workloadKey, config: config}:
		return nil
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return errors.New("failed to update config of workload: timeout waiting for config update")
		}
		return ctx.Err()
	}
}

func (m *Manager) Run(ctx context.Context) {
	defer close(m.done)

	runLoopCtx, stop := context.WithCancel(ctx)
	m.stop = stop

	// main event loop for handling instrumentations
	for {
		select {
		case <-runLoopCtx.Done():
			m.logger.Info("stopping Odiglet instrumentation manager")
			for pid, details := range m.detailsByPid {
				err := details.inst.Close(ctx)
				if err != nil {
					m.logger.Error(err, "failed to close instrumentation", "pid", pid)
				}
				delete(m.detailsByPid, pid)
				// probably shouldn't remove instrumentation instance here
				// as this flow is happening when Odiglet is shutting down
			}
			return
		case e := <-m.procEvents:
			switch e.EventType {
			case detector.ProcessExecEvent:
				m.logger.Info("detected new process", "pid", e.PID, "cmd", e.ExecDetails.CmdLine)
				err := m.handleProcessExecEvent(runLoopCtx, e)
				// ignore the error if no instrumentation factory is found,
				// as this is expected for some language and sdk combinations
				if err != nil && !errors.Is(err, ErrNoInstrumentationFactory) {
					m.logger.Error(err, "failed to handle process exec event")
				}
			case detector.ProcessExitEvent:
				m.cleanInstrumentation(runLoopCtx, e.PID)
			}
		case configUpdate := <-m.configUpdates:
			if configUpdate.config == nil {
				m.logger.Info("received nil config update, skipping")
			}
			for _, sdkConfig := range configUpdate.config.Spec.SdkConfigs {
				err := m.applyInstrumentationConfigurationForSDK(runLoopCtx, configUpdate.workloadKey, &sdkConfig)
				if err != nil {
					m.logger.Error(err, "failed to apply instrumentation configuration")
				}
			}
		}
	}
}

func (m *Manager) Stop() {
	m.stop()
	<-m.done
}

func (m *Manager) cleanInstrumentation(ctx context.Context, pid int) {
	details, found := m.detailsByPid[pid]
	if !found {
		m.logger.V(3).Info("no instrumentation found for exiting pid", "pid", pid)
		return
	}

	m.logger.Info("cleaning instrumentation resources", "pid", pid)

	err := details.inst.Close(ctx)
	if err != nil {
		m.logger.Error(err, "failed to close instrumentation")
	}

	// remove instrumentation instance
	if err = m.client.Delete(ctx, &odigosv1.InstrumentationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.InstrumentationInstanceName(details.pod.Name, pid),
			Namespace: details.pod.Namespace,
		},
	}); err != nil && !apierrors.IsNotFound(err) {
		m.logger.Error(err, "error deleting instrumentation instance", "pod", details.pod.Name, "pid", pid)
	}

	delete(m.detailsByPid, pid)
}

func (m *Manager) handleProcessExecEvent(ctx context.Context, e detector.ProcessEvent) error {
	if _, found := m.detailsByPid[e.PID]; found {
		// this can happen if we have multiple exec events for the same pid (chain loading)
		// TODO: better handle this?
		// this can be done by first closing the existing instrumentation,
		// and then creating a new one
		m.logger.Info("received exec event for process id which is already instrumented with ebpf, skipping it", "pid", e.PID)
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

	factory, found := m.factories[FactoryID{Language: lang, OtelSdk: sdk}]
	if !found {
		return ErrNoInstrumentationFactory
	}

	workload, err := workloadUtils.PodWorkloadObject(ctx, pod)
	if err != nil {
		return fmt.Errorf("failed to get workload object: %w", err)
	}

	// Fetch initial config based on the InstrumentationConfig CR
	sdkConfig := m.instrumentationSDKConfig(ctx, workload, lang, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name})
	// we should always have config for this event.
	// if missing, it means that either:
	// - the config will be generated later due to reconciliation timing in instrumentor
	// - just go deleted and the pod (and the process) will go down soon
	// TODO: sync reconcilers so ins config is guaranteed be created before the webhook is enabled
	//
	// if sdkConfig == nil {
	// 	m.Logger.Info("no sdk config found for language", "language", lang, "pod", pod.Name)
	// 	return nil
	// }

	settings := Settings{
		// TODO: respect reported name annotation (if present) - to override the service name
		// refactor from opAmp code
		ServiceName: workload.Name,
		// TODO: add container name
		ResourceAttributes: utils.GetResourceAttributes(workload, pod.Name),
		InitialConfig:      sdkConfig,
	}

	inst, err := factory.CreateInstrumentation(ctx, e.PID, settings)
	if err != nil {
		m.logger.Error(err, "failed to initialize instrumentation", "language", lang, "sdk", sdk)

		// write instrumentation instance CR with error status
		err = m.updateInstrumentationInstanceStatus(ctx, pod, containerName, workload, e.PID, InstrumentationUnhealthy, FailedToInitialize, err.Error())
		// TODO: should we return here the initialize error? or the instance write error? or both?
		return err
	}

	err = inst.Load(ctx)
	if err != nil {
		m.logger.Error(err, "failed to load instrumentation", "language", lang, "sdk", sdk)

		// write instrumentation instance CR with error status
		err = m.updateInstrumentationInstanceStatus(ctx, pod, containerName, workload, e.PID, InstrumentationUnhealthy, FailedToLoad, err.Error())
		// TODO: should we return here the load error? or the instance write error? or both?
		return err
	}

	m.detailsByPid[e.PID] = instrumentationDetails{
		inst:              inst,
		pod:               types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name},
		lang:              lang,
		workloadName:      workload.Name,
		workloadNamespace: workload.Namespace,
	}

	m.logger.Info("instrumentation loaded", "pid", e.PID, "pod", pod.Name, "container", containerName, "language", lang, "sdk", sdk)

	// write instrumentation instance CR with success status
	msg := fmt.Sprintf("Successfully loaded eBPF probes to pod: %s container: %s", pod.Name, containerName)
	err = m.updateInstrumentationInstanceStatus(ctx, pod, containerName, workload, e.PID, InstrumentationHealthy, LoadedSuccessfully, msg)
	if err != nil {
		m.logger.Error(err, "failed to update instrumentation instance for successful load")
	}

	go func() {
		err := inst.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			m.logger.Error(err, "failed to run instrumentation")
			// these errors occur after the instrumentation is loaded
			// write instrumentation instance CR with error status
			err = m.updateInstrumentationInstanceStatus(ctx, pod, containerName, workload, e.PID, InstrumentationUnhealthy, FailedToRun, err.Error())
			if err != nil {
				m.logger.Error(err, "failed to update instrumentation instance for failed instrumentation run")
			}
		}
	}()

	return nil
}

func (m *Manager) instrumentationSDKConfig(ctx context.Context, w *workloadUtils.PodWorkload, lang common.ProgrammingLanguage, podKey types.NamespacedName) *odigosv1.SdkConfig {
	instrumentationConfig := odigosv1.InstrumentationConfig{}
	instrumentationConfigKey := client.ObjectKey{
		Namespace: w.Namespace,
		Name:      workloadUtils.CalculateWorkloadRuntimeObjectName(w.Name, w.Kind),
	}
	if err := m.client.Get(ctx, instrumentationConfigKey, &instrumentationConfig); err != nil {
		// this can be valid when the instrumentation config is deleted and current pods will go down soon
		m.logger.Error(err, "failed to get initial instrumentation config for instrumented pod", "pod", podKey.Name, "namespace", podKey.Namespace)
		return nil
	}
	for _, config := range instrumentationConfig.Spec.SdkConfigs {
		if config.Language == lang {
			return &config
		}
	}
	return nil
}

func (m *Manager) applyInstrumentationConfigurationForSDK(ctx context.Context, workloadKey types.NamespacedName, sdkConfig *odigosv1.SdkConfig) error {
	var err error
	for _, instDetails := range m.detailsByPid {
		if instDetails.workloadName != workloadKey.Name || instDetails.workloadNamespace != workloadKey.Namespace {
			continue
		}

		if instDetails.lang != sdkConfig.Language {
			continue
		}

		m.logger.Info("applying configuration to instrumentation", "workload", workloadKey, "pid", instDetails.inst, "pod", instDetails.pod)
		err = errors.Join(err, instDetails.inst.ApplyConfig(ctx, sdkConfig))
	}
	return err
}

func (m *Manager) updateInstrumentationInstanceStatus(ctx context.Context, pod *corev1.Pod, containerName string, w *workloadUtils.PodWorkload, pid int, health InstrumentationHealth, reason InstrumentationStatusReason, msg string) error {
	instrumentedAppName := workloadUtils.CalculateWorkloadRuntimeObjectName(w.Name, w.Kind)
	healthy := bool(health)
	return instance.UpdateInstrumentationInstanceStatus(ctx, pod, containerName, m.client, instrumentedAppName, pid, m.client.Scheme(),
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
	// TODO: pass context from outer function
	err := m.client.Get(context.Background(), client.ObjectKey{Namespace: podNamespace, Name: podName}, &pod)
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
