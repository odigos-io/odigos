package instrumentation

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/go-logr/logr"

	"github.com/odigos-io/odigos/instrumentation/detector"
)

var (
	errNoInstrumentationFactory = errors.New("no ebpf factory found")
	errFailedToGetDetails       = errors.New("failed to get details for process event")
	errFailedToGetDistribution  = errors.New("failed to get otel distribution for details")
	errFailedToGetConfigGroup   = errors.New("failed to get config group")
)

// ConfigUpdate is used to send a configuration update request to the manager.
// The manager will apply the configuration to all instrumentations that match the config group.
type ConfigUpdate[configGroup ConfigGroup] map[configGroup]Config

type instrumentationDetails[processDetails ProcessDetails, configGroup ConfigGroup] struct {
	// we want to track the instrumentation even if it failed to load, to be able to report the error
	// and clean up the reporter resources once the process exits.
	// hence, this might be nil if the instrumentation failed to load.
	inst Instrumentation
	pd   processDetails
	cg   configGroup
}

type ManagerOptions[processDetails ProcessDetails, configGroup ConfigGroup] struct {
	Logger logr.Logger

	// Factories is a map of OTel distributions to their corresponding instrumentation factories.
	//
	// The manager will use this map to create new instrumentations based on the process event.
	// If a process event is received and the OTel distribution is not found in this map,
	// the manager will ignore the event.
	Factories map[OtelDistribution]Factory

	// Handler is used to resolve details, config group, OTel distribution and settings for the instrumentation
	// based on the process event.
	//
	// The handler is also used to report the instrumentation lifecycle events.
	Handler *Handler[processDetails, configGroup]

	// DetectorOptions is a list of options to configure the process detector.
	//
	// The process detector is used to trigger new instrumentation for new relevant processes,
	// and un-instrumenting processes once they exit.
	DetectorOptions []detector.DetectorOption

	// ConfigUpdates is a channel for receiving configuration updates.
	// The manager will apply the configuration to all instrumentations that match the config group.
	//
	// The caller is responsible for closing the channel once no more updates are expected.
	ConfigUpdates <-chan ConfigUpdate[configGroup]
}

// Manager is used to orchestrate the ebpf instrumentations lifecycle.
type Manager interface {
	// Run launches the manger.
	// It will block until the context is canceled.
	// It is an error to not cancel the context before the program exits, and may result in leaked resources.
	Run(ctx context.Context) error
}

type manager[processDetails ProcessDetails, configGroup ConfigGroup] struct {
	// channel for receiving process events,
	// used to detect new processes and process exits, and handle their instrumentation accordingly.
	procEvents <-chan detector.ProcessEvent
	detector   detector.Detector
	handler    *Handler[processDetails, configGroup]
	factories  map[OtelDistribution]Factory
	logger     logr.Logger

	// all the created instrumentations by pid,
	// this map is not concurrent safe, so it should be accessed only from the main event loop
	detailsByPid map[int]*instrumentationDetails[processDetails, configGroup]

	// instrumentations by workload, and aggregated by pid
	// this map is not concurrent safe, so it should be accessed only from the main event loop
	detailsByWorkload map[configGroup]map[int]*instrumentationDetails[processDetails, configGroup]

	configUpdates <-chan ConfigUpdate[configGroup]
}

func NewManager[processDetails ProcessDetails, configGroup ConfigGroup](options ManagerOptions[processDetails, configGroup]) (Manager, error) {
	handler := options.Handler
	if handler == nil {
		return nil, errors.New("handler is required for ebpf instrumentation manager")
	}

	if handler.Reporter == nil {
		return nil, errors.New("reporter is required for ebpf instrumentation manager")
	}

	if handler.ProcessDetailsResolver == nil {
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

	if options.ConfigUpdates == nil {
		return nil, errors.New("config updates channel is required for ebpf instrumentation manager")
	}

	logger := options.Logger
	procEvents := make(chan detector.ProcessEvent)
	detector, err := detector.NewDetector(procEvents, options.DetectorOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create process detector: %w", err)
	}

	return &manager[processDetails, configGroup]{
		procEvents:        procEvents,
		detector:          detector,
		handler:           handler,
		factories:         options.Factories,
		logger:            logger.WithName("ebpf-instrumentation-manager"),
		detailsByPid:      make(map[int]*instrumentationDetails[processDetails, configGroup]),
		detailsByWorkload: map[configGroup]map[int]*instrumentationDetails[processDetails, configGroup]{},
		configUpdates:     options.ConfigUpdates,
	}, nil
}

func (m *manager[ProcessDetails, ConfigGroup]) runEventLoop(ctx context.Context) {
	// main event loop for handling instrumentations
	for {
		select {
		case <-ctx.Done():
			m.logger.Info("stopping eBPF instrumentation manager")
			for pid, details := range m.detailsByPid {
				if details.inst == nil {
					continue
				}
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
				if err != nil && !errors.Is(err, errNoInstrumentationFactory) {
					m.logger.Error(err, "failed to handle process exec event")
				}
			case detector.ProcessExitEvent:
				m.cleanInstrumentation(ctx, e.PID)
			}
		case configUpdate := <-m.configUpdates:
			for configGroup, config := range configUpdate {
				err := m.applyInstrumentationConfigurationForSDK(ctx, configGroup, config)
				if err != nil {
					m.logger.Error(err, "failed to apply instrumentation configuration")
				}
			}
		}
	}
}

func (m *manager[ProcessDetails, ConfigGroup]) Run(ctx context.Context) error {
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

func (m *manager[ProcessDetails, ConfigGroup]) cleanInstrumentation(ctx context.Context, pid int) {
	details, found := m.detailsByPid[pid]
	if !found {
		m.logger.V(3).Info("no instrumentation found for exiting pid, nothing to clean", "pid", pid)
		return
	}

	m.logger.Info("cleaning instrumentation resources", "pid", pid, "process group details", details.pd)

	if details.inst != nil {
		err := details.inst.Close(ctx)
		if err != nil {
			m.logger.Error(err, "failed to close instrumentation")
		}
	}

	err := m.handler.Reporter.OnExit(ctx, pid, details.pd)
	if err != nil {
		m.logger.Error(err, "failed to report instrumentation exit")
	}

	m.stopTrackInstrumentation(pid)
}

func (m *manager[ProcessDetails, ConfigGroup]) handleProcessExecEvent(ctx context.Context, e detector.ProcessEvent) error {
	if details, found := m.detailsByPid[e.PID]; found && details.inst != nil {
		// this can happen if we have multiple exec events for the same pid (chain loading)
		// TODO: better handle this?
		// this can be done by first closing the existing instrumentation,
		// and then creating a new one
		m.logger.Info("received exec event for process id which is already instrumented with ebpf, skipping it", "pid", e.PID)
		return nil
	}

	pd, err := m.handler.ProcessDetailsResolver.Resolve(ctx, e)
	if err != nil {
		return errors.Join(err, errFailedToGetDetails)
	}

	otelDisto, err := m.handler.DistributionMatcher.Distribution(ctx, pd)
	if err != nil {
		return errors.Join(err, errFailedToGetDistribution)
	}

	configGroup, err := m.handler.ConfigGroupResolver.Resolve(ctx, pd, otelDisto)
	if err != nil {
		return errors.Join(err, errFailedToGetConfigGroup)
	}

	factory, found := m.factories[otelDisto]
	if !found {
		return errNoInstrumentationFactory
	}

	// Fetch initial settings for the instrumentation
	settings, err := m.handler.SettingsGetter.Settings(ctx, pd, otelDisto)
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
		err = m.handler.Reporter.OnInit(ctx, e.PID, err, pd)
		// TODO: should we return here the initialize error? or the handler error? or both?
		return err
	}

	loadErr := inst.Load(ctx)

	reporterErr := m.handler.Reporter.OnLoad(ctx, e.PID, loadErr, pd)
	if reporterErr != nil {
		m.logger.Error(reporterErr, "failed to report instrumentation load", "loaded", loadErr == nil, "pid", e.PID, "process group details", pd)
	}
	if loadErr != nil {
		// we need to track the instrumentation even if the load failed.
		// consider a reporter which writes a persistent record for a failed/successful load
		// we need to notify the reporter once that PID exits to clean up the resources - hence we track it.
		// saving the inst as nil marking the instrumentation failed to load, and is not valid to run/configure/close.
		m.startTrackInstrumentation(e.PID, nil, pd, configGroup)
		m.logger.Error(err, "failed to load instrumentation", "language", otelDisto.Language, "sdk", otelDisto.OtelSdk)
		// TODO: should we return here the load error? or the instance write error? or both?
		return err
	}

	m.startTrackInstrumentation(e.PID, inst, pd, configGroup)
	m.logger.Info("instrumentation loaded", "pid", e.PID, "process group details", pd)

	go func() {
		err := inst.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			reporterErr := m.handler.Reporter.OnRun(ctx, e.PID, err, pd)
			if reporterErr != nil {
				m.logger.Error(reporterErr, "failed to report instrumentation run")
			}
			m.logger.Error(err, "failed to run instrumentation")
		}
	}()

	return nil
}

func (m *manager[ProcessDetails, ConfigGroup]) startTrackInstrumentation(pid int, inst Instrumentation, processDetails ProcessDetails, configGroup ConfigGroup) {
	instDetails := &instrumentationDetails[ProcessDetails, ConfigGroup]{
		inst: inst,
		pd:   processDetails,
		cg:   configGroup,
	}
	m.detailsByPid[pid] = instDetails

	if _, found := m.detailsByWorkload[configGroup]; !found {
		// first instrumentation for this workload
		m.detailsByWorkload[configGroup] = map[int]*instrumentationDetails[ProcessDetails, ConfigGroup]{pid: instDetails}
	} else {
		m.detailsByWorkload[configGroup][pid] = instDetails
	}
}

func (m *manager[ProcessDetails, ConfigGroup]) stopTrackInstrumentation(pid int) {
	details, ok := m.detailsByPid[pid]
	if !ok {
		return
	}
	workloadConfigID := details.cg

	delete(m.detailsByPid, pid)
	delete(m.detailsByWorkload[workloadConfigID], pid)

	if len(m.detailsByWorkload[workloadConfigID]) == 0 {
		delete(m.detailsByWorkload, workloadConfigID)
	}
}

func (m *manager[ProcessDetails, ConfigGroup]) applyInstrumentationConfigurationForSDK(ctx context.Context, configGroup ConfigGroup, config Config) error {
	var err error

	configGroupInstrumentations, ok := m.detailsByWorkload[configGroup]
	if !ok {
		return nil
	}

	for _, instDetails := range configGroupInstrumentations {
		if instDetails.inst == nil {
			continue
		}
		m.logger.Info("applying configuration to instrumentation", "process group details", instDetails.pd, "configGroup", configGroup)
		applyErr := instDetails.inst.ApplyConfig(ctx, config)
		err = errors.Join(err, applyErr)
	}
	return err
}
