package netmetrics

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// PrometheusEnricher scrapes OBI's raw network metrics and resolves each flow to service
// identity via a ServiceResolver. It can render enriched Prometheus exposition (Render)
// or hand back structured aggregates (ResolveFlows) for the OTLP pusher. Shared by the
// standalone enricher service, the VM-agent controller, and odiglet so the scrape+relabel
// logic exists once.
type PrometheusEnricher struct {
	obiURL   string
	resolver *ServiceResolver
	client   *http.Client
}

// NewPrometheusEnricher builds an enricher that scrapes obiURL (OBI's /metrics) and
// resolves flows via resolver.
func NewPrometheusEnricher(obiURL string, resolver *ServiceResolver) *PrometheusEnricher {
	return &PrometheusEnricher{
		obiURL:   obiURL,
		resolver: resolver,
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

var flowLineRe = regexp.MustCompile(`^obi_network_flow_bytes_total\{([^}]*)\}\s+([0-9.eE+-]+)`)

// FlowAgg is one enriched service-to-service flow aggregate.
type FlowAgg struct {
	Service    string
	Peer       string
	Transport  string
	Direction  string
	ServerPort string
	Bytes      float64
}

type enrichedKey struct{ service, peer, transport, direction, port string }

// ResolveFlows scrapes OBI once and returns enriched aggregates (service -> peer bytes).
func (e *PrometheusEnricher) ResolveFlows() ([]FlowAgg, error) {
	obi, err := e.scrape()
	if err != nil {
		return nil, err
	}
	agg := map[enrichedKey]float64{}
	for _, line := range strings.Split(obi, "\n") {
		m := flowLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		lbl := parsePromLabels(m[1])
		val, _ := strconv.ParseFloat(m[2], 64)
		sp, _ := strconv.Atoi(lbl["src_port"])
		dp, _ := strconv.Atoi(lbl["dst_port"])
		fi, ok := e.resolver.Resolve(lbl["src_address"], sp, lbl["dst_address"], dp)
		if !ok {
			fi.Local = Service{Name: "unknown"}
			fi.Peer = Service{Name: lbl["dst_address"]}
			fi.ServerPort = dp
		}
		agg[enrichedKey{fi.Local.Name, fi.Peer.Name, lbl["transport"], lbl["direction"], strconv.Itoa(fi.ServerPort)}] += val
	}
	out := make([]FlowAgg, 0, len(agg))
	for k, v := range agg {
		out = append(out, FlowAgg{k.service, k.peer, k.transport, k.direction, k.port, v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Bytes > out[j].Bytes })
	return out, nil
}

// Render scrapes OBI and writes enriched Prometheus exposition to w.
func (e *PrometheusEnricher) Render(w io.Writer) error {
	flows, err := e.ResolveFlows()
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "# HELP network_flow_bytes_total Bytes between services (OBI flow enriched via odigos/netmetrics)")
	fmt.Fprintln(w, "# TYPE network_flow_bytes_total counter")
	for _, f := range flows {
		fmt.Fprintf(w, "network_flow_bytes_total{service_name=%q,peer_service_name=%q,network_transport=%q,network_io_direction=%q,server_port=%q} %g\n",
			f.Service, f.Peer, f.Transport, f.Direction, f.ServerPort, f.Bytes)
	}
	fmt.Fprintf(w, "# HELP netmetrics_resolved_endpoints Local socket endpoints resolved to a PID\n# TYPE netmetrics_resolved_endpoints gauge\nnetmetrics_resolved_endpoints %d\n", e.resolver.Endpoints().Size())
	return nil
}

func (e *PrometheusEnricher) scrape() (string, error) {
	resp, err := e.client.Get(e.obiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return string(b), err
}

func parsePromLabels(s string) map[string]string {
	out := map[string]string{}
	for _, kv := range strings.Split(s, ",") {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			out[strings.TrimSpace(kv[:i])] = strings.Trim(strings.TrimSpace(kv[i+1:]), `"`)
		}
	}
	return out
}
