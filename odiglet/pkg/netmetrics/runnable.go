package netmetrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/netmetrics"
	obisdk "github.com/odigos-io/odigos/odiglet/pkg/ebpf/sdks/obi"
	"github.com/odigos-io/odigos/securitymetrics"
)

// OBIStarter is the slice of the OBI factory the component needs: eagerly start the shared
// instrumenter so the node-wide Net/Stats pillars capture flows even when no application
// workload is instrumented. *obi.OBIInstrumentationFactory satisfies it. May be nil (then the
// instrumenter starts lazily on first app instrumentation and the API returns 502 until then).
type OBIStarter interface {
	EnsureRunning(ctx context.Context)
}

// Config tunes the network-metrics component. Defaults are production-sane; every field is
// overridable by an environment variable so operators can adjust thresholds, ports, and export
// destinations without rebuilding the image. See loadConfig for the variable names.
type Config struct {
	// Enabled gates the whole component. When false, the API is not served and OBI is not
	// eager-started (the instrumenter then starts lazily on first app instrumentation, as before),
	// so a node with nothing instrumented pays no always-on eBPF cost for network capture.
	Enabled bool
	APIPort int

	RefreshInterval time.Duration

	// Security
	SecurityEnabled    bool
	SecurityPoll       time.Duration
	SecurityWarmup     time.Duration
	VolumetricFloorBps float64
	VolumetricSpike    float64
	TCPFailedConnScan  float64
	TCPRetxStorm       float64
	// FindingsJSONLPath, when set, appends each finding as JSON-lines for SIEM ingestion ("" = off).
	FindingsJSONLPath string
	// BaselinePath, when set, persists/restores the drift baseline across restarts ("" = off).
	BaselinePath string
	// BaselineSaveInterval is how often the baseline is flushed to BaselinePath.
	BaselineSaveInterval time.Duration
}

const (
	// defaultAPIPort serves /api/network + /api/security for this node (consumed by odictl `w`/`S`).
	// NOTE: odiglet runs hostNetwork on older clusters (k8s < v1.26, or when noHostNetwork is unset
	// on such versions), where this binds on the node and 9100 is node_exporter's conventional port.
	// On a clash the listener fails, the (best-effort) component logs and exits without taking
	// odiglet down; operators override via ODIGOS_NETWORK_METRICS_API_PORT. On pod-network clusters
	// (the common modern case) it is pod-scoped and collision-free.
	defaultAPIPort = 9100

	defaultRefreshInterval      = 2 * time.Second
	defaultSecurityPollInterval = 5 * time.Second
	defaultSecurityWarmup       = 60 * time.Second
	defaultBaselineSaveInterval = 30 * time.Second
)

// loadConfig builds the component config from environment variables, falling back to the
// production defaults above for anything unset or unparseable.
func loadConfig() Config {
	return Config{
		Enabled:              envBool("ODIGOS_NETWORK_METRICS_ENABLED", true),
		APIPort:              envInt("ODIGOS_NETWORK_METRICS_API_PORT", defaultAPIPort),
		RefreshInterval:      envDuration("ODIGOS_NETWORK_METRICS_REFRESH_INTERVAL", defaultRefreshInterval),
		SecurityEnabled:      envBool("ODIGOS_NETWORK_SECURITY_ENABLED", true),
		SecurityPoll:         envDuration("ODIGOS_NETWORK_SECURITY_POLL_INTERVAL", defaultSecurityPollInterval),
		SecurityWarmup:       envDuration("ODIGOS_NETWORK_SECURITY_WARMUP", defaultSecurityWarmup),
		VolumetricFloorBps:   envFloat("ODIGOS_NETWORK_SECURITY_VOLUMETRIC_FLOOR_BPS", 0),
		VolumetricSpike:      envFloat("ODIGOS_NETWORK_SECURITY_VOLUMETRIC_SPIKE", 0),
		TCPFailedConnScan:    envFloat("ODIGOS_NETWORK_SECURITY_TCP_FAILED_CONN_SCAN", 0),
		TCPRetxStorm:         envFloat("ODIGOS_NETWORK_SECURITY_TCP_RETX_STORM", 0),
		FindingsJSONLPath:    os.Getenv("ODIGOS_NETWORK_SECURITY_FINDINGS_PATH"),
		BaselinePath:         os.Getenv("ODIGOS_NETWORK_SECURITY_BASELINE_PATH"),
		BaselineSaveInterval: envDuration("ODIGOS_NETWORK_SECURITY_BASELINE_SAVE_INTERVAL", defaultBaselineSaveInterval),
	}
}

// Component runs the network-map + security engines for this node, scraping the node-wide OBI
// flow/stats metrics (exposed by odiglet's single OBI instance on NetworkPrometheusPort) and
// resolving every flow to a k8s service identity. It is registered as an odiglet Runnable, so
// it lives in odiglet's process — no extra container/binary.
type Component struct {
	client   client.Client
	cache    cache.Cache
	nodeName string
	obi      OBIStarter
	cfg      Config
	log      *commonlogger.OdigosLogger
}

// NewComponent builds the node network-metrics component over the manager's cached client. obi
// may be nil (the OBI instrumenter then starts lazily on first app instrumentation).
func NewComponent(c client.Client, cch cache.Cache, nodeName string, obi OBIStarter) *Component {
	return &Component{
		client:   c,
		cache:    cch,
		nodeName: nodeName,
		obi:      obi,
		cfg:      loadConfig(),
		log:      commonlogger.LoggerCompat().With("subsystem", "netmetrics"),
	}
}

// Run is the odiglet Runnable entrypoint (blocks until ctx is cancelled). It eager-starts OBI's
// node-wide capture, waits for the informer cache so pod lookups work, then builds the shared
// resolver/snapshot/security engine and serves /api/network + /api/security.
func (c *Component) Run(ctx context.Context) error {
	if !c.cfg.Enabled {
		c.log.Info("network metrics disabled (ODIGOS_NETWORK_METRICS_ENABLED=false); not serving")
		return nil
	}

	// Eager-start OBI so the Net/Stats pillars capture node-wide flows even when no application
	// workload is instrumented yet. Without this, OBI starts lazily on the first instrumentation
	// and the network map would be empty until then.
	if c.obi != nil {
		c.obi.EnsureRunning(ctx)
	} else {
		c.log.Info("no OBI starter wired; network capture begins only once a workload is instrumented")
	}

	if !c.cache.WaitForCacheSync(ctx) {
		// ctx cancelled during startup, or the cache failed to sync. Either way we cannot resolve
		// pods, so there is nothing useful to serve.
		return fmt.Errorf("netmetrics: informer cache did not sync (ctx err: %v)", ctx.Err())
	}

	obiURL := fmt.Sprintf("http://127.0.0.1:%d/metrics", obisdk.NetworkPrometheusPort)

	// shared /proc endpoint resolver (ip:port -> PID); works under odiglet's hostPID+privileged.
	endpoints, err := netmetrics.NewEndpointResolver()
	if err != nil {
		return fmt.Errorf("netmetrics: endpoint resolver: %w", err)
	}

	// k8s identity: PID->pod->service and peerIP->pod->service (same service.name as traces).
	id := NewK8sIdentity(ctx, c.client)
	resolver := netmetrics.NewServiceResolver(endpoints,
		netmetrics.PIDToService(id.PIDToService),
		netmetrics.PeerToService(id.PeerToService))

	snapshots := netmetrics.NewSnapshotBuilder(obiURL, resolver, "socket_filter", c.nodeName)

	go c.goRecover(ctx, "proc-refresh", func() { c.refreshLoop(ctx, endpoints) })

	var secEngine *securitymetrics.Engine
	if c.cfg.SecurityEnabled {
		secEngine = c.buildSecurityEngine(ctx, obiURL, resolver)
		go c.goRecover(ctx, "security-engine", func() { secEngine.Run(ctx) })
	}

	return c.serve(ctx, snapshots, secEngine)
}

// buildSecurityEngine wires the same 5 detectors as the VM agent, fed by its OWN snapshot/health
// builders (so its polling doesn't perturb the /api/network rate state), with findings exported
// to the structured log and (optionally) a JSONL file, and the drift baseline persisted across
// restarts (optional). Threshold knobs come from Config; zero means "use the detector default".
func (c *Component) buildSecurityEngine(ctx context.Context, obiURL string, resolver *netmetrics.ServiceResolver) *securitymetrics.Engine {
	secBuilder := netmetrics.NewSnapshotBuilder(obiURL, resolver, "socket_filter", c.nodeName)
	secHealth := netmetrics.NewPrometheusEnricher(obiURL, resolver)

	baseline := securitymetrics.NewBaseline(c.cfg.SecurityWarmup)
	if c.cfg.BaselinePath != "" {
		if err := securitymetrics.LoadBaseline(baseline, c.cfg.BaselinePath); err != nil {
			c.log.Warn("could not load security baseline; starting fresh", "path", c.cfg.BaselinePath, "err", err)
		}
	}

	engine := securitymetrics.NewEngine(baseline).
		AddSource(securitymetrics.NewNetworkSource(secBuilder.Build, c.cfg.SecurityPoll)).
		AddSource(securitymetrics.NewTCPHealthSource(secHealth.ResolveTCPHealth, c.cfg.SecurityPoll)).
		AddDetector(securitymetrics.ExposureDetector{}).
		AddDetector(securitymetrics.DriftDetector{}).
		AddDetector(securitymetrics.NewVolumetricDetector(c.cfg.VolumetricFloorBps, c.cfg.VolumetricSpike)).
		AddDetector(securitymetrics.NewThreatIntelDetector(securitymetrics.NewStaticIntel(nil))).
		AddDetector(securitymetrics.NewTCPHealthDetector(c.cfg.TCPFailedConnScan, c.cfg.TCPRetxStorm))

	// Findings fan out to the structured log and, when configured, a JSONL file for SIEM ingestion.
	sink := securitymetrics.MultiSink{logSink{log: c.log}}
	if c.cfg.FindingsJSONLPath != "" {
		if js, err := securitymetrics.NewJSONLSink(c.cfg.FindingsJSONLPath); err != nil {
			c.log.Warn("could not open findings JSONL sink; export disabled", "path", c.cfg.FindingsJSONLPath, "err", err)
		} else {
			sink = append(sink, js)
		}
	}
	engine.OnFinding(sink.Emit)

	if c.cfg.BaselinePath != "" {
		go c.goRecover(ctx, "baseline-persist", func() { c.persistBaselineLoop(ctx, baseline) })
	}
	return engine
}

// logSink adapts the structured logger to securitymetrics.FindingSink so it can sit in a MultiSink.
type logSink struct{ log *commonlogger.OdigosLogger }

func (l logSink) Emit(f securitymetrics.Finding) {
	l.log.Info("security finding",
		"severity", f.Severity.String(), "category", string(f.Cat),
		"service", f.Subject.Service, "title", f.Title)
}

// persistBaselineLoop flushes the drift baseline to disk periodically and once more on shutdown,
// so a restart resumes from learned edges/destinations/ports instead of re-flagging everything.
func (c *Component) persistBaselineLoop(ctx context.Context, b *securitymetrics.Baseline) {
	t := time.NewTicker(c.cfg.BaselineSaveInterval)
	defer t.Stop()
	save := func() {
		if err := securitymetrics.SaveBaseline(b, c.cfg.BaselinePath); err != nil {
			c.log.Warn("could not save security baseline", "path", c.cfg.BaselinePath, "err", err)
		}
	}
	for {
		select {
		case <-ctx.Done():
			save()
			return
		case <-t.C:
			save()
		}
	}
}

// goRecover runs fn, recovering from a panic so a fault in a background loop can never crash
// odiglet (network/security is best-effort). The component simply loses that loop and logs it.
func (c *Component) goRecover(_ context.Context, name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			c.log.Error("network-metrics goroutine panicked; that loop is stopped", "loop", name, "panic", fmt.Sprintf("%v", r))
		}
	}()
	fn()
}

// refreshLoop keeps the /proc endpoint table current.
func (c *Component) refreshLoop(ctx context.Context, endpoints *netmetrics.EndpointResolver) {
	if err := endpoints.Refresh(); err != nil {
		c.log.Warn("initial /proc refresh", "err", err)
	}
	t := time.NewTicker(c.cfg.RefreshInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := endpoints.Refresh(); err != nil {
				c.log.Warn("/proc refresh", "err", err)
			}
		}
	}
}

// serve exposes /api/network (+ /api/security when enabled) on APIPort until ctx is cancelled.
func (c *Component) serve(ctx context.Context, snapshots *netmetrics.SnapshotBuilder, sec *securitymetrics.Engine) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/network", func(w http.ResponseWriter, _ *http.Request) {
		snap, err := snapshots.Build()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(snap); err != nil {
			c.log.Warn("encode network snapshot", "err", err)
		}
	})
	if sec != nil {
		mux.HandleFunc("/api/security", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(sec.Report()); err != nil {
				c.log.Warn("encode security report", "err", err)
			}
		})
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", c.cfg.APIPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()
	c.log.Info("network-metrics API serving",
		"port", c.cfg.APIPort, "obi_prometheus_port", obisdk.NetworkPrometheusPort, "security", sec != nil)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// --- small env helpers (production defaults when unset/unparseable) ---

func envBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
