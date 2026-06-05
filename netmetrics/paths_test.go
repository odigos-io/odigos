package netmetrics

import "testing"

// demoSnap models a small slice of the simple-demo call graph:
//
//	frontend -> inventory(●), pricing(●), coupon(◐ -> membership ◐ -> pricing ●), stripe(☁)
func demoSnap() Snapshot {
	n := func(name string, st NodeState, bps float64, out ...string) ServiceNode {
		return ServiceNode{Name: name, State: st, BytesPerSec: bps, Out: out}
	}
	return Snapshot{Nodes: []ServiceNode{
		n("frontend", StateInstrumented, 1000, "inventory", "pricing", "coupon", "stripe"),
		n("inventory", StateInstrumented, 400),
		n("pricing", StateInstrumented, 300),
		n("coupon", StateDiscovered, 200, "membership"),
		n("membership", StateDiscovered, 100, "pricing"), // pricing reached again (dedupe)
		n("stripe", StateExternal, 50),
	}}
}

func TestBuildCallTree_PathAndCoverage(t *testing.T) {
	ct := BuildCallTree(demoSnap(), "frontend", 0)

	if ct.Root != "frontend" || len(ct.Hops) == 0 || ct.Hops[0].Name != "frontend" {
		t.Fatalf("root hop wrong: %+v", ct.Hops)
	}
	// Hottest-first child order under frontend: inventory(400) before coupon(200).
	order := map[string]int{}
	for i, h := range ct.Hops {
		order[h.Name] = i
	}
	if order["inventory"] > order["coupon"] {
		t.Errorf("expected hotter inventory before coupon, got %+v", ct.Hops)
	}
	// coupon -> membership must be deeper than coupon.
	var coupon, membership CallHop
	for _, h := range ct.Hops {
		switch h.Name {
		case "coupon":
			coupon = h
		case "membership":
			membership = h
		}
	}
	if membership.Depth != coupon.Depth+1 {
		t.Errorf("membership should be one level under coupon: coupon=%d membership=%d", coupon.Depth, membership.Depth)
	}

	// Coverage: distinct instrumentable downstream hops = inventory, pricing, coupon,
	// membership (stripe is external, excluded). Instrumented = inventory, pricing.
	// Gaps = coupon, membership.
	if ct.Total != 4 || ct.Instrumented != 2 {
		t.Errorf("coverage wrong: total=%d instrumented=%d (want 4/2)", ct.Total, ct.Instrumented)
	}
	gaps := map[string]bool{}
	for _, g := range ct.Gaps {
		gaps[g] = true
	}
	if !gaps["coupon"] || !gaps["membership"] || gaps["pricing"] {
		t.Errorf("gaps wrong: %v (want coupon+membership, not pricing)", ct.Gaps)
	}
}

func TestBuildCallTree_CycleTerminates(t *testing.T) {
	// a -> b -> c -> a (cycle)
	snap := Snapshot{Nodes: []ServiceNode{
		{Name: "a", State: StateDiscovered, Out: []string{"b"}},
		{Name: "b", State: StateDiscovered, Out: []string{"c"}},
		{Name: "c", State: StateDiscovered, Out: []string{"a"}},
	}}
	ct := BuildCallTree(snap, "a", 0)
	// must terminate and mark the back-edge to "a" as a cycle, not recurse forever.
	var sawCycle bool
	for _, h := range ct.Hops {
		if h.Name == "a" && h.Cycle {
			sawCycle = true
		}
	}
	if !sawCycle {
		t.Errorf("expected the back-edge to 'a' to be marked as a cycle; hops=%+v", ct.Hops)
	}
}

func TestBuildCallTree_ExternalIsLeaf(t *testing.T) {
	snap := Snapshot{Nodes: []ServiceNode{
		{Name: "app", State: StateDiscovered, Out: []string{"db.example.com"}},
		{Name: "db.example.com", State: StateExternal, Out: []string{"app"}}, // would cycle if expanded
	}}
	ct := BuildCallTree(snap, "app", 0)
	// external peer contributes nothing to instrumentable coverage.
	if ct.Total != 0 || len(ct.Gaps) != 0 {
		t.Errorf("external peer should not count toward coverage: total=%d gaps=%v", ct.Total, ct.Gaps)
	}
}
