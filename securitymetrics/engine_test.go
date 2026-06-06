package securitymetrics

import (
	"context"
	"testing"
	"time"

	"github.com/odigos-io/odigos/netmetrics"
)

// fakeSnap returns a fixed network snapshot modelling: frontend(local) → payments(local) →
// api.stripe.com(external), plus frontend exposed.
func fakeSnap() netmetrics.Snapshot {
	n := func(name string, st netmetrics.NodeState, elig bool) netmetrics.ServiceNode {
		return netmetrics.ServiceNode{Name: name, State: st, Eligible: elig, WorkloadKind: "systemd"}
	}
	return netmetrics.Snapshot{
		Timestamp: time.Now(),
		Host:      "h",
		Nodes: []netmetrics.ServiceNode{
			n("frontend", netmetrics.StateDiscovered, true),
			n("payments", netmetrics.StateDiscovered, true),
			n("api.stripe.com", netmetrics.StateExternal, false),
			n("users", netmetrics.StateExternal, false), // off-host client → makes frontend a wildcard exposure
		},
		Edges: []netmetrics.Edge{
			{Client: "users", Server: "frontend", ServerPort: "8080", Transport: "tcp"}, // external → frontend = wildcard exposure
			{Client: "frontend", Server: "payments", ServerPort: "8443", Transport: "tcp"},
			{Client: "payments", Server: "api.stripe.com", ServerPort: "443", Transport: "tcp"},
		},
	}
}

func runEngineOnce(t *testing.T) Report {
	t.Helper()
	eng := NewEngine(NewBaseline(0)). // warm-up over so drift fires
						AddSource(NewNetworkSource(func() (netmetrics.Snapshot, error) { return fakeSnap(), nil }, time.Hour)).
						AddDetector(EgressDetector{}).
						AddDetector(ExposureDetector{}).
						AddDetector(DriftDetector{})

	ctx, cancel := context.WithCancel(context.Background())
	go eng.Run(ctx)
	// let the immediate emit + handling settle, then stop.
	deadline := time.Now().Add(2 * time.Second)
	var rep Report
	for time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		rep = eng.Report()
		if rep.Totals.Findings > 0 {
			break
		}
	}
	cancel()
	return rep
}

func TestEngine_EndToEndFindings(t *testing.T) {
	rep := runEngineOnce(t)

	// egress inventory should contain payments → api.stripe.com:443 (the only external dest)
	if len(rep.Inventory) != 1 || rep.Inventory[0].Service != "payments" || rep.Inventory[0].Peer != "api.stripe.com" {
		t.Errorf("egress inventory wrong: %+v", rep.Inventory)
	}

	// findings should include: an egress (external), an exposure for payments (server),
	// and drift (new external dest + new internal edge). Assert categories present.
	cats := map[Category]int{}
	for _, f := range rep.Findings {
		cats[f.Cat]++
	}
	if cats[CategoryEgress] == 0 {
		t.Error("expected an egress finding")
	}
	if cats[CategoryExposure] == 0 {
		t.Error("expected an exposure finding")
	}
	if cats[CategoryFlowNew] == 0 {
		t.Error("expected a drift (flow.new) finding")
	}

	// the new-external-dest drift finding should carry the instrument pivot (payments eligible).
	var pivot bool
	for _, f := range rep.Findings {
		if f.Cat == CategoryFlowNew && len(f.Actions) > 0 && f.Actions[0] == "instrument" {
			pivot = true
		}
	}
	if !pivot {
		t.Error("expected a drift finding offering the instrument pivot")
	}
}

func TestNetworkSource_SkipsEphemeralExposure(t *testing.T) {
	// a node serving on a high ephemeral port (an OS-assigned client source port) must NOT
	// produce an exposure event; a real low listen port must.
	snap := netmetrics.Snapshot{
		Timestamp: time.Now(),
		Nodes: []netmetrics.ServiceNode{
			{Name: "svc", State: netmetrics.StateDiscovered},
			{Name: "peer", State: netmetrics.StateDiscovered},
		},
		Edges: []netmetrics.Edge{
			{Client: "peer", Server: "svc", ServerPort: "44208", Transport: "tcp"}, // ephemeral → skip
			{Client: "peer", Server: "svc", ServerPort: "8080", Transport: "tcp"},  // real listen → keep
		},
	}
	src := NewNetworkSource(func() (netmetrics.Snapshot, error) { return snap, nil }, time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()
	exposurePorts := map[int]bool{}
	for ev := range src.Events(ctx) {
		if ev.Cat == CategoryExposure {
			exposurePorts[ev.Object.Port] = true
		}
		if len(exposurePorts) > 0 && exposurePorts[8080] {
			break
		}
	}
	if exposurePorts[44208] {
		t.Error("ephemeral port 44208 must not yield an exposure finding")
	}
	if !exposurePorts[8080] {
		t.Error("real listen port 8080 must yield an exposure finding")
	}
}

func TestEngine_DedupAggregates(t *testing.T) {
	eng := NewEngine(NewBaseline(0)).AddDetector(EgressDetector{})
	ev := egressEvent("a", "ext.example.com", 443, true, false, false)
	eng.handle(ev)
	eng.handle(ev)
	eng.handle(ev)
	rep := eng.Report()
	if len(rep.Findings) != 1 {
		t.Fatalf("identical events should dedupe to 1 finding, got %d", len(rep.Findings))
	}
	if rep.Findings[0].Count != 3 {
		t.Errorf("expected Count=3 after 3 sightings, got %d", rep.Findings[0].Count)
	}
}

func TestNetworkSource_EmitsExpectedShapes(t *testing.T) {
	src := NewNetworkSource(func() (netmetrics.Snapshot, error) { return fakeSnap(), nil }, time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ch := src.Events(ctx)

	var egress, exposure int
	timeout := time.After(800 * time.Millisecond)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				goto done
			}
			switch ev.Cat {
			case CategoryEgress:
				egress++
			case CategoryExposure:
				exposure++
			}
		case <-timeout:
			goto done
		}
	}
done:
	// one egress (payments→stripe; frontend→payments is internal so still egress event but not external)
	if egress < 1 {
		t.Errorf("expected at least one egress event, got %d", egress)
	}
	if exposure < 1 {
		t.Errorf("expected at least one exposure event (payments is a server), got %d", exposure)
	}
}
