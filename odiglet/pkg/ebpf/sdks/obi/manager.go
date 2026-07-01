package obi

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/common/api/agentsignalconfig"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/instrumentation"

	"go.opentelemetry.io/obi/pkg/appolly/discover"
	obiconfig "go.opentelemetry.io/obi/pkg/config"
	"go.opentelemetry.io/obi/pkg/export"
	"go.opentelemetry.io/obi/pkg/export/instrumentations"
	"go.opentelemetry.io/obi/pkg/instrumenter"
	obipkg "go.opentelemetry.io/obi/pkg/obi"
)

// DistroName is the Odigos Otel distribution name for OBI trace instrumentation.
const DistroName = "opentelemetry-ebpf-instrumentation"

var _ instrumentation.Factory = (*Manager)(nil)

// Manager owns the shared OBI instrumenter and implements instrumentation.Factory for the OBI distro.
// Run waits until ctx is canceled, then stops the instrumenter. CreateInstrumentation handles OBI traces for the OBI distro.
// SyncMetrics attaches network/stats metrics for any instrumented process when enabled via InstrumentationRule metricsConfig.
//
// PID selection updates are not synchronized here. They are invoked from the instrumentation manager
// event loop (Load/Close and lifecycle callbacks), which processes one event at a time.
type Manager struct {
	logger *commonlogger.OdigosLogger
	obiCfg *obipkg.Config

	selector *discover.DynamicPIDSelector

	runCtx    context.Context
	runCancel context.CancelFunc
}

// NewManager creates a manager with a fresh dynamic PID selector.
func NewManager() *Manager {
	return &Manager{
		selector: discover.NewDynamicPIDSelector(),
		obiCfg:   obiConfigForOdigos(),
		logger:   commonlogger.LoggerCompat().With("subsystem", "opentelemetry-ebpf-instrumentation"),
	}
}

// CreateInstrumentation returns a handle for an OBI distro process (traces via Load/Close).
func (m *Manager) CreateInstrumentation(_ context.Context, pid int, _ instrumentation.Settings) (instrumentation.Instrumentation, error) {
	return &processInstrumentation{
		manager: m,
		pid:     pid,
	}, nil
}

// SyncMetrics enables or disables network and stats metrics for pid based on per-workload metrics configuration.
func (m *Manager) SyncMetrics(pid int, metrics *agentsignalconfig.AgentMetricsConfig) {
	if pid <= 0 || metrics == nil {
		m.removeMetricsPIDs(pid, true, true)
		m.maybeStopInstrumenter()
		return
	}

	networkMetricsEnabled := instrumentationrules.MetricSignalEnabled(metrics.NetworkMetrics)
	statsMetricsEnabled := instrumentationrules.MetricSignalEnabled(metrics.StatsMetrics)

	if !networkMetricsEnabled && !statsMetricsEnabled {
		m.removeMetricsPIDs(pid, true, true)
		m.maybeStopInstrumenter()
		return
	}

	m.ensureInstrumenterRunning()
	if networkMetricsEnabled {
		m.selector.NetworkMetrics().AddPIDs(uint32(pid))
	} else {
		m.selector.NetworkMetrics().RemovePIDs(uint32(pid))
	}
	if statsMetricsEnabled {
		m.selector.StatsMetrics().AddPIDs(uint32(pid))
	} else {
		m.selector.StatsMetrics().RemovePIDs(uint32(pid))
	}
}

// Run waits until ctx is canceled, then stops the OBI instrumenter.
func (m *Manager) Run(ctx context.Context) error {
	<-ctx.Done()
	m.stopInstrumenter()
	return ctx.Err()
}

type processInstrumentation struct {
	manager *Manager
	pid     int
}

func (p *processInstrumentation) Load(_ context.Context) (instrumentation.Status, error) {
	p.manager.ensureInstrumenterRunning()
	p.manager.selector.Traces().AddPIDs(uint32(p.pid))
	return instrumentation.Status{}, nil
}

func (p *processInstrumentation) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (p *processInstrumentation) Close(_ context.Context) error {
	p.manager.selector.Traces().RemovePIDs(uint32(p.pid))
	p.manager.maybeStopInstrumenter()
	return nil
}

func (p *processInstrumentation) ApplyConfig(context.Context, instrumentation.Config) error {
	return nil
}

func (m *Manager) removeMetricsPIDs(pid int, networkMetricsEnabled, statsMetricsEnabled bool) {
	if networkMetricsEnabled {
		m.selector.NetworkMetrics().RemovePIDs(uint32(pid))
	}
	if statsMetricsEnabled {
		m.selector.StatsMetrics().RemovePIDs(uint32(pid))
	}
}

func obiConfigForOdigos() *obipkg.Config {
	cfg := obipkg.DefaultConfig
	cfg.EBPF.ContextPropagation = obiconfig.ContextPropagationHeaders

	collectorEndpoint := fmt.Sprintf("http://localhost:%d", consts.OTLPPort)
	cfg.Traces.TracesEndpoint = collectorEndpoint
	cfg.OTELMetrics.MetricsEndpoint = collectorEndpoint

	cfg.Traces.Instrumentations = append(cfg.Traces.Instrumentations, instrumentations.InstrumentationDNS)

	cfg.Metrics.Features = export.FeatureNetwork | export.FeatureStats

	return &cfg
}

func (m *Manager) ensureInstrumenterRunning() {
	if m.runCancel != nil {
		return
	}

	runCtx, runCancel := context.WithCancel(context.Background())
	obiCfg := m.obiCfg
	m.runCtx = runCtx
	m.runCancel = runCancel

	go func() {
		err := instrumenter.Run(runCtx, obiCfg, instrumenter.WithDynamicPIDSelector(m.selector))
		if err != nil && runCtx.Err() == nil {
			m.logger.Error("OBI instrumenter exited with error", "err", err)
		}
	}()
}

func (m *Manager) maybeStopInstrumenter() {
	if m.runCancel == nil || m.hasAnySelectedPIDs() {
		return
	}
	m.stopInstrumenter()
}

func (m *Manager) stopInstrumenter() {
	if m.runCancel == nil {
		return
	}
	m.runCancel()
	m.runCancel = nil
	m.runCtx = nil
}

func (m *Manager) hasAnySelectedPIDs() bool {
	if _, ok := m.selector.Traces().GetPIDs(); ok {
		return true
	}
	if _, ok := m.selector.NetworkMetrics().GetPIDs(); ok {
		return true
	}
	if _, ok := m.selector.StatsMetrics().GetPIDs(); ok {
		return true
	}
	return false
}
