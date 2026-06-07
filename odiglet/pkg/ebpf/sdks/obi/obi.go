package obi

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/instrumentation"

	"go.opentelemetry.io/obi/pkg/appolly/discover"
	"go.opentelemetry.io/obi/pkg/config"
	"go.opentelemetry.io/obi/pkg/export"
	"go.opentelemetry.io/obi/pkg/export/attributes"
	"go.opentelemetry.io/obi/pkg/instrumenter"
	"go.opentelemetry.io/obi/pkg/obi"
)

// NetworkPrometheusPort is the Prometheus port OBI exposes node-wide network-flow + TCP-stats
// metrics on, scraped in-process by the network-metrics runnable's resolver. App traces still
// export via OTLP (cfg.Traces); this endpoint carries only the NetO11y/StatsO11y pillars.
const NetworkPrometheusPort = 8999

// OBIInstrumentationFactory creates instrumentations that add/remove PIDs on a shared
// DynamicPIDSelector while a single OBI instrumenter runs in the background.
// Requires OBI with DynamicPIDSelector support (e.g. go.opentelemetry.io/obi from main after PR 1388).
type OBIInstrumentationFactory struct {
	logger       *commonlogger.OdigosLogger
	obiCtx       context.Context
	obiCtxCancel context.CancelFunc

	selector *discover.DynamicPIDSelector
	obiCfg   *obi.Config
}

// NewOBIInstrumentationFactory returns a factory that uses the OBI SDK with a dynamic PID selector.
// The OBI instrumenter starts on first CreateInstrumentation; PIDs are added/removed via the selector.
func NewOBIInstrumentationFactory() *OBIInstrumentationFactory {
	return &OBIInstrumentationFactory{
		selector: discover.NewDynamicPIDSelector(),
		obiCfg:   obiConfigForOdigos(),
		logger:   commonlogger.LoggerCompat().With("subsystem", "opentelemetry-ebpf-instrumentation"),
	}
}

// obiConfigForOdigos returns an OBI config that runs App O11y (traces) AND, on the SAME OBI
// instance, the node-wide Net O11y + Stats O11y pillars. OBI runs all enabled pillars as
// concurrent pipelines (pkg/instrumenter: setupAppO11y + setupNetO11y in one errgroup), so a
// single instrumenter serves both — no second OBI instance, no extra container. App PIDs are
// still supplied dynamically via the DynamicPIDSelector; netolly is node-wide (not PID-scoped).
func obiConfigForOdigos() *obi.Config {
	cfg := obi.DefaultConfig

	// --- App O11y (unchanged): per-workload traces exported to the node collector. ---
	cfg.EBPF.ContextPropagation = config.ContextPropagationHeaders
	cfg.Traces.TracesEndpoint = fmt.Sprintf("http://localhost:%d", consts.OTLPPort)

	// --- Net O11y + Stats O11y (added): node-wide flow + TCP-health capture. ---
	cfg.NetworkFlows.Enable = true
	// socket_filter is CNI-safe (it does not attach TC programs that could clash with the CNI's);
	// in k8s pods talk over the pod network, so loopback-only capture is not the concern it is on
	// a VM. Capture all interfaces.
	cfg.NetworkFlows.Source = "socket_filter"
	cfg.NetworkFlows.ExcludeInterfaces = []string{}
	cfg.Prometheus.Port = NetworkPrometheusPort
	cfg.Prometheus.Path = "/metrics"
	cfg.Metrics.Features = export.LoadFeatures([]string{"network", "stats"})

	// Select the per-flow / per-stat attributes the netmetrics resolver needs to map a flow's
	// src/dst (address+port) to a PID/pod. Without these OBI emits only default-on labels and the
	// resolver has nothing to join on. App O11y keeps the DynamicPIDSelector regardless of this.
	cfg.Attributes.Select = attributes.Selection{
		attributes.NetworkFlow.Section: attributes.InclusionLists{
			Include: []string{"src.address", "dst.address", "src.port", "dst.port", "direction", "transport"},
		},
		// OBI v0.8.0 exposes a single TCP stat metric: obi_stat_tcp_rtt_seconds. Retransmits and
		// failed-connection stats are not emitted by this OBI version, so there is nothing to select
		// for them here; the downstream TCP-health parser (string-based over the Prometheus text)
		// simply finds no such lines and the TCPHealth detector stays inert until OBI ships them.
		attributes.StatTCPRtt.Section: attributes.InclusionLists{
			Include: []string{"src.address", "dst.address", "src.port", "dst.port"},
		},
	}
	// instrumenter.Run does not call normalize() (only LoadConfig does); normalize ourselves.
	cfg.Attributes.Select.Normalize()
	return &cfg
}

// CreateInstrumentation starts the OBI instrumenter if it is not already running
// and returns an obiInstrumentation that allows adding/removing this PID using the dynamic selector.
func (f *OBIInstrumentationFactory) CreateInstrumentation(ctx context.Context, pid int, settings instrumentation.Settings) (instrumentation.Instrumentation, error) {
	if f.obiCtx == nil {
		obiCtx, obiCtxCancel := context.WithCancel(ctx)
		f.obiCtx = obiCtx
		f.obiCtxCancel = obiCtxCancel

		go func() {
			err := instrumenter.Run(f.obiCtx, f.obiCfg, instrumenter.WithDynamicPIDSelector(f.selector))
			if err != nil && f.obiCtx.Err() == nil {
				f.logger.Error("OBI instrumenter exited with error", "err", err)
			}
		}()
	}
	return &obiInstrumentation{selector: f.selector, pid: pid, factory: f}, nil
}

// obiInstrumentation implements instrumentation.Instrumentation; it only holds the factory's
// selector so Close(ctx) can call RemovePIDs for the PID.
type obiInstrumentation struct {
	selector *discover.DynamicPIDSelector
	pid      int
	factory  *OBIInstrumentationFactory
}

func (o *obiInstrumentation) Load(context.Context) (instrumentation.Status, error) {
	o.selector.AddPIDs(uint32(o.pid))
	return instrumentation.Status{}, nil
}

func (o *obiInstrumentation) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (o *obiInstrumentation) Close(_ context.Context) error {
	o.selector.RemovePIDs(uint32(o.pid))
	if _, ok := o.selector.GetPIDs(); !ok {
		if o.factory.obiCtxCancel != nil {
			o.factory.obiCtxCancel()
			o.factory.obiCtxCancel = nil
			o.factory.obiCtx = nil
		}
	}
	return nil
}

func (o *obiInstrumentation) ApplyConfig(context.Context, instrumentation.Config) error {
	return nil
}
