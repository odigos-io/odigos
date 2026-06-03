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

// PrometheusEnricher scrapes OBI's raw network metrics, resolves each flow to service
// identity via a ServiceResolver, and renders enriched, OTel-semconv-named exposition.
// Shared by the standalone enricher service AND the VM-agent network controller so the
// scrape+relabel logic exists once.
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

type enrichedKey struct{ service, peer, transport, direction, port string }

// Render scrapes OBI and writes enriched exposition to w.
func (e *PrometheusEnricher) Render(w io.Writer) error {
	obi, err := e.scrape()
	if err != nil {
		return err
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
	fmt.Fprintln(w, "# HELP network_flow_bytes_total Bytes between services (OBI flow enriched via odigos/netmetrics)")
	fmt.Fprintln(w, "# TYPE network_flow_bytes_total counter")
	keys := make([]enrichedKey, 0, len(agg))
	for k := range agg {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return agg[keys[i]] > agg[keys[j]] })
	for _, k := range keys {
		fmt.Fprintf(w, "network_flow_bytes_total{service_name=%q,peer_service_name=%q,network_transport=%q,network_io_direction=%q,server_port=%q} %g\n",
			k.service, k.peer, k.transport, k.direction, k.port, agg[k])
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
