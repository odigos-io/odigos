package instrumentation

import (
	"context"
	"errors"
	"fmt"
	"time"

	cilumebpf "github.com/cilium/ebpf"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"golang.org/x/sync/errgroup"

	"github.com/odigos-io/odigos/common/unixfd"
	"github.com/odigos-io/odigos/distros/distro"
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

// Request is used to send an instrumentation, un-instrumentation, or retry-failed request to
// the manager.
//
// For instrumentation requests, set Instrument=true and populate ProcessDetailsByPid with the
// details of each process to instrument. For un-instrumentation requests, set Instrument=false
// and populate ProcessGroup to un-instrument all processes that match it (the manager keeps an
// index of instrumented processes by process group to make this efficient).
//
// For retry-failed requests, set RetryDistros to a non-nil slice; the manager will then iterate
// over its tracked instrumentations and retry any whose previous initialize/load failed (i.e.
// inst == nil) AND whose OTel distribution name matches one of the supplied values. A non-nil
// empty slice retries every failed instrumentation regardless of distribution. When
// RetryDistros is non-nil the Instrument / ProcessDetailsByPid / ProcessGroup fields are
// ignored.
type Request[processGroup ProcessGroup, configGroup ConfigGroup, processDetails ProcessDetails[processGroup, configGroup]] struct {
	Instrument          bool
	ProcessDetailsByPid map[int]processDetails
	ProcessGroup        processGroup
	RetryDistros        []string
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

	// InstrumentationRequests is a channel for receiving explicit instrumentation, un-
	// instrumentation, or retry-failed requests. See the Request docs for the encoding of each
	// request type. The same channel carries all three so callers (e.g. enterprise odiglet) can
	// piggy-back retry signals on the existing pipeline without plumbing a second channel.
	InstrumentationRequests <-chan Request[processGroup, configGroup, processDetails]

	// TracesMap is the optional common eBPF map that will be used to send events from eBPF probes.
	TracesMap *cilumebpf.Map

	// MetricsMap is the optional common eBPF map that is used to read metrics per Java process at each interval.
	MetricsMap *cilumebpf.Map

	// MetricsAttributesMap is the optional eBPF Hash map for UUID -> packed resource attributes.
	// Used alongside MetricsMap to store resource attributes separately from the metrics hash key.
	MetricsAttributesMap *cilumebpf.Map

	// Logger is optional. When set, the manager uses it; otherwise it uses commonlogger.LoggerCompat().With("subsystem", "ebpfmanager").
	Logger *commonlogger.OdigosLogger

	// LogsMap is the optional common eBPF map that will be used to send log events from eBPF probes.
	LogsMap *cilumebpf.Map

	// LogsAttrSubscribe streams per-process resource attributes over the logs unix socket.
	LogsAttrSubscribe func() (updates <-chan string, snapshot []string)
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
	logger     *commonlogger.OdigosLogger

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

	requests <-chan Request[processGroup, configGroup, processDetails]

	metrics *managerMetrics

	tracesMap            *cilumebpf.Map
	metricsMap           *cilumebpf.Map
	metricsAttributesMap *cilumebpf.Map
	logsMap              *cilumebpf.Map
	logsAttrSubscribe    func() (updates <-chan string, snapshot []string)
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

	logger := commonlogger.LoggerCompat().With("subsystem", "ebpfmanager")
	if options.Logger != nil {
		logger = options.Logger
	}
	procEvents := make(chan detector.ProcessEvent)
	detector, err := detector.NewDetector(procEvents, options.DetectorOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create process detector: %w", err)
	}

	return &manager[processGroup, configGroup, processDetails]{
		procEvents:            procEvents,
		detector:              detector,
		handler:               handler,
		factories:             options.Factories,
		logger:                logger,
		detailsByPid:          make(map[int]*instrumentationDetails[processGroup, configGroup, processDetails]),
		detailsByConfigGroup:  map[configGroup]map[int]*instrumentationDetails[processGroup, configGroup, processDetails]{},
		detailsByProcessGroup: map[processGroup]map[int]*instrumentationDetails[processGroup, configGroup, processDetails]{},
		configUpdates:         options.ConfigUpdates,
		requests:              options.InstrumentationRequests,
		metrics:               managerMetrics,
		tracesMap:             options.TracesMap,
		metricsMap:            options.MetricsMap,
		metricsAttributesMap:  options.MetricsAttributesMap,
		logsMap:               options.LogsMap,
		logsAttrSubscribe:     options.LogsAttrSubscribe,
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
				m.logger.Error("context canceled while cleaning up instrumentations before shutdown", "err", ctx.Err())
				return
			default:
				if details.inst != nil {
					if err := details.inst.Close(ctx); err != nil {
						m.logger.Error("failed to close instrumentation", "err", err, "pid", pid)
					}
				}
				if err := m.handler.Reporter.OnExit(ctx, pid, details.pd); err != nil {
					m.logger.Error("failed to report instrumentation exit", "err", err)
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
				m.logger.Debug("detected new process", "pid", e.PID, "cmd", e.ExecDetails.CmdLine)
				err := m.tryInstrumentFromProcessEvent(ctx, e)
				if err != nil {
					m.handleInstrumentError(err)
				}
			case detector.ProcessExitEvent:
				m.cleanInstrumentation(ctx, e.PID)
			}
		case req, ok := <-m.requests:
			if !ok {
				m.logger.Info("instrumentation requests channel closed, stopping eBPF instrumentation manager")
				return
			}
			// A non-nil RetryDistros marks this request as a retry-failed signal rather than a
			// normal instrument / un-instrument request. We check it first so the rest of the
			// request fields can be left zero by retry senders.
			if req.RetryDistros != nil {
				m.retryFailedInstrumentationsForDistros(ctx, req.RetryDistros)
				continue
			}
			if req.Instrument {
				instrumentedPIDs := make([]int, len(req.ProcessDetailsByPid))
				for pid, details := range req.ProcessDetailsByPid {
					// handle duplicate requests gracefully, this can happen
					// in environments where the requests are triggered by external systems such as k8s controllers
					if m.isInstrumented(pid) {
						continue
					}
					m.logger.Info("received explicit instrumentation request", "process details", details, "pid", pid)
					err := m.tryInstrument(ctx, details, pid)
					if err != nil {
						m.handleInstrumentError(err)
					} else {
						instrumentedPIDs = append(instrumentedPIDs, pid)
					}
				}
				// let the detector know that we are interested to get events for the instrumented processes
				// specifically, we want to be notified once these processes exit, so we can clean the instrumentation resources.
				m.detector.TrackProcesses(instrumentedPIDs)
			} else {
				// for un-instrumentation requests, we find all instrumentations that match the process group
				// and clean them up.
				procs, ok := m.detailsByProcessGroup[req.ProcessGroup]
				if !ok {
					continue
				}
				m.logger.Info("received explicit un-instrumentation request", "process group", req.ProcessGroup, "numPIDs", len(procs))
				for pid := range procs {
					m.cleanInstrumentation(ctx, pid)
				}
				// we could add a detector.UntrackProcesses call here, for now this is not necessary
				// reasoning to add it in the future might be to save resources in the detector
				// we might get exit events for already un-instrumented processes, which is a no-op.
			}
		case configUpdate := <-m.configUpdates:
			for configGroup, config := range configUpdate {
				err := m.applyInstrumentationConfigurationForSDK(ctx, configGroup, config)
				if err != nil {
					m.logger.Error("failed to apply instrumentation configuration", "err", err)
				}
			}
		}
	}
}

// retryFailedInstrumentationsForDistros re-attempts instrumentation for every entry in
// detailsByPid whose previous initialize/load attempt failed (inst == nil). When distroFilter is
// non-empty, only failed entries whose OTel distribution name matches one of the supplied values
// are retried; an empty (but non-nil) filter retries every failed entry regardless of
// distribution.
//
// We iterate the full detailsByPid map here because there is typically a small number of tracked
// processes per node and maintaining a parallel "failed" sub-index doesn't pay for itself. The
// distribution name comes from pd.Distribution(ctx), which is already memoized after the first
// successful resolution that happened when we tracked the entry.
func (m *manager[ProcessGroup, ConfigGroup, ProcessDetails]) retryFailedInstrumentationsForDistros(ctx context.Context, distroFilter []string) {
	wanted := make(map[string]struct{}, len(distroFilter))
	for _, name := range distroFilter {
		wanted[name] = struct{}{}
	}

	// Collect the PIDs to retry into a snapshot first; tryInstrument re-enters
	// startTrackInstrumentation, which mutates detailsByPid for the same pid, and we don't want
	// to modify the map while ranging over it.
	type retryEntry struct {
		pid int
		pd  ProcessDetails
	}
	var toRetry []retryEntry

	for pid, details := range m.detailsByPid {
		if details.inst != nil {
			continue
		}
		if len(wanted) > 0 {
			distribution, err := details.pd.Distribution(ctx)
			if err != nil || distribution == nil {
				continue
			}
			if _, ok := wanted[distribution.Name]; !ok {
				continue
			}
		}
		toRetry = append(toRetry, retryEntry{pid: pid, pd: details.pd})
	}

	if len(toRetry) == 0 {
		return
	}

	m.logger.Info("retrying failed instrumentations", "count", len(toRetry), "distroFilter", distroFilter)
	retriedPIDs := make([]int, 0, len(toRetry))
	for _, e := range toRetry {
		if err := m.tryInstrument(ctx, e.pd, e.pid); err != nil {
			m.handleInstrumentError(err)
			continue
		}
		retriedPIDs = append(retriedPIDs, e.pid)
	}
	if len(retriedPIDs) > 0 {
		// Re-arm the detector for the successfully retried processes so we still get their exit
		// events for cleanup. TrackProcesses is idempotent for already-tracked PIDs.
		m.detector.TrackProcesses(retriedPIDs)
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
		m.logger.Debug("process language does not match the detected language for container, skipping instrumentation", "err", err)
		return
	}

	// fallback to log an error
	if err != nil {
		m.logger.Error("failed to handle process exec event", "err", err)
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
			Logger:     commonlogger.ToLogr(),
			TracesFDProvider: func() int {
				return m.tracesMap.FD()
			},
			MetricsFDsProvider: func() []int {
				var fds []int
				if m.metricsMap != nil {
					fds = append(fds, m.metricsMap.FD())
				}
				if m.metricsAttributesMap != nil {
					fds = append(fds, m.metricsAttributesMap.FD())
				}
				return fds
			},
			LogsFDsProvider: func() []int {
				if m.logsMap != nil {
					return []int{m.logsMap.FD()}
				}
				return nil
			},
			LogsAttrSubscribe: m.logsAttrSubscribe,
		}

		// Run server in background to serve the map FD to relevant data collection client.
		// The server will continue running until odiglet shuts down, allowing collectors to reconnect after restarts
		// and ask for a new FD.
		if err := server.Run(errCtx); err != nil {
			m.logger.Error("unixfd server failed", "err", err)
		}

		m.logger.Info("eBPF maps created, FD server started",
			"socket", unixfd.DefaultSocketPath,
			"traces_map_fd", m.tracesMap.FD())
		return nil
	})

	err := g.Wait()

	return err
}

func (m *manager[ProcessGroup, ConfigGroup, ProcessDetails]) metricsAttributeSet(distribution *distro.OtelDistro) attribute.Set {
	return attribute.NewSet(
		semconv.TelemetryDistroName(distribution.Name),
		semconv.TelemetrySDKLanguageKey.String(string(distribution.Language)),
	)
}

func (m *manager[ProcessGroup, ConfigGroup, ProcessDetails]) cleanInstrumentation(ctx context.Context, pid int) {
	details, found := m.detailsByPid[pid]
	if !found {
		m.logger.Debug("no instrumentation found for exiting pid, nothing to clean", "pid", pid)
		return
	}

	m.logger.Info("cleaning instrumentation resources", "pid", pid, "process group details", details.pd)

	if details.inst != nil {
		err := details.inst.Close(ctx)
		if err != nil {
			m.logger.Error("failed to close instrumentation", "err", err)
		}
		distribution, _ := details.pd.Distribution(ctx)
		m.metrics.instrumentedProcesses.Add(ctx, -1, metric.WithAttributeSet(m.metricsAttributeSet(distribution)))
	}

	err := m.handler.Reporter.OnExit(ctx, pid, details.pd)
	if err != nil {
		m.logger.Error("failed to report instrumentation exit", "err", err)
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

	// Fetch initial settings for the instrumentation (SettingsGetter interface requires logr.Logger)
	settings, err := m.handler.SettingsGetter.Settings(ctx, commonlogger.ToLogr().WithName("ebpf-instrumentation-manager"), pd, otelDistro)
	if err != nil {
		// for k8s instrumentation config CR will be queried to get the settings
		// we should always have config for this event.
		// if missing, it means that either:
		// - the config will be generated later due to reconciliation timing in instrumentor
		// - just got deleted and the pod (and the process) will go down soon
		// TODO: sync reconcilers so inst config is guaranteed be created before the webhook is enabled
		//
		m.logger.Info("failed to get initial settings for instrumentation", "language", otelDistro.Language, "distroName", otelDistro.Name, "err", err)
		// return nil
	}

	settings.TracesMap = ReaderMap{
		Map:            m.tracesMap,
		ExternalReader: true,
	}

	settings.MetricsMap = MetricsMap{
		HashMapOfMaps: m.metricsMap,
		AttributesMap: m.metricsAttributesMap,
	}

	settings.LogsMap = ReaderMap{
		Map:            m.logsMap,
		ExternalReader: true,
	}

	inst, initErr := factory.CreateInstrumentation(ctx, pid, settings)
	reporterErr := m.handler.Reporter.OnInit(ctx, pid, initErr, pd)
	if reporterErr != nil {
		m.logger.Error("failed to report instrumentation init", "err", reporterErr, "initialized", initErr == nil, "pid", pid, "process group details", pd)
	}
	if initErr != nil {
		// we need to track the instrumentation even if the initialization failed.
		// consider a reporter which writes a persistent record for a failed/successful init
		// we need to notify the reporter once that PID exits to clean up the resources - hence we track it.
		m.startTrackInstrumentation(ctx, pid, nil, pd, processGroup, configGroup, otelDistro)
		m.logger.Error("failed to initialize instrumentation", "err", initErr, "language", otelDistro.Language, "distroName", otelDistro.Name)
		// TODO: should we return here the initialize error? or the handler error? or both?
		return initErr
	}

	status, loadErr := inst.Load(ctx)
	reporterErr = m.handler.Reporter.OnLoad(ctx, pid, loadErr, pd, status)
	if reporterErr != nil {
		m.logger.Error("failed to report instrumentation load", "err", reporterErr, "loaded", loadErr == nil, "pid", pid, "process group details", pd)
	}
	if loadErr != nil {
		// we need to track the instrumentation even if the load failed.
		// consider a reporter which writes a persistent record for a failed/successful load
		// we need to notify the reporter once that PID exits to clean up the resources - hence we track it.
		// saving the inst as nil marking the instrumentation failed to load, and is not valid to run/configure/close.
		m.startTrackInstrumentation(ctx, pid, nil, pd, processGroup, configGroup, otelDistro)
		m.logger.Error("failed to load instrumentation", "err", loadErr, "language", otelDistro.Language, "distroName", otelDistro.Name)
		// TODO: should we return here the load error? or the instance write error? or both?
		return loadErr
	}

	m.startTrackInstrumentation(ctx, pid, inst, pd, processGroup, configGroup, otelDistro)
	m.logger.Info("instrumentation loaded", "pid", pid, "process group details", pd)

	go func() {
		err := inst.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			reporterErr := m.handler.Reporter.OnRun(ctx, pid, err, pd)
			if reporterErr != nil {
				m.logger.Error("failed to report instrumentation run", "err", reporterErr)
			}
			m.logger.Error("failed to run instrumentation", "err", err)
		}
	}()

	return nil
}

func (m *manager[ProcessGroup, ConfigGroup, ProcessDetails]) startTrackInstrumentation(
	ctx context.Context,
	pid int,
	inst Instrumentation,
	processDetails ProcessDetails,
	processGroup ProcessGroup,
	configGroup ConfigGroup,
	distribution *distro.OtelDistro,
) {
	prevDetails, hadPrev := m.detailsByPid[pid]
	prevHadInst := hadPrev && prevDetails.inst != nil

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

	metricAttributeSet := m.metricsAttributeSet(distribution)
	switch {
	case inst == nil && !hadPrev:
		// First time we are tracking this pid and the attempt failed; count it once.
		m.metrics.failedInstrumentations.Add(ctx, 1, metric.WithAttributeSet(metricAttributeSet))
	case inst != nil && !prevHadInst:
		// Transition from "not instrumented" (either never seen or previously failed) to
		// "instrumented". failedInstrumentations is a monotonic counter so we don't decrement
		// it; we just record the successful instrumentation.
		m.metrics.instrumentedProcesses.Add(ctx, 1, metric.WithAttributeSet(metricAttributeSet))
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
