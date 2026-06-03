// enricher is a thin reference consumer of the shared odigos/netmetrics package.
// It is the stand-in for the per-agent glue (vm-agent / odiglet) and the collector
// processor: it scrapes OBI's raw network flows, resolves identity via the SHARED
// ServiceResolver, and re-exposes enriched, OTel-semconv-named metrics.
//
// The identity sources are injected here (config-file PID/peer maps) exactly as the
// VM agent would inject its PID->Source table and odiglet its k8s informer — proving
// the same shared resolver works under either producer.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	obiURL := flag.String("obi", "http://localhost:8999/metrics", "OBI prometheus endpoint")
	listen := flag.String("listen", ":9100", "enriched metrics listen address")
	cfgPath := flag.String("config", "config.json", "service mapping config")
	interval := flag.Duration("interval", 2*time.Second, "/proc refresh interval")
	flag.Parse()

	raw, err := os.ReadFile(*cfgPath)
	if err != nil {
		log.Fatalf("read config: %v", err)
	}
	var cfg config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Fatalf("parse config: %v", err)
	}

	// SHARED endpoint resolver (socket->PID via /proc).
	endpoints, err := netmetrics.NewEndpointResolver()
	if err != nil {
		log.Fatalf("endpoint resolver: %v", err)
	}

	// peer registry (CIDR -> service): the injected PeerToService.
	var peers []peerNet
	for _, p := range cfg.Peers {
		if _, n, err := net.ParseCIDR(p.CIDR); err == nil {
			peers = append(peers, peerNet{n, p.Service})
		} else {
			log.Printf("bad peer cidr %q: %v", p.CIDR, err)
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

	// PID -> service: the injected PIDToService. In the real agents this is the
	// profileattrs table (vm-agent) or the k8s informer (odiglet). Here we derive it
	// by applying the demo config rules (port / comm / cmdline) against the shared
	// resolver's endpoint snapshot, rebuilt each refresh.
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

	go func() {
		_ = endpoints.Refresh()
		rebuild()
		for range time.Tick(*interval) {
			_ = endpoints.Refresh()
			rebuild()
		}
	}()

	log.Printf("netmetrics enricher: OBI=%s listen=%s (shared odigos/netmetrics)", *obiURL, *listen)
	http.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		obi, err := scrape(*obiURL)
		if err != nil {
			http.Error(w, err.Error(), 502)
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
				// neither endpoint is local: leave as unknown -> peer raw IP
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
		fmt.Fprintf(w, "# HELP enricher_resolved_endpoints Local socket endpoints resolved to a PID\n# TYPE enricher_resolved_endpoints gauge\nenricher_resolved_endpoints %d\n", endpoints.Size())
	})
	log.Fatal(http.ListenAndServe(*listen, nil))
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
