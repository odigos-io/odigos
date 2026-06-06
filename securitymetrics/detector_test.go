package securitymetrics

import (
	"testing"
	"time"
)

func egressEvent(service, peer string, port int, external bool, eligible, instrumented bool) SecurityEvent {
	return SecurityEvent{
		Time:    time.Now(),
		Source:  "network",
		Subject: Subject{Service: service, Eligible: eligible, Instrumented: instrumented},
		Cat:     CategoryEgress,
		Verb:    "connect",
		Object:  Object{PeerService: peer, Port: port, Transport: "tcp", External: external},
	}
}

func listenEvent(service string, port int, wildcard bool) SecurityEvent {
	return SecurityEvent{
		Time:    time.Now(),
		Source:  "network",
		Subject: Subject{Service: service},
		Cat:     CategoryExposure,
		Verb:    "listen",
		Object:  Object{Port: port, Transport: "tcp"},
		Attrs:   map[string]any{"wildcard": wildcard},
	}
}

func TestEgressDetector_OnlyExternal(t *testing.T) {
	d := EgressDetector{}
	if f := d.Inspect(egressEvent("checkout", "redis", 6379, false, false, false), nil); len(f) != 0 {
		t.Errorf("internal egress should produce no egress finding, got %d", len(f))
	}
	f := d.Inspect(egressEvent("checkout", "api.stripe.com", 443, true, false, false), nil)
	if len(f) != 1 || f[0].Severity != SeverityInfo || f[0].Cat != CategoryEgress {
		t.Fatalf("external egress should yield 1 info egress finding, got %+v", f)
	}
	if f[0].Subject.Service != "checkout" {
		t.Errorf("subject wrong: %q", f[0].Subject.Service)
	}
}

func TestExposureDetector_Severity(t *testing.T) {
	d := ExposureDetector{}
	// loopback listener -> info
	if f := d.Inspect(listenEvent("svc", 8080, false), nil); f[0].Severity != SeverityInfo {
		t.Errorf("loopback listen should be info, got %s", f[0].Severity)
	}
	// wildcard non-sensitive port -> low
	if f := d.Inspect(listenEvent("svc", 8080, true), nil); f[0].Severity != SeverityLow {
		t.Errorf("wildcard listen should be low, got %s", f[0].Severity)
	}
	// wildcard cleartext-sensitive port (postgres) -> medium
	f := d.Inspect(listenEvent("db", 5432, true), nil)
	if f[0].Severity != SeverityMedium {
		t.Errorf("wildcard postgres should be medium, got %s", f[0].Severity)
	}
}

func TestDriftDetector_NewExternalDest(t *testing.T) {
	b := NewBaseline(0) // warm-up already over → new things flag immediately
	d := DriftDetector{}
	ev := egressEvent("coupon", "45.33.1.1", 443, true, true, false)
	ev.Object.PeerService = "45.33.1.1"

	first := d.Inspect(ev, b)
	if len(first) != 1 || first[0].Cat != CategoryFlowNew || first[0].Severity != SeverityMedium {
		t.Fatalf("first sighting of a new external dest should be a medium flow.new finding, got %+v", first)
	}
	if len(first[0].Actions) != 1 || first[0].Actions[0] != "instrument" {
		t.Errorf("eligible+uninstrumented subject should offer the instrument pivot, got %v", first[0].Actions)
	}
	// second identical sighting → no new finding (baseline now knows it)
	if again := d.Inspect(ev, b); len(again) != 0 {
		t.Errorf("repeat sighting must not re-flag, got %d findings", len(again))
	}
}

func TestDriftDetector_WarmupSuppresses(t *testing.T) {
	b := NewBaseline(time.Hour) // still warming → learn, don't flag
	d := DriftDetector{}
	ev := egressEvent("coupon", "45.33.1.1", 443, true, true, false)
	ev.Object.PeerService = "45.33.1.1"
	if f := d.Inspect(ev, b); len(f) != 0 {
		t.Errorf("during warm-up new activity must be learned silently, got %d findings", len(f))
	}
}

func TestDriftDetector_NewInternalEdge(t *testing.T) {
	b := NewBaseline(0)
	d := DriftDetector{}
	f := d.Inspect(egressEvent("frontend", "payments", 8443, false, false, false), b)
	if len(f) != 1 || f[0].Cat != CategoryFlowNew || f[0].Severity != SeverityLow {
		t.Fatalf("new internal edge should be a low flow.new finding, got %+v", f)
	}
}

func TestPivotActions_GatedOnEligibility(t *testing.T) {
	if a := pivotActions(Subject{Service: "x", Eligible: true, Instrumented: false}); len(a) != 1 {
		t.Error("eligible + uninstrumented should offer instrument")
	}
	if a := pivotActions(Subject{Service: "x", Eligible: true, Instrumented: true}); a != nil {
		t.Error("already-instrumented should not offer instrument")
	}
	if a := pivotActions(Subject{Service: "x", Eligible: false}); a != nil {
		t.Error("ineligible should not offer instrument")
	}
}
