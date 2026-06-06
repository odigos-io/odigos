package netmetrics

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeOBI serves a fixed set of obi_network_flow_bytes_total lines so the builder
// can be exercised without a running OBI/eBPF.
func fakeOBI(t *testing.T, lines string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, lines)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func flow(src string, sp int, dst string, dp int, transport, dir string, bytes float64) string {
	return fmt.Sprintf("obi_network_flow_bytes_total{src_address=%q,dst_address=%q,src_port=%q,dst_port=%q,transport=%q,direction=%q} %g\n",
		src, dst, fmt.Sprint(sp), fmt.Sprint(dp), transport, dir, bytes)
}

func TestSnapshot_StatesOrientationAndRates(t *testing.T) {
	// inventory(instrumented) listens :18080; redis(discovered, comm-only) listens :9000.
	resolver := newTestResolver(
		map[string]Endpoint{
			"172.17.0.1:18080": {PID: 100, Comm: "python3"},
			"10.0.0.5:9000":    {PID: 7, Comm: "redis-server"},
			"172.17.0.1:50000": {PID: 100, Comm: "python3"}, // inventory's egress socket to stripe
		},
		map[int]Service{100: {Name: "inventory", Instrumented: true}}, // redis pid7 unmapped -> comm fallback
		map[string]Service{"172.17.0.3": {Name: "frontend"}},          // frontend resolved via registry (off-host)
	)

	lines := "" +
		flow("172.17.0.3", 40000, "172.17.0.1", 18080, "tcp", "ingress", 1000) + // frontend -> inventory
		flow("172.17.0.1", 50000, "1.2.3.4", 443, "tcp", "egress", 500) + // inventory -> stripe (raw IP)
		flow("10.0.0.5", 9000, "8.8.8.8", 53, "udp", "egress", 200) // redis -> external

	b := NewSnapshotBuilder(fakeOBI(t, lines), resolver, "socket_filter", "test-host")

	// First build: establishes the rate baseline; cumulative bytes are exact already.
	s1, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	byName := func(s Snapshot) map[string]ServiceNode {
		m := map[string]ServiceNode{}
		for _, n := range s.Nodes {
			m[n.Name] = n
		}
		return m
	}
	n1 := byName(s1)

	if got := n1["inventory"].State; got != StateInstrumented {
		t.Errorf("inventory should be instrumented, got %s", got)
	}
	if got := n1["redis-server"].State; got != StateDiscovered {
		t.Errorf("redis-server should be discovered, got %s", got)
	}
	if got := n1["frontend"].State; got != StateExternal {
		t.Errorf("frontend (off-host peer) should be external, got %s", got)
	}
	if got := n1["1.2.3.4"].State; got != StateExternal {
		t.Errorf("raw-IP peer should be external, got %s", got)
	}

	// Edge orientation: client is the initiator, server owns the listen port.
	var fe *Edge
	for i := range s1.Edges {
		if s1.Edges[i].Client == "frontend" {
			fe = &s1.Edges[i]
		}
	}
	if fe == nil || fe.Server != "inventory" || fe.ServerPort != "18080" {
		t.Fatalf("expected frontend->inventory:18080 edge, got %+v", fe)
	}
	if fe.Bytes != 1000 {
		t.Errorf("edge cumulative bytes should be 1000, got %g", fe.Bytes)
	}

	// Totals: 1 instrumented (inventory), 1 discovered (redis), 3 external
	// (frontend, 1.2.3.4, 8.8.8.8).
	if s1.Totals.Instrumented != 1 || s1.Totals.Discovered != 1 || s1.Totals.External != 3 {
		t.Errorf("totals mismatch: %+v", s1.Totals)
	}

	// Second build with grown counters -> positive rates.
	lines2 := "" +
		flow("172.17.0.3", 40000, "172.17.0.1", 18080, "tcp", "ingress", 3000) + // +2000
		flow("172.17.0.1", 50000, "1.2.3.4", 443, "tcp", "egress", 500) +
		flow("10.0.0.5", 9000, "8.8.8.8", 53, "udp", "egress", 200)
	b.enricher.obiURL = fakeOBI(t, lines2)

	// Age the baseline sample past minRateWindow so the next Build will diff against it.
	ageSamples(b, 2*minRateWindow)

	s2, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	n2 := byName(s2)
	if n2["inventory"].RxPerSec <= 0 {
		t.Errorf("inventory should have positive rx rate after counter grew, got %g", n2["inventory"].RxPerSec)
	}
	if n2["frontend"].TxPerSec <= 0 {
		t.Errorf("frontend (client) should have positive tx rate, got %g", n2["frontend"].TxPerSec)
	}
	if s2.Totals.BytesPerSec <= 0 {
		t.Errorf("total throughput should be positive, got %g", s2.Totals.BytesPerSec)
	}
}

// ageSamples backdates all of the builder's clocks (rate samples, EWMA last-build, and
// each node's last-seen), simulating that `by` has elapsed without the test sleeping —
// so a baseline becomes eligible and the time-aware EWMA blends a meaningful fraction.
func ageSamples(b *SnapshotBuilder, by time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.samples {
		b.samples[i].t = b.samples[i].t.Add(-by)
	}
	if !b.lastBuild.IsZero() {
		b.lastBuild = b.lastBuild.Add(-by)
	}
	for _, s := range b.smoothed {
		s.lastSeen = s.lastSeen.Add(-by)
	}
}

// TestSnapshot_StickyNodeSurvivesMissedScrape verifies a node lingers (decaying) after
// a scrape misses it, instead of blinking out, and is dropped only past stickyTTL.
func TestSnapshot_StickyNodeSurvivesMissedScrape(t *testing.T) {
	resolver := newTestResolver(
		map[string]Endpoint{"172.17.0.1:18080": {PID: 100, Comm: "python3"}},
		map[int]Service{100: {Name: "inventory", Instrumented: true}}, nil,
	)
	present := flow("172.17.0.3", 40000, "172.17.0.1", 18080, "tcp", "ingress", 1000)
	b := NewSnapshotBuilder(fakeOBI(t, present), resolver, "s", "h")

	if _, err := b.Build(); err != nil { // inventory seen
		t.Fatal(err)
	}
	has := func(s Snapshot, name string) bool {
		for _, n := range s.Nodes {
			if n.Name == name {
				return true
			}
		}
		return false
	}

	// Next scrape returns nothing, only a little time has passed: node must persist.
	b.enricher.obiURL = fakeOBI(t, "")
	ageSamples(b, 2*time.Second)
	s2, _ := b.Build()
	if !has(s2, "inventory") {
		t.Error("node should be sticky across a single missed scrape, not blink out")
	}

	// Now let it age past stickyTTL with continued absence: it should be dropped.
	ageSamples(b, 2*stickyTTL)
	s3, _ := b.Build()
	if has(s3, "inventory") {
		t.Error("node should be dropped once it is older than stickyTTL")
	}
}

// TestPeerResolver_HostAndNames covers host-IP recognition and that already-named peers
// (non-IPs) pass through reverse-DNS untouched.
func TestPeerResolver_HostAndNames(t *testing.T) {
	p := newPeerResolver("myhost")
	if !p.isHostIP("127.0.0.1") || !p.isHostIP("::1") {
		t.Error("loopback must be recognized as a host IP")
	}
	// the whole loopback range (incl. the systemd-resolved stub) and link-local are host-local,
	// so localhost service calls and the cloud metadata endpoint are never flagged as external.
	if !p.isHostIP("127.0.0.53") {
		t.Error("127.0.0.53 (systemd-resolved stub) must be host-local, not external")
	}
	if !p.isHostIP("169.254.169.254") {
		t.Error("link-local (cloud metadata) must be host-local, not external")
	}
	if p.isHostIP("8.8.8.8") {
		t.Error("a public IP must not be a host IP")
	}
	if got := p.pretty("frontend"); got != "frontend" {
		t.Errorf("an already-named peer must pass through unchanged, got %q", got)
	}
	if got := p.pretty("203.0.113.7"); got != "203.0.113.7" {
		t.Errorf("first sighting of an IP returns the IP while rDNS resolves in bg, got %q", got)
	}
}

// TestSnapshot_RatesStableUnderRapidPolling verifies the sample-ring keeps reporting a
// rate (no flicker to 0) when Build is polled faster than OBI refreshes its counters:
// even with the counter plateaued across several rapid polls, each diffs against a
// baseline that is still >= minRateWindow old.
func TestSnapshot_RatesStableUnderRapidPolling(t *testing.T) {
	resolver := newTestResolver(
		map[string]Endpoint{"172.17.0.1:18080": {PID: 100, Comm: "python3"}},
		map[int]Service{100: {Name: "inventory", Instrumented: true}}, nil,
	)
	b := NewSnapshotBuilder(fakeOBI(t, flow("172.17.0.3", 40000, "172.17.0.1", 18080, "tcp", "ingress", 1000)), resolver, "s", "h")

	if _, err := b.Build(); err != nil { // baseline sample at counter=1000
		t.Fatal(err)
	}
	ageSamples(b, 2*minRateWindow) // make it an eligible baseline

	// Counter jumps to 5000 and then PLATEAUS; poll 3x rapidly — every poll must still
	// report a rate from the aged baseline, not 0.
	b.enricher.obiURL = fakeOBI(t, flow("172.17.0.3", 40000, "172.17.0.1", 18080, "tcp", "ingress", 5000))
	for i := 0; i < 3; i++ {
		s, err := b.Build()
		if err != nil {
			t.Fatal(err)
		}
		var rate float64
		for _, n := range s.Nodes {
			if n.Name == "inventory" {
				rate = n.RxPerSec
			}
		}
		if rate <= 0 {
			t.Fatalf("poll %d flickered to 0; the aged baseline should keep the rate stable", i+1)
		}
	}
}
