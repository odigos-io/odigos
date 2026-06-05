package netmetrics

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// NodeState classifies a service node by how Odigos sees it. It is what lets the
// network map double as an instrumentation surface: a Discovered node is a real
// process Odigos could instrument in one click, an Instrumented node already shares
// service.name with its traces/profiles, and an External node is off-host.
type NodeState string

// minRateWindow is the shortest interval over which a byte/sec rate is computed. The
// rate baseline is held (not re-sampled) until this elapses, so rates stay correct even
// when Build is polled faster than OBI refreshes its counters. See Build.
const minRateWindow = 5 * time.Second

const (
	// StateInstrumented: resolved to an enabled Odigos Source — has traces/profiles
	// under the same service.name.
	StateInstrumented NodeState = "instrumented"
	// StateDiscovered: a real local process not yet an Odigos Source — an
	// instrumentation candidate ("trace this in one click").
	StateDiscovered NodeState = "discovered"
	// StateExternal: a peer not owned by any local process (raw IP / off-host).
	StateExternal NodeState = "external"
)

// Edge is a directed client->server aggregate. Client is the connection initiator,
// server is the listening side (owner of ServerPort), so edges are stable regardless
// of which side OBI happened to observe as the flow source.
type Edge struct {
	Client      string  `json:"client"`
	Server      string  `json:"server"`
	Transport   string  `json:"transport"`
	ServerPort  string  `json:"server_port"`
	Bytes       float64 `json:"bytes"`         // cumulative bytes seen on this edge
	BytesPerSec float64 `json:"bytes_per_sec"` // rate vs the previous snapshot
}

func (e Edge) key() string {
	return e.Client + "\x00" + e.Server + "\x00" + e.Transport + "\x00" + e.ServerPort
}

// ServiceNode is one service aggregated across every edge it participates in.
type ServiceNode struct {
	Name         string    `json:"name"`
	State        NodeState `json:"state"`
	Instrumented bool      `json:"instrumented"`
	BytesPerSec  float64   `json:"bytes_per_sec"` // total throughput (as client + as server)
	TxPerSec     float64   `json:"tx_per_sec"`    // bytes/s where this node is the client
	RxPerSec     float64   `json:"rx_per_sec"`    // bytes/s where this node is the server
	Peers        int       `json:"peers"`         // distinct neighbors
	Out          []string  `json:"out"`           // servers this node calls (as client)
	In           []string  `json:"in"`            // clients that call this node (as server)
}

// Totals is the at-a-glance header summary for the whole host.
type Totals struct {
	BytesPerSec  float64 `json:"bytes_per_sec"`
	Services     int     `json:"services"`
	Edges        int     `json:"edges"`
	Instrumented int     `json:"instrumented"`
	Discovered   int     `json:"discovered"`
	External     int     `json:"external"`
}

// Snapshot is the full network picture at one instant: nodes + edges + totals.
// It is what the odictl network TUI renders and what /api/network serves as JSON.
type Snapshot struct {
	Timestamp time.Time     `json:"timestamp"`
	Source    string        `json:"source"`
	Host      string        `json:"host"`
	Nodes     []ServiceNode `json:"nodes"`
	Edges     []Edge        `json:"edges"`
	Totals    Totals        `json:"totals"`
}

// SnapshotBuilder turns successive OBI scrapes into Snapshots. It is stateful: it
// remembers the previous cumulative byte counts per edge so it can derive per-second
// rates (OBI exposes monotonic counters; the TUI wants rates). Safe for concurrent
// Build calls (one HTTP handler at a time, but guarded anyway).
type SnapshotBuilder struct {
	enricher *PrometheusEnricher
	source   string
	host     string

	mu       sync.Mutex
	prev     map[string]float64 // edge.key() -> cumulative bytes at prevTime
	prevTime time.Time
}

// NewSnapshotBuilder builds a snapshot source over the same OBI scrape + resolver the
// enricher uses, so the network map and the exported metrics never disagree.
func NewSnapshotBuilder(obiURL string, resolver *ServiceResolver, source, host string) *SnapshotBuilder {
	return &SnapshotBuilder{
		enricher: NewPrometheusEnricher(obiURL, resolver),
		source:   source,
		host:     host,
		prev:     map[string]float64{},
	}
}

// resolvedFlow is one OBI flow line resolved to identities, before edge orientation.
type resolvedFlow struct {
	fi        FlowIdentity
	transport string
	direction string
	bytes     float64
}

// Build scrapes OBI once, resolves every flow, aggregates into nodes+edges, and fills
// in per-second rates relative to the previous Build. The first Build after start has
// no baseline, so rates are 0 until the second call (cumulative Bytes is still exact).
func (b *SnapshotBuilder) Build() (Snapshot, error) {
	flows, err := b.resolveFlows()
	if err != nil {
		return Snapshot{}, err
	}

	now := time.Now()
	b.mu.Lock()
	firstBuild := b.prevTime.IsZero()
	dt := now.Sub(b.prevTime).Seconds()
	haveBaseline := !firstBuild && dt > 0
	prev := b.prev
	b.mu.Unlock()

	edges := map[string]*Edge{}
	// node provenance, accumulated as we see each side of every flow.
	type nodeAcc struct {
		instrumented bool
		external     bool
		seen         bool
		out          map[string]struct{}
		in           map[string]struct{}
		tx, rx       float64 // cumulative; converted to rates below
	}
	nodes := map[string]*nodeAcc{}
	node := func(name string) *nodeAcc {
		n := nodes[name]
		if n == nil {
			n = &nodeAcc{out: map[string]struct{}{}, in: map[string]struct{}{}}
			nodes[name] = n
		}
		return n
	}

	for _, rf := range flows {
		localName := rf.fi.Local.Name
		peerName := rf.fi.Peer.Name
		if localName == "" {
			continue
		}

		// Orient the edge: client = initiator, server = listener (ServerPort owner).
		var client, server string
		if rf.fi.LocalIsSrc {
			client, server = localName, peerName
		} else {
			client, server = peerName, localName
		}

		ln := node(localName)
		ln.seen = true
		if rf.fi.Local.Instrumented {
			ln.instrumented = true
		}
		if peerName != "" {
			pn := node(peerName)
			pn.seen = true
			if !rf.fi.PeerIsLocal {
				pn.external = true
			} else if rf.fi.Peer.Instrumented {
				pn.instrumented = true
			}
			node(client).out[server] = struct{}{}
			node(server).in[client] = struct{}{}
			node(client).tx += rf.bytes
			node(server).rx += rf.bytes
		}

		e := &Edge{
			Client:     client,
			Server:     server,
			Transport:  rf.transport,
			ServerPort: strconv.Itoa(rf.fi.ServerPort),
		}
		k := e.key()
		if cur := edges[k]; cur != nil {
			cur.Bytes += rf.bytes
		} else {
			e.Bytes = rf.bytes
			edges[k] = e
		}
	}

	// finalize edges + rates
	cur := make(map[string]float64, len(edges))
	outEdges := make([]Edge, 0, len(edges))
	for k, e := range edges {
		cur[k] = e.Bytes
		if haveBaseline {
			if p, ok := prev[k]; ok && e.Bytes >= p {
				e.BytesPerSec = (e.Bytes - p) / dt
			}
		}
		outEdges = append(outEdges, *e)
	}
	sort.Slice(outEdges, func(i, j int) bool { return outEdges[i].BytesPerSec > outEdges[j].BytesPerSec })

	// finalize nodes
	var totals Totals
	outNodes := make([]ServiceNode, 0, len(nodes))
	for name, n := range nodes {
		if !n.seen {
			continue
		}
		state := StateDiscovered
		switch {
		case n.external:
			state = StateExternal
		case n.instrumented:
			state = StateInstrumented
		}
		sn := ServiceNode{
			Name:         name,
			State:        state,
			Instrumented: n.instrumented,
			Peers:        len(n.out) + len(n.in),
			Out:          keys(n.out),
			In:           keys(n.in),
		}
		if haveBaseline {
			// node tx/rx rates are derived from its edges so they stay consistent with
			// the edge rates; recompute by summing oriented edge rates.
			sn.TxPerSec, sn.RxPerSec = nodeRates(name, outEdges)
			sn.BytesPerSec = sn.TxPerSec + sn.RxPerSec
		}
		outNodes = append(outNodes, sn)

		totals.BytesPerSec += sn.BytesPerSec
		switch state {
		case StateInstrumented:
			totals.Instrumented++
		case StateDiscovered:
			totals.Discovered++
		case StateExternal:
			totals.External++
		}
	}
	totals.Services = len(outNodes)
	totals.Edges = len(outEdges)

	sort.Slice(outNodes, func(i, j int) bool { return outNodes[i].BytesPerSec > outNodes[j].BytesPerSec })

	// Advance the rate baseline only once per minRateWindow, NOT on every Build. OBI
	// exposes monotonic counters it refreshes on its own (multi-second) cadence; if the
	// caller polls faster than that (the TUI refreshes every ~2s) and we re-baselined
	// every call, most polls would diff two identical counter reads and show 0. By
	// holding the baseline until at least minRateWindow has elapsed, every Build reports
	// the rate over a meaningful window regardless of how often it is called.
	b.mu.Lock()
	if firstBuild || dt >= minRateWindow.Seconds() {
		b.prev = cur
		b.prevTime = now
	}
	b.mu.Unlock()

	return Snapshot{
		Timestamp: now,
		Source:    b.source,
		Host:      b.host,
		Nodes:     outNodes,
		Edges:     outEdges,
		Totals:    totals,
	}, nil
}

// nodeRates sums oriented edge rates for a node: tx = edges where it is client,
// rx = edges where it is server.
func nodeRates(name string, edges []Edge) (tx, rx float64) {
	for _, e := range edges {
		if e.Client == name {
			tx += e.BytesPerSec
		}
		if e.Server == name {
			rx += e.BytesPerSec
		}
	}
	return tx, rx
}

// resolveFlows scrapes OBI and resolves each raw flow line to a FlowIdentity, keeping
// provenance (instrumented / peer-local) that the name-collapsed FlowAgg path discards.
func (b *SnapshotBuilder) resolveFlows() ([]resolvedFlow, error) {
	raw, err := b.enricher.scrape()
	if err != nil {
		return nil, err
	}
	var out []resolvedFlow
	for _, line := range strings.Split(raw, "\n") {
		m := flowLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		lbl := parsePromLabels(m[1])
		val, _ := strconv.ParseFloat(m[2], 64)
		sp, _ := strconv.Atoi(lbl["src_port"])
		dp, _ := strconv.Atoi(lbl["dst_port"])
		fi, ok := b.enricher.resolver.Resolve(lbl["src_address"], sp, lbl["dst_address"], dp)
		if !ok {
			// neither endpoint is a local process; still show it as an external->external
			// flow anchored on the destination address so it is not silently dropped.
			fi = FlowIdentity{
				Local:      Service{Name: lbl["dst_address"]},
				Peer:       Service{Name: lbl["src_address"]},
				ServerPort: dp,
			}
		}
		out = append(out, resolvedFlow{fi: fi, transport: lbl["transport"], direction: lbl["direction"], bytes: val})
	}
	return out, nil
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
