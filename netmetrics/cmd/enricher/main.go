// Command enricher is a deployable network-metrics enrichment service built on the
// shared github.com/odigos-io/odigos/netmetrics package.
//
// It scrapes OBI's raw network flows (bare 5-tuple), resolves each flow's local
// endpoint to a service via the shared ServiceResolver (/proc socket->PID, then an
// injected PID->service and peer->service lookup), and re-exposes enriched,
// OTel-semconv-named metrics for Prometheus to scrape (and thus any destination,
// e.g. Grafana, via remote_write).
//
// Identity sources are injected from config here, exactly as the VM agent injects
// its profileattrs PID->Source table and odiglet injects its k8s informer — proving
// the one shared resolver works under either producer.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/odigos-io/odigos/netmetrics"
)

type rule struct {
	Port    int    `json:"port,omitempty"`
	Comm    string `json:"comm,omitempty"`
	Cmdline string `json:"cmdline,omitempty"`
	Service string `json:"service"`
}
type peerRule struct {
	CIDR    string `json:"cidr"`
	Service string `json:"service"`
}
type config struct {
	Services []rule     `json:"services"`
	Peers    []peerRule `json:"peers"`
}

type peerNet struct {
	net *net.IPNet
	svc string
}

var flowLine = regexp.MustCompile(`^obi_network_flow_bytes_total\{([^}]*)\}\s+([0-9.eE+-]+)`)

func parseLabels(s string) map[string]string {
	out := map[string]string{}
	for _, kv := range strings.Split(s, ",") {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			out[strings.TrimSpace(kv[:i])] = strings.Trim(strings.TrimSpace(kv[i+1:]), `"`)
		}
	}
	return out
}

type aggKey struct{ service, peer, transport, direction, port string }

func main() {
	obiURL := flag.String("obi", envOr("OBI_METRICS_URL", "http://localhost:8999/metrics"), "OBI prometheus endpoint")
	listen := flag.String("listen", envOr("LISTEN_ADDR", ":9100"), "enriched metrics + health listen address")
	cfgPath := flag.String("config", envOr("CONFIG_PATH", "config.json"), "service mapping config")
	interval := flag.Duration("interval", 2*time.Second, "/proc refresh interval")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		log.Error("load config", "err", err)
		os.Exit(1)
	}

	// SHARED endpoint resolver (socket->PID via /proc).
	endpoints, err := netmetrics.NewEndpointResolver()
	if err != nil {
		log.Error("endpoint resolver", "err", err)
		os.Exit(1)
	}

	// Injected PeerToService (CIDR registry; stand-in for k8s informer / DNS feed).
	var peers []peerNet
	for _, p := range cfg.Peers {
		if _, n, err := net.ParseCIDR(p.CIDR); err == nil {
			peers = append(peers, peerNet{n, p.Service})
		} else {
			log.Warn("bad peer cidr", "cidr", p.CIDR, "err", err)
		}
	}
	peerToSvc := netmetrics.PeerToService(func(ip string) (netmetrics.Service, bool) {
		pip := net.ParseIP(ip)
		for _, p := range peers {
			if pip != nil && p.net.Contains(pip) {
				return netmetrics.Service{Name: p.svc}, true
			}
		}
		return netmetrics.Service{}, false
	})

	// Injected PIDToService: in the real agents the profileattrs table (VM) or k8s
	// informer (odiglet). Here we build a PID->service map by applying config rules
	// against the shared resolver's endpoint snapshot, rebuilt each refresh.
	var pidMu sync.RWMutex
	pidSvc := map[int]netmetrics.Service{}
	rebuild := func() {
		m := map[int]netmetrics.Service{}
		for _, ep := range endpoints.Snapshot() {
			for _, r := range cfg.Services {
				if (r.Cmdline != "" && strings.Contains(ep.Cmd, r.Cmdline)) ||
					(r.Port != 0 && r.Port == ep.Port) ||
					(r.Comm != "" && strings.Contains(ep.Comm, r.Comm)) {
					m[ep.PID] = netmetrics.Service{Name: r.Service}
					break
				}
			}
		}
		pidMu.Lock()
		pidSvc = m
		pidMu.Unlock()
	}
	pidToSvc := netmetrics.PIDToService(func(pid int) (netmetrics.Service, bool) {
		pidMu.RLock()
		defer pidMu.RUnlock()
		s, ok := pidSvc[pid]
		return s, ok
	})

	resolver := netmetrics.NewServiceResolver(endpoints, pidToSvc, peerToSvc)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var ready atomic.Bool
	go func() {
		if err := endpoints.Refresh(); err != nil {
			log.Warn("initial /proc refresh", "err", err)
		}
		rebuild()
		ready.Store(true)
		t := time.NewTicker(*interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := endpoints.Refresh(); err != nil {
					log.Warn("/proc refresh", "err", err)
				}
				rebuild()
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if ready.Load() {
			w.Write([]byte("ready"))
			return
		}
		http.Error(w, "not ready", http.StatusServiceUnavailable)
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		obi, err := scrape(*obiURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		agg := map[aggKey]float64{}
		for _, line := range strings.Split(obi, "\n") {
			m := flowLine.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			lbl := parseLabels(m[1])
			val, _ := strconv.ParseFloat(m[2], 64)
			sp, _ := strconv.Atoi(lbl["src_port"])
			dp, _ := strconv.Atoi(lbl["dst_port"])
			fi, ok := resolver.Resolve(lbl["src_address"], sp, lbl["dst_address"], dp)
			if !ok {
				fi.Local = netmetrics.Service{Name: "unknown"}
				fi.Peer = netmetrics.Service{Name: lbl["dst_address"]}
				fi.ServerPort = dp
			}
			agg[aggKey{fi.Local.Name, fi.Peer.Name, lbl["transport"], lbl["direction"], strconv.Itoa(fi.ServerPort)}] += val
		}
		fmt.Fprintln(w, "# HELP network_flow_bytes_total Bytes between services (OBI flow enriched via shared odigos/netmetrics)")
		fmt.Fprintln(w, "# TYPE network_flow_bytes_total counter")
		keys := make([]aggKey, 0, len(agg))
		for k := range agg {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return agg[keys[i]] > agg[keys[j]] })
		for _, k := range keys {
			fmt.Fprintf(w, "network_flow_bytes_total{service_name=%q,peer_service_name=%q,network_transport=%q,network_io_direction=%q,server_port=%q} %g\n",
				k.service, k.peer, k.transport, k.direction, k.port, agg[k])
		}
		fmt.Fprintf(w, "# HELP netmetrics_resolved_endpoints Local socket endpoints resolved to a PID\n# TYPE netmetrics_resolved_endpoints gauge\nnetmetrics_resolved_endpoints %d\n", endpoints.Size())
	})

	srv := &http.Server{Addr: *listen, Handler: mux}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	log.Info("netmetrics enricher started", "obi", *obiURL, "listen", *listen, "interval", interval.String())
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("http server", "err", err)
		os.Exit(1)
	}
	log.Info("netmetrics enricher stopped")
}

func loadConfig(path string) (config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}
	var cfg config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

func scrape(url string) (string, error) {
	c := &http.Client{Timeout: 5 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return string(b), err
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
