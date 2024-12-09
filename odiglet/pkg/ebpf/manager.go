package ebpf

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/container"
	instance "github.com/odigos-io/odigos/k8sutils/pkg/instrumentation_instance"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
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
	ErrNoInstrumentationFactory = errors.New("no ebpf factory found")
)

const (
	configUpdatesBufferSize = 10
)

type errRequiredEnvVarNotFound struct {
	envVarName string
}

func (e *errRequiredEnvVarNotFound) Error() string {
	return fmt.Sprintf("required environment variable not found: %s", e.envVarName)
}

var _ error = &errRequiredEnvVarNotFound{}

var (
	errContainerNameNotReported = &errRequiredEnvVarNotFound{envVarName: consts.OdigosEnvVarContainerName}
	errPodNameNotReported       = &errRequiredEnvVarNotFound{envVarName: consts.OdigosEnvVarPodName}
	errPodNameSpaceNotReported  = &errRequiredEnvVarNotFound{envVarName: consts.OdigosEnvVarNamespace}
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

type ConfigUpdate struct {
	PodWorkload workload.PodWorkload
	Config      *odigosv1.InstrumentationConfig
}

type instrumentationDetails struct {
	inst     Instrumentation
	pod      types.NamespacedName
	configID workloadConfigID
}

// workloadConfigID is used to identify a workload and its language for configuration updates
type workloadConfigID struct {
	podWorkload workload.PodWorkload
	lang        common.ProgrammingLanguage
}

type Manager struct {
	// channel for receiving process events,
	// used to detect new processes and process exits, and handle their instrumentation accordingly.
	procEvents <-chan detector.ProcessEvent
	detector   *detector.Detector
	client     client.Client
	factories  map[OtelDistribution]Factory
	logger     logr.Logger

	// all the active instrumentations by pid,
	// this map is not concurrent safe, so it should be accessed only from the main event loop
	detailsByPid map[int]*instrumentationDetails

	// active instrumentations by workload, and aggregated by pid
	// this map is not concurrent safe, so it should be accessed only from the main event loop
	detailsByWorkload map[workloadConfigID]map[int]*instrumentationDetails

	configUpdates chan ConfigUpdate
}

func NewManager(client client.Client, logger logr.Logger, factories map[OtelDistribution]Factory) (*Manager, error) {
	procEvents := make(chan detector.ProcessEvent)
	detector, err := detector.NewK8SProcDetector(context.Background(), logger, procEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to create process detector: %w", err)
	}

	return &Manager{
		procEvents:        procEvents,
		detector:          detector,
		client:            client,
		factories:         factories,
		logger:            logger.WithName("ebpf-instrumentation-manager"),
		detailsByPid:      make(map[int]*instrumentationDetails),
		detailsByWorkload: map[workloadConfigID]map[int]*instrumentationDetails{},
		configUpdates:     make(chan ConfigUpdate, configUpdatesBufferSize),
	}, nil
}

// ConfigUpdates returns a channel for receiving configuration updates for instrumentations
// sending on the channel will add an event to the main event loop to apply the configuration.
// closing this channel is in the responsibility of the caller.
func (m *Manager) ConfigUpdates() chan<- ConfigUpdate {
	return m.configUpdates
}

func (m *Manager) runEventLoop(ctx context.Context) {
	// main event loop for handling instrumentations
	for {
		select {
		case <-ctx.Done():
			m.logger.Info("stopping Odiglet instrumentation manager")
			for pid, details := range m.detailsByPid {
				err := details.inst.Close(ctx)
				if err != nil {
					m.logger.Error(err, "failed to close instrumentation", "pid", pid)
				}
				// probably shouldn't remove instrumentation instance here
				// as this flow is happening when Odiglet is shutting down
			}
			m.detailsByPid = nil
			m.detailsByWorkload = nil
			return
		case e := <-m.procEvents:
			switch e.EventType {
			case detector.ProcessExecEvent:
				m.logger.V(1).Info("detected new process", "pid", e.PID, "cmd", e.ExecDetails.CmdLine)
				err := m.handleProcessExecEvent(ctx, e)
				// ignore the error if no instrumentation factory is found,
				// as this is expected for some language and sdk combinations
				if err != nil && !errors.Is(err, ErrNoInstrumentationFactory) {
					m.logger.Error(err, "failed to handle process exec event")
				}
			case detector.ProcessExitEvent:
				m.cleanInstrumentation(ctx, e.PID)
			}
		case configUpdate := <-m.configUpdates:
			if configUpdate.Config == nil {
				m.logger.Info("received nil config update, skipping")
				break
			}
			for _, sdkConfig := range configUpdate.Config.Spec.SdkConfigs {
				err := m.applyInstrumentationConfigurationForSDK(ctx, configUpdate.PodWorkload, &sdkConfig)
				if err != nil {
					m.logger.Error(err, "failed to apply instrumentation configuration")
				}
			}
		}
	}
}

func (m *Manager) Run(ctx context.Context) error {
	g, errCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return m.detector.Run(errCtx)
	})

	g.Go(func() error {
		m.runEventLoop(errCtx)
		return nil
	})

	err := g.Wait()
	return err
}

func (m *Manager) cleanInstrumentation(ctx context.Context, pid int) {
	details, found := m.detailsByPid[pid]
	if !found {
		m.logger.V(3).Info("no instrumentation found for exiting pid, nothing to clean", "pid", pid)
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

	m.stopTrackInstrumentation(pid)
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
	pod, err := m.podFromProcEvent(ctx, e)
	if err != nil {
		return fmt.Errorf("failed to get pod from process event: %w", err)
	}

	containerName, found := containerNameFromProcEvent(e)
	if !found {
		return errContainerNameNotReported
	}

	// get the language and sdk for this process event
	// based on the pod spec and the container name from the process event
	// TODO: We should have all the required information in the process event
	// to determine the language - hence in the future we can improve this
	lang, sdk, err := odgiosK8s.LanguageSdkFromPodContainer(pod, containerName)
	if err != nil {
		return fmt.Errorf("failed to get language and sdk: %w", err)
	}

	factory, found := m.factories[OtelDistribution{Language: lang, OtelSdk: sdk}]
	if !found {
		return ErrNoInstrumentationFactory
	}

	podWorkload, err := workloadUtils.PodWorkloadObjectOrError(ctx, pod)
	if err != nil {
		return fmt.Errorf("failed to find workload object from pod manifest owners references: %w", err)
	}

	// Fetch initial config based on the InstrumentationConfig CR
	sdkConfig, serviceName := m.instrumentationSDKConfig(ctx, podWorkload, lang, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name})
	// we should always have config for this event.
	// if missing, it means that either:
	// - the config will be generated later due to reconciliation timing in instrumentor
	// - just got deleted and the pod (and the process) will go down soon
	// TODO: sync reconcilers so inst config is guaranteed be created before the webhook is enabled
	//
	// if sdkConfig == nil {
	// 	m.Logger.Info("no sdk config found for language", "language", lang, "pod", pod.Name)
	// 	return nil
	// }

	OtelServiceName := serviceName

	if serviceName == "" {
		OtelServiceName = podWorkload.Name
	}

	settings := Settings{
		ServiceName: OtelServiceName,
		// TODO: add container name
		ResourceAttributes: utils.GetResourceAttributes(podWorkload, pod.Name),
		InitialConfig:      sdkConfig,
	}

	inst, err := factory.CreateInstrumentation(ctx, e.PID, settings)
	if err != nil {
		m.logger.Error(err, "failed to initialize instrumentation", "language", lang, "sdk", sdk)

		// write instrumentation instance CR with error status
		err = m.updateInstrumentationInstanceStatus(ctx, pod, containerName, podWorkload, e.PID, InstrumentationUnhealthy, FailedToInitialize, err.Error())
		// TODO: should we return here the initialize error? or the instance write error? or both?
		return err
	}

	err = inst.Load(ctx)
	if err != nil {
		m.logger.Error(err, "failed to load instrumentation", "language", lang, "sdk", sdk)

		// write instrumentation instance CR with error status
		err = m.updateInstrumentationInstanceStatus(ctx, pod, containerName, podWorkload, e.PID, InstrumentationUnhealthy, FailedToLoad, err.Error())
		// TODO: should we return here the load error? or the instance write error? or both?
		return err
	}

	m.startTrackInstrumentation(e.PID, lang, inst, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}, *podWorkload)

	m.logger.Info("instrumentation loaded", "pid", e.PID, "pod", pod.Name, "container", containerName, "language", lang, "sdk", sdk)

	// write instrumentation instance CR with success status
	msg := fmt.Sprintf("Successfully loaded eBPF probes to pod: %s container: %s", pod.Name, containerName)
	err = m.updateInstrumentationInstanceStatus(ctx, pod, containerName, podWorkload, e.PID, InstrumentationHealthy, LoadedSuccessfully, msg)
	if err != nil {
		m.logger.Error(err, "failed to update instrumentation instance for successful load")
	}

	go func() {
		err := inst.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			m.logger.Error(err, "failed to run instrumentation")
			err = m.updateInstrumentationInstanceStatus(ctx, pod, containerName, podWorkload, e.PID, InstrumentationUnhealthy, FailedToRun, err.Error())
			if err != nil {
				m.logger.Error(err, "failed to update instrumentation instance for failed instrumentation run")
			}
		}
	}()

	return nil
}

func (m *Manager) startTrackInstrumentation(pid int, lang common.ProgrammingLanguage, inst Instrumentation, pod types.NamespacedName, podWorkload workload.PodWorkload) {
	workloadConfigID := workloadConfigID{
		podWorkload: podWorkload,
		lang:        lang,
	}

	details := &instrumentationDetails{
		inst:     inst,
		pod:      pod,
		configID: workloadConfigID,
	}
	m.detailsByPid[pid] = details

	if _, found := m.detailsByWorkload[workloadConfigID]; !found {
		// first instrumentation for this workload
		m.detailsByWorkload[workloadConfigID] = map[int]*instrumentationDetails{pid: details}
	} else {
		m.detailsByWorkload[workloadConfigID][pid] = details
	}
}

func (m *Manager) stopTrackInstrumentation(pid int) {
	details, ok := m.detailsByPid[pid]
	if !ok {
		return
	}
	workloadConfigID := details.configID

	delete(m.detailsByPid, pid)
	delete(m.detailsByWorkload[workloadConfigID], pid)

	if len(m.detailsByWorkload[workloadConfigID]) == 0 {
		delete(m.detailsByWorkload, workloadConfigID)
	}
}

func (m *Manager) instrumentationSDKConfig(ctx context.Context, w *workloadUtils.PodWorkload, lang common.ProgrammingLanguage, podKey types.NamespacedName) (sdkConfig *odigosv1.SdkConfig, serviceName string) {
	instrumentationConfig := odigosv1.InstrumentationConfig{}
	instrumentationConfigKey := client.ObjectKey{
		Namespace: w.Namespace,
		Name:      workloadUtils.CalculateWorkloadRuntimeObjectName(w.Name, w.Kind),
	}
	if err := m.client.Get(ctx, instrumentationConfigKey, &instrumentationConfig); err != nil {
		// this can be valid when the instrumentation config is deleted and current pods will go down soon
		m.logger.Error(err, "failed to get initial instrumentation config for instrumented pod", "pod", podKey.Name, "namespace", podKey.Namespace)
		return nil, ""
	}

	for _, config := range instrumentationConfig.Spec.SdkConfigs {
		if config.Language == lang {
			return &config, instrumentationConfig.Spec.ServiceName
		}
	}
	return nil, instrumentationConfig.Spec.ServiceName
}

func (m *Manager) applyInstrumentationConfigurationForSDK(ctx context.Context, podWorkload workload.PodWorkload, sdkConfig *odigosv1.SdkConfig) error {
	var err error

	configID := workloadConfigID{
		podWorkload: podWorkload,
		lang:        sdkConfig.Language,
	}

	workloadInstrumentations, ok := m.detailsByWorkload[configID]
	if !ok {
		return nil
	}

	for _, instDetails := range workloadInstrumentations {
		m.logger.Info("applying configuration to instrumentation", "podWorkload", podWorkload, "pod", instDetails.pod, "language", sdkConfig.Language)
		applyErr := instDetails.inst.ApplyConfig(ctx, sdkConfig)
		err = errors.Join(err, applyErr)
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

func (m *Manager) podFromProcEvent(ctx context.Context, event detector.ProcessEvent) (*corev1.Pod, error) {
	eventEnvs := event.ExecDetails.Environments

	podName, ok := eventEnvs[consts.OdigosEnvVarPodName]
	if !ok {
		return nil, errPodNameNotReported
	}

	podNamespace, ok := eventEnvs[consts.OdigosEnvVarNamespace]
	if !ok {
		return nil, errPodNameSpaceNotReported
	}

	pod := corev1.Pod{}
	err := m.client.Get(ctx, client.ObjectKey{Namespace: podNamespace, Name: podName}, &pod)
	if err != nil {
		return nil, fmt.Errorf("error fetching pod object: %w", err)
	}

	return &pod, nil
}

func containerNameFromProcEvent(event detector.ProcessEvent) (string, bool) {
	containerName, ok := event.ExecDetails.Environments[consts.OdigosEnvVarContainerName]
	return containerName, ok
}
