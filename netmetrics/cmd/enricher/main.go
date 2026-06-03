// Command enricher is a deployable network-metrics enrichment service built on the
// shared github.com/odigos-io/odigos/netmetrics package.
//
// It scrapes OBI's raw network flows (bare 5-tuple), resolves each flow's local
// endpoint to a service via the shared ServiceResolver + PrometheusEnricher (/proc
// socket->PID, then injected PID->service and peer->service lookups), and re-exposes
// enriched, OTel-semconv-named metrics for Prometheus to scrape (hence any destination
// such as Grafana via remote_write).
//
// Identity sources are injected from config here, exactly as the VM agent injects its
// profileattrs PID->Source table and odiglet its k8s informer — the one shared resolver
// works under either producer.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"log/slog"

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

	endpoints, err := netmetrics.NewEndpointResolver()
	if err != nil {
		log.Error("endpoint resolver", "err", err)
		os.Exit(1)
	}

	// Injected PeerToService (CIDR registry; stand-in for k8s informer / DNS feed).
	type peerNet struct {
		net *net.IPNet
		svc string
	}
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
	// informer (odiglet). Here derived from config rules over the endpoint snapshot.
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
	enricher := netmetrics.NewPrometheusEnricher(*obiURL, resolver)

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
		if err := enricher.Render(w); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
		}
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
		return config{}, err
	}
	return cfg, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
