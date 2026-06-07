package obi

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"

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
//
// The single instrumenter runs every enabled OBI pillar concurrently: App O11y (per-PID traces,
// scoped by the DynamicPIDSelector) and the node-wide Net O11y + Stats O11y pillars
// (see obiConfigForOdigos). It can be started two ways:
//   - lazily, on the first CreateInstrumentation call (an app workload was instrumented), or
//   - eagerly, via EnsureRunning, so node-wide network/stats capture begins at odiglet boot
//     even when no application workload is instrumented yet.
//
// mu guards the start/stop lifecycle (obiCtx/obiCtxCancel/eager) so concurrent
// CreateInstrumentation/Close/EnsureRunning calls are race-free.
type OBIInstrumentationFactory struct {
	logger *commonlogger.OdigosLogger

	mu           sync.Mutex
	obiCtx       context.Context
	obiCtxCancel context.CancelFunc
	// eager is set by EnsureRunning. When true, the instrumenter is kept running for the whole
	// odiglet lifetime (node-wide flow capture) and is NOT torn down when the last app PID is
	// removed by Close — only ctx cancellation stops it.
	eager bool

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

// NetworkMetricsEnabledEnv is the env var that turns the node-wide network-map + security
// capture on. It is OFF by default because the feature requires odiglet to run with
// hostNetwork (so OBI's socket_filter attaches in the HOST network namespace and observes every
// pod's traffic — see obiConfigForOdigos); that is a deployment change the operator opts into
// via Helm (odiglet.networkMetrics.enabled), which sets both hostNetwork and this env together.
// When unset/false, odiglet behaves exactly as before: App O11y traces only, no network pillars.
const NetworkMetricsEnabledEnv = "ODIGOS_NETWORK_METRICS_ENABLED"

// NetworkMetricsEnabled reports whether the network-map/security capture is enabled for this
// odiglet (default false). Read by both obiConfigForOdigos (whether to add the Net/Stats pillars
// to OBI) and the network-metrics runnable (whether to eager-start and serve), so the two stay
// in lockstep off a single switch.
func NetworkMetricsEnabled() bool {
	v, err := strconv.ParseBool(os.Getenv(NetworkMetricsEnabledEnv))
	return err == nil && v
}

// obiConfigForOdigos returns the OBI config for odiglet. It always runs App O11y (per-workload
// traces). When NetworkMetricsEnabled() is true it ALSO adds, on the SAME OBI instance, the
// node-wide Net O11y + Stats O11y pillars — OBI runs all enabled pillars as concurrent pipelines
// (pkg/instrumenter: setupAppO11y + setupNetO11y in one errgroup), so a single instrumenter serves
// both with no second OBI instance and no extra container. App PIDs are supplied dynamically via
// the DynamicPIDSelector; netolly is node-wide (not PID-scoped). When the feature is disabled the
// returned config is byte-for-byte the original App-only config, so odiglet's behaviour and
// resource profile are unchanged for clusters that do not opt in.
func obiConfigForOdigos() *obi.Config {
	cfg := obi.DefaultConfig

	// --- App O11y (always): per-workload traces exported to the node collector. ---
	cfg.EBPF.ContextPropagation = config.ContextPropagationHeaders
	cfg.Traces.TracesEndpoint = fmt.Sprintf("http://localhost:%d", consts.OTLPPort)

	if !NetworkMetricsEnabled() {
		return &cfg
	}

	// --- Net O11y + Stats O11y (opt-in): node-wide flow + TCP-health capture. ---
	cfg.NetworkFlows.Enable = true
	// socket_filter is CNI-safe: it attaches an AF_PACKET socket filter rather than TC programs
	// that could clash with the CNI's. It captures every interface in the CURRENT network namespace,
	// so to see all pods' traffic (not just odiglet's own) odiglet must run with hostNetwork=true —
	// then socket_filter attaches in the host netns where the CNI bridge and all pods' veth host-ends
	// live, and node-wide pod-to-pod flows transit interfaces it can read. (Without hostNetwork the
	// map would only show odiglet's own connections.) The Helm network-metrics flag sets hostNetwork.
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

// startLocked starts the background OBI instrumenter bound to ctx if it is not already running.
// Caller must hold f.mu. The instrumenter runs all enabled pillars (App + Net + Stats) and lives
// until ctx is cancelled. The first caller's ctx wins; subsequent calls are no-ops, so an early
// EnsureRunning (odiglet's long-lived root ctx) takes precedence over a later lazy start.
func (f *OBIInstrumentationFactory) startLocked(ctx context.Context) {
	if f.obiCtx != nil {
		return
	}
	obiCtx, obiCtxCancel := context.WithCancel(ctx)
	f.obiCtx = obiCtx
	f.obiCtxCancel = obiCtxCancel

	go func() {
		err := instrumenter.Run(obiCtx, f.obiCfg, instrumenter.WithDynamicPIDSelector(f.selector))
		if err != nil && obiCtx.Err() == nil {
			f.logger.Error("OBI instrumenter exited with error", "err", err)
		}
	}()
}

// EnsureRunning eagerly starts the OBI instrumenter (idempotent) bound to the given long-lived
// context, so the node-wide Net O11y + Stats O11y pillars capture flows even when NO application
// workload is instrumented. Without this the instrumenter starts lazily on the first
// CreateInstrumentation call, which would leave the network map / security view empty until
// something is app-instrumented. The eager flag also keeps the instrumenter alive across app
// (de)instrumentation: per-PID Close no longer tears it down. Safe to call concurrently.
func (f *OBIInstrumentationFactory) EnsureRunning(ctx context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.eager = true
	f.startLocked(ctx)
}

// CreateInstrumentation starts the OBI instrumenter if it is not already running
// and returns an obiInstrumentation that allows adding/removing this PID using the dynamic selector.
func (f *OBIInstrumentationFactory) CreateInstrumentation(ctx context.Context, pid int, settings instrumentation.Settings) (instrumentation.Instrumentation, error) {
	f.mu.Lock()
	f.startLocked(ctx)
	f.mu.Unlock()
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
	f := o.factory
	f.mu.Lock()
	defer f.mu.Unlock()
	// When eager-started for node-wide network/stats capture, keep the instrumenter running
	// regardless of app PIDs — tearing it down here would stop flow collection the moment the
	// last app workload is un-instrumented. Only ctx cancellation stops an eager instrumenter.
	if f.eager {
		return nil
	}
	if _, ok := f.selector.GetPIDs(); !ok {
		if f.obiCtxCancel != nil {
			f.obiCtxCancel()
			f.obiCtxCancel = nil
			f.obiCtx = nil
		}
	}
	return nil
}

func (o *obiInstrumentation) ApplyConfig(context.Context, instrumentation.Config) error {
	return nil
}
