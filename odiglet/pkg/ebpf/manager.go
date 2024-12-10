package ebpf

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common/instrumentation/types"
	"github.com/odigos-io/odigos/odiglet/pkg/detector"
)

var (
	errNoInstrumentationFactory = errors.New("no ebpf factory found")
	errFailedToGetDetails       = errors.New("failed to get details for process event")
	errFailedToGetDistribution  = errors.New("failed to get otel distribution for details")
	errFailedToGetConfigGroup   = errors.New("failed to get config group")
)

const (
	configUpdatesBufferSize = 10
)

type ConfigUpdate[ConfigGroup types.ConfigGroup] map[ConfigGroup]types.Config

type instrumentationDetails[Details types.Details, ConfigGroup types.ConfigGroup] struct {
	inst     types.Instrumentation
	details  Details
	configID ConfigGroup
}

type Manager[Details types.Details, ConfigGroup types.ConfigGroup] struct {
	// channel for receiving process events,
	// used to detect new processes and process exits, and handle their instrumentation accordingly.
	procEvents <-chan types.ProcessEvent
	detector   *types.Detector
	handler    *types.Handler[Details, ConfigGroup]
	factories  map[types.OtelDistribution]types.Factory
	logger     logr.Logger

	// all the active instrumentations by pid,
	// this map is not concurrent safe, so it should be accessed only from the main event loop
	detailsByPid map[int]*instrumentationDetails[Details, ConfigGroup]

	// active instrumentations by workload, and aggregated by pid
	// this map is not concurrent safe, so it should be accessed only from the main event loop
	detailsByWorkload map[ConfigGroup]map[int]*instrumentationDetails[Details, ConfigGroup]

	configUpdates chan ConfigUpdate[ConfigGroup]
}

func NewManager[Details types.Details, ConfigGroup types.ConfigGroup](logger logr.Logger, factories map[types.OtelDistribution]types.Factory, handler *types.Handler[Details, ConfigGroup]) (*Manager[Details, ConfigGroup], error) {
	if handler == nil {
		return nil, errors.New("handler is required for ebpf instrumentation manager")
	}

	if handler.Reporter == nil {
		return nil, errors.New("reporter is required for ebpf instrumentation manager")
	}

	if handler.DetailsResolver == nil {
		return nil, errors.New("details resolver is required for ebpf instrumentation manager")
	}

	if handler.ConfigGroupResolver == nil {
		return nil, errors.New("config group resolver is required for ebpf instrumentation manager")
	}

	if handler.DistributionMatcher == nil {
		return nil, errors.New("distribution matcher is required for ebpf instrumentation manager")
	}

	if handler.SettingsGetter == nil {
		return nil, errors.New("settings getter is required for ebpf instrumentation manager")
	}

	procEvents := make(chan types.ProcessEvent)
	detector, err := detector.NewK8SProcDetector(context.Background(), logger, procEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to create process detector: %w", err)
	}

	return &Manager[Details, ConfigGroup]{
		procEvents:        procEvents,
		detector:          detector,
		handler:           handler,
		factories:         factories,
		logger:            logger.WithName("ebpf-instrumentation-manager"),
		detailsByPid:      make(map[int]*instrumentationDetails[Details, ConfigGroup]),
		detailsByWorkload: map[ConfigGroup]map[int]*instrumentationDetails[Details, ConfigGroup]{},
		configUpdates:     make(chan ConfigUpdate[ConfigGroup], configUpdatesBufferSize),
	}, nil
}

// ConfigUpdates returns a channel for receiving configuration updates for instrumentations
// sending on the channel will add an event to the main event loop to apply the configuration.
// closing this channel is in the responsibility of the caller.
func (m *Manager[Details, ConfigGroup]) ConfigUpdates() chan<- ConfigUpdate[ConfigGroup] {
	return m.configUpdates
}

func (m *Manager[Details, ConfigGroup]) runEventLoop(ctx context.Context) {
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
			case types.ProcessExecEvent:
				m.logger.V(1).Info("detected new process", "pid", e.PID, "cmd", e.ExecDetails.CmdLine)
				err := m.handleProcessExecEvent(ctx, e)
				// ignore the error if no instrumentation factory is found,
				// as this is expected for some language and sdk combinations
				if err != nil && !errors.Is(err, errNoInstrumentationFactory) {
					m.logger.Error(err, "failed to handle process exec event")
				}
			case types.ProcessExitEvent:
				m.cleanInstrumentation(ctx, e.PID)
			}
		case configUpdate := <-m.configUpdates:
			if len(configUpdate) == 0 {
				m.logger.Info("received empty config update, skipping")
				break
			}
			for configGroup, config := range configUpdate {
				err := m.applyInstrumentationConfigurationForSDK(ctx, configGroup, config)
				if err != nil {
					m.logger.Error(err, "failed to apply instrumentation configuration")
				}
			}
		}
	}
}

func (m *Manager[Details, ConfigGroup]) Run(ctx context.Context) error {
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

func (m *Manager[Details, ConfigGroup]) cleanInstrumentation(ctx context.Context, pid int) {
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

	err = m.handler.Reporter.OnExit(ctx, pid, details.details)
	if err != nil {
		m.logger.Error(err, "failed to report instrumentation exit")
	}

	m.stopTrackInstrumentation(pid)
}

func (m *Manager[Details, ConfigGroup]) handleProcessExecEvent(ctx context.Context, e types.ProcessEvent) error {
	if _, found := m.detailsByPid[e.PID]; found {
		// this can happen if we have multiple exec events for the same pid (chain loading)
		// TODO: better handle this?
		// this can be done by first closing the existing instrumentation,
		// and then creating a new one
		m.logger.Info("received exec event for process id which is already instrumented with ebpf, skipping it", "pid", e.PID)
		return nil
	}

	details, err := m.handler.DetailsResolver.Resolve(ctx, e)
	if err != nil {
		return errors.Join(err, errFailedToGetDetails)
	}

	otelDisto, err := m.handler.DistributionMatcher.Distribution(ctx, details)
	if err != nil {
		return errors.Join(err, errFailedToGetDistribution)
	}

	configGroup, err := m.handler.ConfigGroupResolver.Resolve(ctx, details, otelDisto)
	if err != nil {
		return errors.Join(err, errFailedToGetConfigGroup)
	}

	factory, found := m.factories[otelDisto]
	if !found {
		return errNoInstrumentationFactory
	}

	// Fetch initial config based on the InstrumentationConfig CR
	settings, err := m.handler.SettingsGetter.Settings(ctx, details, otelDisto)
	if err != nil {
		// for k8s instrumentation config CR will be queried to get the settings
		// we should always have config for this event.
		// if missing, it means that either:
		// - the config will be generated later due to reconciliation timing in instrumentor
		// - just got deleted and the pod (and the process) will go down soon
		// TODO: sync reconcilers so inst config is guaranteed be created before the webhook is enabled
		//
		m.logger.Info("failed to get initial settings for instrumentation", "language", otelDisto.Language, "sdk", otelDisto.OtelSdk, "error", err)
		// return nil
	}

	inst, err := factory.CreateInstrumentation(ctx, e.PID, settings)
	if err != nil {
		m.logger.Error(err, "failed to initialize instrumentation", "language", otelDisto.Language, "sdk", otelDisto.OtelSdk)
		err = m.handler.Reporter.OnInit(ctx, e.PID, err, details)
		// TODO: should we return here the initialize error? or the handler error? or both?
		return err
	}

	err = inst.Load(ctx)
	// call the reporter regardless of the load result - as we want to report the load status
	reporterErr := m.handler.Reporter.OnLoad(ctx, e.PID, err, details)
	if err != nil {
		m.logger.Error(err, "failed to load instrumentation", "language", otelDisto.Language, "sdk", otelDisto.OtelSdk)
		// TODO: should we return here the load error? or the instance write error? or both?
		return err
	}

	if reporterErr != nil {
		m.logger.Error(reporterErr, "failed to report instrumentation load")
	}

	m.startTrackInstrumentation(e.PID, inst, details, configGroup)

	m.logger.Info("instrumentation loaded", "pid", e.PID, "details", details)

	go func() {
		err := inst.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			reporterErr := m.handler.Reporter.OnRun(ctx, e.PID, err, details)
			if reporterErr != nil {
				m.logger.Error(reporterErr, "failed to report instrumentation run")
			}
			m.logger.Error(err, "failed to run instrumentation")
		}
	}()

	return nil
}

func (m *Manager[Details, ConfigGroup]) startTrackInstrumentation(pid int, inst types.Instrumentation, details Details, configGroup ConfigGroup) {
	instDetails := &instrumentationDetails[Details, ConfigGroup]{
		inst:     inst,
		details:  details,
		configID: configGroup,
	}
	m.detailsByPid[pid] = instDetails

	if _, found := m.detailsByWorkload[configGroup]; !found {
		// first instrumentation for this workload
		m.detailsByWorkload[configGroup] = map[int]*instrumentationDetails[Details, ConfigGroup]{pid: instDetails}
	} else {
		m.detailsByWorkload[configGroup][pid] = instDetails
	}
}

func (m *Manager[Details, ConfigGroup]) stopTrackInstrumentation(pid int) {
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

func (m *Manager[Details, ConfigGroup]) applyInstrumentationConfigurationForSDK(ctx context.Context, configGroup ConfigGroup, config types.Config) error {
	var err error

	configGroupInstrumentations, ok := m.detailsByWorkload[configGroup]
	if !ok {
		return nil
	}

	for _, instDetails := range configGroupInstrumentations {
		m.logger.Info("applying configuration to instrumentation", "details", instDetails.details, "configGroup", configGroup)
		applyErr := instDetails.inst.ApplyConfig(ctx, config)
		err = errors.Join(err, applyErr)
	}
	return err
}
