package netmetrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/netmetrics"
	obisdk "github.com/odigos-io/odigos/odiglet/pkg/ebpf/sdks/obi"
	"github.com/odigos-io/odigos/securitymetrics"
)

const (
	// APIPort serves /api/network + /api/security for this node (consumed by odictl `w`/`S`).
	APIPort = 9100

	refreshInterval      = 2 * time.Second
	securityPollInterval = 5 * time.Second
	securityWarmup       = 60 * time.Second
)

// Component runs the network-map + security engines for this node, scraping the node-wide OBI
// flow/stats metrics (exposed by odiglet's single OBI instance on NetworkPrometheusPort) and
// resolving every flow to a k8s service identity. It is registered as an odiglet Runnable, so
// it lives in odiglet's process — no extra container/binary.
type Component struct {
	client   client.Client
	cache    cache.Cache
	nodeName string
	log      *commonlogger.OdigosLogger
}

// NewComponent builds the node network-metrics component over the manager's cached client.
func NewComponent(c client.Client, cch cache.Cache, nodeName string) *Component {
	return &Component{
		client:   c,
		cache:    cch,
		nodeName: nodeName,
		log:      commonlogger.LoggerCompat().With("subsystem", "netmetrics"),
	}
}

// Run is the odiglet Runnable entrypoint (blocks until ctx is cancelled). It waits for the
// informer cache so pod lookups work, then builds the shared resolver/snapshot/security engine
// and serves /api/network + /api/security.
func (c *Component) Run(ctx context.Context) error {
	if !c.cache.WaitForCacheSync(ctx) {
		return fmt.Errorf("netmetrics: cache did not sync")
	}

	obiURL := fmt.Sprintf("http://localhost:%d/metrics", obisdk.NetworkPrometheusPort)

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

	// security engine over its OWN snapshot builder (so its polling doesn't perturb the
	// /api/network rate state) — same 5 detectors as the VM agent.
	secBuilder := netmetrics.NewSnapshotBuilder(obiURL, resolver, "socket_filter", c.nodeName)
	secHealth := netmetrics.NewPrometheusEnricher(obiURL, resolver)
	secEngine := securitymetrics.NewEngine(securitymetrics.NewBaseline(securityWarmup)).
		AddSource(securitymetrics.NewNetworkSource(secBuilder.Build, securityPollInterval)).
		AddSource(securitymetrics.NewTCPHealthSource(secHealth.ResolveTCPHealth, securityPollInterval)).
		AddDetector(securitymetrics.ExposureDetector{}).
		AddDetector(securitymetrics.DriftDetector{}).
		AddDetector(securitymetrics.NewVolumetricDetector(0, 0)).
		AddDetector(securitymetrics.NewThreatIntelDetector(securitymetrics.NewStaticIntel(nil))).
		AddDetector(securitymetrics.NewTCPHealthDetector(0, 0))
	secEngine.OnFinding(func(f securitymetrics.Finding) {
		c.log.Info("security finding", "severity", f.Severity.String(), "category", string(f.Cat),
			"service", f.Subject.Service, "title", f.Title)
	})

	go c.refreshLoop(ctx, endpoints)
	go secEngine.Run(ctx)

	return c.serve(ctx, snapshots, secEngine)
}

// refreshLoop keeps the /proc endpoint table current.
func (c *Component) refreshLoop(ctx context.Context, endpoints *netmetrics.EndpointResolver) {
	if err := endpoints.Refresh(); err != nil {
		c.log.Warn("initial /proc refresh", "err", err)
	}
	t := time.NewTicker(refreshInterval)
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

// serve exposes /api/network + /api/security on APIPort until ctx is cancelled.
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
	mux.HandleFunc("/api/security", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(sec.Report()); err != nil {
			c.log.Warn("encode security report", "err", err)
		}
	})

	srv := &http.Server{Addr: fmt.Sprintf(":%d", APIPort), Handler: mux}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()
	c.log.Info("network-metrics API serving", "port", APIPort, "obi_prometheus_port", obisdk.NetworkPrometheusPort)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
