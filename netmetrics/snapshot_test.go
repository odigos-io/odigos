package netmetrics

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
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

// TestSnapshot_BaselineHeldUnderRapidPolling verifies the rate baseline is NOT
// re-sampled on every Build — only once per minRateWindow — so polling faster than
// OBI refreshes its counters still yields a rate instead of 0.
func TestSnapshot_BaselineHeldUnderRapidPolling(t *testing.T) {
	resolver := newTestResolver(
		map[string]Endpoint{"172.17.0.1:18080": {PID: 100, Comm: "python3"}},
		map[int]Service{100: {Name: "inventory", Instrumented: true}}, nil,
	)
	b := NewSnapshotBuilder(fakeOBI(t, flow("172.17.0.3", 40000, "172.17.0.1", 18080, "tcp", "ingress", 1000)), resolver, "s", "h")

	if _, err := b.Build(); err != nil { // first build: sets baseline
		t.Fatal(err)
	}
	baseline := b.prevTime
	if baseline.IsZero() {
		t.Fatal("first build should set a baseline")
	}

	// Counter grows; poll again immediately (dt << minRateWindow).
	b.enricher.obiURL = fakeOBI(t, flow("172.17.0.3", 40000, "172.17.0.1", 18080, "tcp", "ingress", 5000))
	s2, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	if !b.prevTime.Equal(baseline) {
		t.Error("baseline must be HELD (not advanced) within minRateWindow under rapid polling")
	}
	// rate is still reported despite the tiny gap, because it diffs against the held baseline.
	var rate float64
	for _, n := range s2.Nodes {
		if n.Name == "inventory" {
			rate = n.RxPerSec
		}
	}
	if rate <= 0 {
		t.Errorf("rapid re-poll should still report a rate from the held baseline, got %g", rate)
	}
}
