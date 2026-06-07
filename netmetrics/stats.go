package netmetrics

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// TCP-health (StatsO11y) metrics OBI emits when the "stats" feature is enabled. Each is
// resolved to service identity exactly like the byte-flow metric — they carry the same
// src/dst address+port labels once those are selected (see the controller's Attributes.Select).
var (
	statRttRe        = regexp.MustCompile(`^obi_stat_tcp_rtt_seconds_(sum|count)\{([^}]*)\}\s+([0-9.eE+-]+)`)
	statRetransmitRe = regexp.MustCompile(`^obi_stat_tcp_retransmits(?:_total)?\{([^}]*)\}\s+([0-9.eE+-]+)`)
	statFailedConnRe = regexp.MustCompile(`^obi_stat_tcp_failed_connections(?:_total)?\{([^}]*)\}\s+([0-9.eE+-]+)`)
)

// TCPHealth is per-service-to-peer TCP health, resolved from OBI's stats pillar: average
// round-trip time and cumulative retransmit / failed-connection counts. It is the substrate
// for security scan/RST/latency detection and for ops health on the network map.
type TCPHealth struct {
	Service     string  `json:"service"`
	Peer        string  `json:"peer"`
	ServerPort  string  `json:"server_port"`
	AvgRttMs    float64 `json:"avg_rtt_ms"`
	Retransmits float64 `json:"retransmits"`
	FailedConns float64 `json:"failed_connections"`
}

type tcpHealthKey struct{ service, peer, port string }

type tcpHealthAcc struct {
	rttSum, rttCount float64
	retransmits      float64
	failedConns      float64
}

// ResolveTCPHealth scrapes OBI once and returns per-edge TCP health, resolved to service
// names via the same ServiceResolver used for flows. Returns an empty slice (not an error)
// if the stats feature is off or no stat lines are present.
func (e *PrometheusEnricher) ResolveTCPHealth() ([]TCPHealth, error) {
	raw, err := e.scrape()
	if err != nil {
		return nil, err
	}
	return e.resolveTCPHealthFrom(raw), nil
}

// resolveTCPHealthFrom parses already-scraped OBI exposition for TCP health, so a caller that
// also parses flows from the same body avoids a second (expensive) scrape of OBI's large
// stats output.
func (e *PrometheusEnricher) resolveTCPHealthFrom(raw string) []TCPHealth {
	agg := map[tcpHealthKey]*tcpHealthAcc{}
	get := func(lbl map[string]string) *tcpHealthAcc {
		sp, _ := strconv.Atoi(lbl["src_port"])
		dp, _ := strconv.Atoi(lbl["dst_port"])
		fi, ok := e.resolver.Resolve(lbl["src_address"], sp, lbl["dst_address"], dp)
		if !ok {
			return nil
		}
		k := tcpHealthKey{fi.Local.Name, fi.Peer.Name, strconv.Itoa(fi.ServerPort)}
		a := agg[k]
		if a == nil {
			a = &tcpHealthAcc{}
			agg[k] = a
		}
		return a
	}

	for _, line := range strings.Split(raw, "\n") {
		if m := statRttRe.FindStringSubmatch(line); m != nil {
			lbl := parsePromLabels(m[2])
			v, _ := strconv.ParseFloat(m[3], 64)
			if a := get(lbl); a != nil {
				if m[1] == "sum" {
					a.rttSum += v
				} else {
					a.rttCount += v
				}
			}
			continue
		}
		if m := statRetransmitRe.FindStringSubmatch(line); m != nil {
			lbl := parsePromLabels(m[1])
			v, _ := strconv.ParseFloat(m[2], 64)
			if a := get(lbl); a != nil {
				a.retransmits += v
			}
			continue
		}
		if m := statFailedConnRe.FindStringSubmatch(line); m != nil {
			lbl := parsePromLabels(m[1])
			v, _ := strconv.ParseFloat(m[2], 64)
			if a := get(lbl); a != nil {
				a.failedConns += v
			}
		}
	}

	out := make([]TCPHealth, 0, len(agg))
	for k, a := range agg {
		h := TCPHealth{Service: k.service, Peer: k.peer, ServerPort: k.port, Retransmits: a.retransmits, FailedConns: a.failedConns}
		if a.rttCount > 0 {
			h.AvgRttMs = (a.rttSum / a.rttCount) * 1000 // seconds → ms
		}
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Service != out[j].Service {
			return out[i].Service < out[j].Service
		}
		return out[i].Peer < out[j].Peer
	})
	return out
}
