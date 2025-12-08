package instrumentation

import (
	"context"
	"errors"
	"fmt"
	"time"

	cilumebpf "github.com/cilium/ebpf"
	"go.opentelemetry.io/otel"
	"golang.org/x/sync/errgroup"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common/unixfd"
	"github.com/odigos-io/odigos/instrumentation/detector"
)

var (
	errNoInstrumentationFactory = errors.New("no ebpf factory found")
	errFailedToGetDetails       = errors.New("failed to get details for process event")
	errFailedToGetDistribution  = errors.New("failed to get otel distribution for details")
	errFailedToGetConfigGroup   = errors.New("failed to get config group")
	errFailedToGetProcessGroup  = errors.New("failed to get process group")
)

const (
	shutdownCleanupTimeout = 10 * time.Second
	otelMeterName          = "github.com/odigos.io/odigos/instrumentation"
)

var meter = otel.Meter(otelMeterName)

// ConfigUpdate is used to send a configuration update request to the manager.
// The manager will apply the configuration to all instrumentations that match the config group.
type ConfigUpdate[configGroup ConfigGroup] map[configGroup]Config

type InstrumentationRequest[processGroup ProcessGroup, configGroup ConfigGroup, processDetails ProcessDetails[processGroup, configGroup]] struct {
	ProcessDetailsByPid map[int]processDetails
}

type instrumentationDetails[processGroup ProcessGroup, configGroup ConfigGroup, processDetails ProcessDetails[processGroup, configGroup]] struct {
	// we want to track the instrumentation even if it failed to load, to be able to report the error
	// and clean up the reporter resources once the process exits.
	// hence, this might be nil if the instrumentation failed to load.
	inst Instrumentation
	pd   processDetails
	cg   configGroup
	pg   processGroup
}

type ManagerOptions[processGroup ProcessGroup, configGroup ConfigGroup, processDetails ProcessDetails[processGroup, configGroup]] struct {
	Logger logr.Logger

	// Factories is a map of Odigos Otel distribution names to their corresponding instrumentation factories.
	//
	// The manager will use this map to create new instrumentations based on the process event.
	// If a process event is received and the distribution name is not found in this map,
	// the manager will ignore the event.
	Factories map[string]Factory

	// Handler is used to resolve details, config group, OTel distribution and settings for the instrumentation
	// based on the process event.
	//
	// The handler is also used to report the instrumentation lifecycle events.
	Handler *Handler[processGroup, configGroup, processDetails]

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

	InstrumentationRequests <-chan InstrumentationRequest[processGroup, configGroup, processDetails]

	// TracesMap is the optional common eBPF map that will be used to send events from eBPF probes.
	TracesMap *cilumebpf.Map
}

// Manager is used to orchestrate the ebpf instrumentations lifecycle.
type Manager interface {
	// Run launches the manger.
	// It will block until the context is canceled.
	// It is an error to not cancel the context before the program exits, and may result in leaked resources.
	Run(ctx context.Context) error
}

type manager[processGroup ProcessGroup, configGroup ConfigGroup, processDetails ProcessDetails[processGroup, configGroup]] struct {
	// channel for receiving process events,
	// used to detect new processes and process exits, and handle their instrumentation accordingly.
	procEvents <-chan detector.ProcessEvent
	detector   detector.Detector
	handler    *Handler[processGroup, configGroup, processDetails]
	factories  map[string]Factory
	logger     logr.Logger

	// all the created instrumentations by pid,
	// this map is not concurrent safe, so it should be accessed only from the main event loop
	detailsByPid map[int]*instrumentationDetails[processGroup, configGroup, processDetails]

	// instrumentations by config group, and aggregated by pid
	// this map is not concurrent safe, so it should be accessed only from the main event loop
	detailsByConfigGroup map[configGroup]map[int]*instrumentationDetails[processGroup, configGroup, processDetails]

	// instrumentations by process group, and aggregated by pid
	// this map is not concurrent safe, so it should be accessed only from the main event loop
	detailsByProcessGroup map[processGroup]map[int]*instrumentationDetails[processGroup, configGroup, processDetails]

	configUpdates <-chan ConfigUpdate[configGroup]

	instrumentationRequests <-chan InstrumentationRequest[processGroup, configGroup, processDetails]

	metrics *managerMetrics

	tracesMap *cilumebpf.Map
}

func NewManager[processGroup ProcessGroup, configGroup ConfigGroup, processDetails ProcessDetails[processGroup, configGroup]](options ManagerOptions[processGroup, configGroup, processDetails]) (Manager, error) {
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

	if handler.SettingsGetter == nil {
		return nil, errors.New("settings getter is required for ebpf instrumentation manager")
	}

	if options.ConfigUpdates == nil {
		return nil, errors.New("config updates channel is required for ebpf instrumentation manager")
	}

	managerMetrics, err := newManagerMetrics(meter)
	if err != nil {
		return nil, fmt.Errorf("failed to create ebpf instrumentation manager metrics: %w", err)
	}

	logger := options.Logger
	procEvents := make(chan detector.ProcessEvent)
	detector, err := detector.NewDetector(procEvents, options.DetectorOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create process detector: %w", err)
	}

	return &manager[processGroup, configGroup, processDetails]{
		procEvents:              procEvents,
		detector:                detector,
		handler:                 handler,
		factories:               options.Factories,
		logger:                  logger.WithName("ebpf-instrumentation-manager"),
		detailsByPid:            make(map[int]*instrumentationDetails[processGroup, configGroup, processDetails]),
		detailsByConfigGroup:    map[configGroup]map[int]*instrumentationDetails[processGroup, configGroup, processDetails]{},
		detailsByProcessGroup:   map[processGroup]map[int]*instrumentationDetails[processGroup, configGroup, processDetails]{},
		configUpdates:           options.ConfigUpdates,
		instrumentationRequests: options.InstrumentationRequests,
		metrics:                 managerMetrics,
		tracesMap:               options.TracesMap,
	}, nil
}

func (m *manager[ProcessGroup, ConfigGroup, ProcessDetails]) runEventLoop(ctx context.Context) {
	// cleanup all instrumentations on shutdown
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownCleanupTimeout)
		defer cancel()

		for pid, details := range m.detailsByPid {
			select {
			case <-ctx.Done():
				m.logger.Error(ctx.Err(), "context canceled while cleaning up instrumentations before shutdown")
				return
			default:
				if details.inst == nil {
					continue
				}
				if err := details.inst.Close(ctx); err != nil {
					m.logger.Error(err, "failed to close instrumentation", "pid", pid)
				}
				if err := m.handler.Reporter.OnExit(ctx, pid, details.pd); err != nil {
					m.logger.Error(err, "failed to report instrumentation exit")
				}
			}
		}

		m.detailsByPid = nil
		m.detailsByConfigGroup = nil
		m.detailsByProcessGroup = nil
		m.logger.Info("all instrumentations cleaned up")
	}()

	// main event loop for handling instrumentations
	for {
		select {
		case <-ctx.Done():
			m.logger.Info("stopping eBPF instrumentation manager")
			return
		case e, ok := <-m.procEvents:
			if !ok {
				m.logger.Info("process events channel closed, stopping eBPF instrumentation manager")
				return
			}
			switch e.EventType {
			case detector.ProcessExecEvent, detector.ProcessForkEvent, detector.ProcessFileOpenEvent:
				m.logger.V(1).Info("detected new process", "pid", e.PID, "cmd", e.ExecDetails.CmdLine)
				err := m.tryInstrumentFromProcessEvent(ctx, e)
				if err != nil {
					m.handleInstrumentError(err)
				}
			case detector.ProcessExitEvent:
				m.cleanInstrumentation(ctx, e.PID)
			}
		case req, ok := <-m.instrumentationRequests:
			if !ok {
				m.logger.Info("instrumentation requests channel closed, stopping eBPF instrumentation manager")
				return
			}
			for pid, details := range req.ProcessDetailsByPid {
				// handle duplicate requests gracefully, this can happen
				// in environments where the requests are triggered by external systems such as k8s controllers
				if m.isInstrumented(pid) {
					continue
				}
				m.logger.Info("received explicit instrumentation request", "pid", pid)
				err := m.tryInstrument(ctx, details, pid)
				if err != nil {
					m.handleInstrumentError(err)
				}
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

func (m *manager[ProcessGroup, ConfigGroup, ProcessDetails]) handleInstrumentError(err error) {
	// ignore the error if no instrumentation factory is found,
	// as this is expected for some language and sdk combinations which don't have ebpf support.
	if errors.Is(err, errNoInstrumentationFactory) {
		return
	}

	// in cases where we detected a certain language for a container, but multiple processes are running in it,
	// only one or some of them are in the language we detected.
	if errors.Is(err, ErrProcessLanguageNotMatchesDistribution) {
		m.logger.Info("process language does not match the detected language for container, skipping instrumentation", "error", err)
		return
	}

	// fallback to log an error
	if err != nil {
		m.logger.Error(err, "failed to handle process exec event")
	}
}

func (m *manager[ProcessGroup, ConfigGroup, ProcessDetails]) Run(ctx context.Context) error {
	g, errCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return m.detector.Run(errCtx)
	})

	g.Go(func() error {
		m.runEventLoop(errCtx)
		return nil
	})

	g.Go(func() error {
		// Start the FD server
		server := &unixfd.Server{
			SocketPath: unixfd.DefaultSocketPath,
			Logger:     m.logger,
			FDProvider: func() int {
				return m.tracesMap.FD()
			},
		}

		// Run server in background to serve the map FD to relevant data collection client.
		// The server will continue running until odiglet shuts down, allowing collectors to reconnect after restarts
		// and ask for a new FD.
		if err := server.Run(ctx); err != nil {
			m.logger.Error(err, "unixfd server failed")
		}

		m.logger.Info("TracesMap created, FD server started",
			"socket", unixfd.DefaultSocketPath,
			"map_fd", m.tracesMap.FD())
		return nil
	})

	err := g.Wait()

	return err
}

func (m *manager[ProcessGroup, ConfigGroup, ProcessDetails]) cleanInstrumentation(ctx context.Context, pid int) {
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
		m.metrics.instrumentedProcesses.Add(ctx, -1)
	}

	err := m.handler.Reporter.OnExit(ctx, pid, details.pd)
	if err != nil {
		m.logger.Error(err, "failed to report instrumentation exit")
	}

	m.stopTrackInstrumentation(pid)
}

func (m *manager[ProcessGroup, ConfigGroup, ProcessDetails]) isInstrumented(pid int) bool {
	details, found := m.detailsByPid[pid]
	return found && details.inst != nil
}

func (m *manager[ProcessGroup, ConfigGroup, ProcessDetails]) tryInstrumentFromProcessEvent(ctx context.Context, e detector.ProcessEvent) error {
	pd, err := m.handler.ProcessDetailsResolver.Resolve(ctx, e)
	if err != nil {
		return errors.Join(err, errFailedToGetDetails)
	}

	return m.tryInstrument(ctx, pd, e.PID)
}

func (m *manager[ProcessGroup, ConfigGroup, ProcessDetails]) tryInstrument(ctx context.Context, pd ProcessDetails, pid int) error {
	if m.isInstrumented(pid) {
		// this can happen if we have multiple exec events for the same pid (chain loading)
		// TODO: better handle this?
		// this can be done by first closing the existing instrumentation,
		// and then creating a new one
		m.logger.Info("received exec event for process id which is already instrumented with ebpf, skipping it", "pid", pid, "process details", pd.String())
		return nil
	}

	otelDistro, err := pd.Distribution(ctx)
	if err != nil {
		return errors.Join(err, errFailedToGetDistribution)
	}

	configGroup, err := pd.ConfigGroup(ctx)
	if err != nil {
		return errors.Join(err, errFailedToGetConfigGroup)
	}

	processGroup, err := pd.ProcessGroup(ctx)
	if err != nil {
		return errors.Join(err, errFailedToGetProcessGroup)
	}

	factory, found := m.factories[otelDistro.Name]
	if !found {
		return errNoInstrumentationFactory
	}

	// Fetch initial settings for the instrumentation
	settings, err := m.handler.SettingsGetter.Settings(ctx, m.logger, pd, otelDistro)
	if err != nil {
		// for k8s instrumentation config CR will be queried to get the settings
		// we should always have config for this event.
		// if missing, it means that either:
		// - the config will be generated later due to reconciliation timing in instrumentor
		// - just got deleted and the pod (and the process) will go down soon
		// TODO: sync reconcilers so inst config is guaranteed be created before the webhook is enabled
		//
		m.logger.Info("failed to get initial settings for instrumentation", "language", otelDistro.Language, "distroName", otelDistro.Name, "error", err)
		// return nil
	}

	settings.TracesMap = ReaderMap{
		Map:            m.tracesMap,
		ExternalReader: true,
	}

	inst, initErr := factory.CreateInstrumentation(ctx, pid, settings)
	reporterErr := m.handler.Reporter.OnInit(ctx, pid, initErr, pd)
	if reporterErr != nil {
		m.logger.Error(reporterErr, "failed to report instrumentation init", "initialized", initErr == nil, "pid", pid, "process group details", pd)
	}
	if initErr != nil {
		// we need to track the instrumentation even if the initialization failed.
		// consider a reporter which writes a persistent record for a failed/successful init
		// we need to notify the reporter once that PID exits to clean up the resources - hence we track it.
		m.startTrackInstrumentation(pid, nil, pd, processGroup, configGroup)
		m.logger.Error(err, "failed to initialize instrumentation", "language", otelDistro.Language, "distroName", otelDistro.Name)
		m.metrics.failedInstrumentations.Add(ctx, 1)
		// TODO: should we return here the initialize error? or the handler error? or both?
		return initErr
	}

	status, loadErr := inst.Load(ctx)
	reporterErr = m.handler.Reporter.OnLoad(ctx, pid, loadErr, pd, status)
	if reporterErr != nil {
		m.logger.Error(reporterErr, "failed to report instrumentation load", "loaded", loadErr == nil, "pid", pid, "process group details", pd)
	}
	if loadErr != nil {
		// we need to track the instrumentation even if the load failed.
		// consider a reporter which writes a persistent record for a failed/successful load
		// we need to notify the reporter once that PID exits to clean up the resources - hence we track it.
		// saving the inst as nil marking the instrumentation failed to load, and is not valid to run/configure/close.
		m.startTrackInstrumentation(pid, nil, pd, processGroup, configGroup)
		m.logger.Error(err, "failed to load instrumentation", "language", otelDistro.Language, "distroName", otelDistro.Name)
		m.metrics.failedInstrumentations.Add(ctx, 1)
		// TODO: should we return here the load error? or the instance write error? or both?
		return loadErr
	}

	m.startTrackInstrumentation(pid, inst, pd, processGroup, configGroup)
	m.logger.Info("instrumentation loaded", "pid", pid, "process group details", pd)
	m.metrics.instrumentedProcesses.Add(ctx, 1)

	go func() {
		err := inst.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			reporterErr := m.handler.Reporter.OnRun(ctx, pid, err, pd)
			if reporterErr != nil {
				m.logger.Error(reporterErr, "failed to report instrumentation run")
			}
			m.logger.Error(err, "failed to run instrumentation")
		}
	}()

	return nil
}

func (m *manager[ProcessGroup, ConfigGroup, ProcessDetails]) startTrackInstrumentation(pid int, inst Instrumentation, processDetails ProcessDetails, processGroup ProcessGroup, configGroup ConfigGroup) {
	instDetails := &instrumentationDetails[ProcessGroup, ConfigGroup, ProcessDetails]{
		inst: inst,
		pd:   processDetails,
		cg:   configGroup,
		pg:   processGroup,
	}
	m.detailsByPid[pid] = instDetails

	if _, found := m.detailsByConfigGroup[configGroup]; !found {
		// first instrumentation for this workload
		m.detailsByConfigGroup[configGroup] = map[int]*instrumentationDetails[ProcessGroup, ConfigGroup, ProcessDetails]{pid: instDetails}
	} else {
		m.detailsByConfigGroup[configGroup][pid] = instDetails
	}

	if _, found := m.detailsByProcessGroup[processGroup]; !found {
		// first instrumentation for this workload
		m.detailsByProcessGroup[processGroup] = map[int]*instrumentationDetails[ProcessGroup, ConfigGroup, ProcessDetails]{pid: instDetails}
	} else {
		m.detailsByProcessGroup[processGroup][pid] = instDetails
	}
}

func (m *manager[ProcessGroup, ConfigGroup, ProcessDetails]) stopTrackInstrumentation(pid int) {
	details, ok := m.detailsByPid[pid]
	if !ok {
		return
	}
	cg := details.cg
	pg := details.pg

	delete(m.detailsByPid, pid)
	delete(m.detailsByConfigGroup[cg], pid)
	delete(m.detailsByProcessGroup[pg], pid)

	if len(m.detailsByConfigGroup[cg]) == 0 {
		delete(m.detailsByConfigGroup, cg)
	}

	if len(m.detailsByProcessGroup[pg]) == 0 {
		delete(m.detailsByProcessGroup, pg)
	}
}

func (m *manager[ProcessGroup, ConfigGroup, ProcessDetails]) applyInstrumentationConfigurationForSDK(ctx context.Context, configGroup ConfigGroup, config Config) error {
	var err error

	configGroupInstrumentations, ok := m.detailsByConfigGroup[configGroup]
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
