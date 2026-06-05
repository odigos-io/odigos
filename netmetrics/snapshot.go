package netmetrics

import (
	"math"
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

// Display-smoothing constants. The raw per-window rate is noisy (TCP bursts, OBI
// counter cadence), so the map shows an exponential moving average instead — numbers
// and bars glide rather than jump. Nodes also linger for stickyTTL after their last
// sighting (with a decaying rate) so a single missed scrape doesn't blink them out.
const (
	smoothTau = 6 * time.Second  // EWMA time constant; larger = smoother, slower to react
	stickyTTL = 12 * time.Second // keep a node this long after it was last seen
)

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

	// Instrumentation surface — what it takes to trace this node. Populated for
	// discovered local processes; empty for external peers.
	WorkloadKind string `json:"workload_kind,omitempty"` // Source Kind to create ("docker"/"systemd"/"process")
	Eligible     bool   `json:"eligible,omitempty"`      // language/runtime matches an instrumentation target
	Runtime      string `json:"runtime,omitempty"`       // detected language/runtime
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
	peer     *peerResolver

	mu        sync.Mutex
	samples   []rateSample           // recent (time, cumulative-bytes) reads, oldest first
	smoothed  map[string]*nodeSmooth // node name -> EWMA + last-seen state (display smoothing)
	lastBuild time.Time              // for time-aware EWMA between Build calls
}

// rateSample is one OBI read: the cumulative bytes per edge at a point in time.
type rateSample struct {
	t     time.Time
	bytes map[string]float64 // edge.key() -> cumulative bytes
}

// nodeSmooth holds a node's exponentially-smoothed rates plus the metadata from its
// most recent sighting, so a node can be carried forward (decaying) while it is sticky.
type nodeSmooth struct {
	bps, tx, rx  float64
	lastSeen     time.Time
	state        NodeState
	instrumented bool
	peers        int
	out, in      []string
	kind         string
	eligible     bool
	runtime      string
}

// NewSnapshotBuilder builds a snapshot source over the same OBI scrape + resolver the
// enricher uses, so the network map and the exported metrics never disagree.
func NewSnapshotBuilder(obiURL string, resolver *ServiceResolver, source, host string) *SnapshotBuilder {
	return &SnapshotBuilder{
		enricher: NewPrometheusEnricher(obiURL, resolver),
		source:   source,
		host:     host,
		peer:     newPeerResolver(host),
		smoothed: map[string]*nodeSmooth{},
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
	// Pick the rate baseline: the newest prior sample that is already at least
	// minRateWindow old. This keeps the rate window stable (always ~minRateWindow,
	// never near-zero) even when Build is polled faster than OBI refreshes its
	// counters — so rates don't flicker to 0 between OBI updates.
	b.mu.Lock()
	var prev map[string]float64
	var dt float64
	for i := len(b.samples) - 1; i >= 0; i-- {
		if age := now.Sub(b.samples[i].t); age >= minRateWindow {
			prev = b.samples[i].bytes
			dt = age.Seconds()
			break
		}
	}
	b.mu.Unlock()
	haveBaseline := prev != nil && dt > 0

	edges := map[string]*Edge{}
	// node provenance, accumulated as we see each side of every flow.
	type nodeAcc struct {
		instrumented bool
		external     bool
		seen         bool
		out          map[string]struct{}
		in           map[string]struct{}
		tx, rx       float64 // cumulative; converted to rates below
		kind         string  // workload Kind (docker/systemd/process)
		eligible     bool    // instrumentation-eligible
		runtime      string  // detected runtime/language
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
	// applyWorkloadMeta copies the instrumentation-surface fields from a resolved
	// Service onto its node, preferring a non-empty/eligible value (a node may be
	// seen across several flows/PIDs; an eligible sighting wins).
	applyWorkloadMeta := func(n *nodeAcc, svc Service) {
		if svc.Kind != "" && n.kind == "" {
			n.kind = svc.Kind
		}
		if svc.Runtime != "" && n.runtime == "" {
			n.runtime = svc.Runtime
		}
		if svc.Eligible {
			n.eligible = true
		}
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
		applyWorkloadMeta(ln, rf.fi.Local)
		if peerName != "" {
			pn := node(peerName)
			pn.seen = true
			if !rf.fi.PeerIsLocal {
				pn.external = true
			} else if rf.fi.Peer.Instrumented {
				pn.instrumented = true
			}
			if rf.fi.PeerIsLocal {
				applyWorkloadMeta(pn, rf.fi.Peer)
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

	// Fold this scrape's instant node rates into the per-node EWMA, carry forward
	// recently-seen nodes that are missing this scrape (decaying), and emit the
	// smoothed, stable result. All under the lock since b.smoothed is mutated.
	b.mu.Lock()

	// EWMA blend factor from the real gap since the last Build (time-aware so the
	// smoothing is correct regardless of poll cadence).
	alpha := 1.0
	if !b.lastBuild.IsZero() {
		if gap := now.Sub(b.lastBuild).Seconds(); gap > 0 {
			alpha = 1 - math.Exp(-gap/smoothTau.Seconds())
		}
	}

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
		var tx, rx float64
		if haveBaseline {
			tx, rx = nodeRates(name, outEdges)
		}
		s := b.smoothed[name]
		if s == nil {
			s = &nodeSmooth{}
			b.smoothed[name] = s
		}
		s.bps += alpha * ((tx + rx) - s.bps)
		s.tx += alpha * (tx - s.tx)
		s.rx += alpha * (rx - s.rx)
		s.lastSeen = now
		s.state, s.instrumented = state, n.instrumented
		s.peers = len(n.out) + len(n.in)
		s.out, s.in = keys(n.out), keys(n.in)
		s.kind, s.eligible, s.runtime = n.kind, n.eligible, n.runtime
	}

	// Carry forward sticky nodes (seen recently but absent this scrape): decay their
	// rate toward 0 so they fade instead of blinking out; drop once past stickyTTL.
	var totals Totals
	outNodes := make([]ServiceNode, 0, len(b.smoothed))
	for name, s := range b.smoothed {
		if now.Sub(s.lastSeen) > stickyTTL {
			delete(b.smoothed, name)
			continue
		}
		if !s.lastSeen.Equal(now) {
			s.bps += alpha * (0 - s.bps)
			s.tx += alpha * (0 - s.tx)
			s.rx += alpha * (0 - s.rx)
		}
		outNodes = append(outNodes, ServiceNode{
			Name:         name,
			State:        s.state,
			Instrumented: s.instrumented,
			BytesPerSec:  s.bps,
			TxPerSec:     s.tx,
			RxPerSec:     s.rx,
			Peers:        s.peers,
			Out:          s.out,
			In:           s.in,
			WorkloadKind: s.kind,
			Eligible:     s.eligible,
			Runtime:      s.runtime,
		})
		// total throughput counts each byte once: a node's rx is bytes it received,
		// and every edge has exactly one receiver, so summing rx avoids double counting.
		totals.BytesPerSec += s.rx
		switch s.state {
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

	// Stable order: smoothed rate (changes slowly) then name, so rows don't leapfrog
	// each other every frame the way sorting by the raw instantaneous rate would.
	sort.Slice(outNodes, func(i, j int) bool {
		if outNodes[i].BytesPerSec != outNodes[j].BytesPerSec {
			return outNodes[i].BytesPerSec > outNodes[j].BytesPerSec
		}
		return outNodes[i].Name < outNodes[j].Name
	})

	// Record this read and drop samples older than we'd ever use as a baseline
	// (bounding memory at hundreds of edges × a few samples).
	b.samples = append(b.samples, rateSample{t: now, bytes: cur})
	keepFrom := now.Add(-3 * minRateWindow)
	drop := 0
	for drop < len(b.samples)-1 && b.samples[drop].t.Before(keepFrom) {
		drop++
	}
	b.samples = b.samples[drop:]
	b.lastBuild = now
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
		src, dst := lbl["src_address"], lbl["dst_address"]
		fi, ok := b.enricher.resolver.Resolve(src, sp, dst, dp)
		if ok {
			// Local side resolved to a PID/Source. Prettify the peer if it is an
			// off-host raw IP (reverse-DNS / hostname), leaving named peers untouched.
			if !fi.PeerIsLocal {
				fi.Peer.Name = b.peer.pretty(fi.Peer.Name)
			}
		} else {
			// Neither endpoint resolved to a local PID. If one side is a host IP, this
			// is host traffic we could not attribute — collapse it under a single host
			// node (so it is not a wall of bare host-IP nodes), with the off-host side
			// as the named external peer. Otherwise show both, reverse-DNS named.
			srcHost, dstHost := b.peer.isHostIP(src), b.peer.isHostIP(dst)
			switch {
			case dstHost && !srcHost: // inbound to this host
				fi = FlowIdentity{Local: Service{Name: b.host}, Peer: Service{Name: b.peer.pretty(src)}, ServerPort: dp}
			case srcHost && !dstHost: // outbound from this host
				fi = FlowIdentity{Local: Service{Name: b.host}, Peer: Service{Name: b.peer.pretty(dst)}, ServerPort: dp, LocalIsSrc: true}
			default: // both off-host (or both host): anchor on dst, name both
				fi = FlowIdentity{Local: Service{Name: b.peer.pretty(dst)}, Peer: Service{Name: b.peer.pretty(src)}, ServerPort: dp}
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
