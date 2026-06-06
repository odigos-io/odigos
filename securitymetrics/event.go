// Package securitymetrics is the shared, source-agnostic security layer that sits on top
// of network (and, later, runtime) telemetry. Every signal — a network flow today, a
// process exec or syscall tomorrow — becomes the SAME SecurityEvent, resolved to the SAME
// Subject (service identity), and flows through the SAME Detectors into the SAME Findings.
//
// The bet: the only thing that differs per signal source is the Source string field. So
// composing a runtime layer later (Tetragon/Falco/OBI-L7) is a new Source adapter — the
// detectors, baseline, findings, and UI never change. Cross-source correlation falls out
// for free because a Finding's Evidence is a slice of SecurityEvents that can come from
// different sources but share one Subject.
//
// MVP is network-only and adds NO new eBPF: events are derived from the netmetrics
// Snapshot (see source_network.go), which the VM agent already produces.
package securitymetrics

import "time"

// Category groups events by the kind of activity. Network categories ship in the MVP;
// the runtime categories are reserved so detectors and the UI already understand them
// the day a runtime Source is composed in.
type Category string

const (
	// Network categories (MVP).
	CategoryEgress    Category = "egress"     // a local service connecting OUT to a peer
	CategoryExposure  Category = "exposure"   // a local service LISTENING (attack surface)
	CategoryFlowNew   Category = "flow.new"   // a service-to-service edge not seen before (drift)
	CategoryTCPHealth Category = "tcp.health" // per-edge rtt/retransmits/failed-conns (scan/RST signal)

	// Runtime categories (reserved — composed later, no pipeline change).
	CategoryProcessExec Category = "process.exec"
	CategoryFileOpen    Category = "file.open"
	CategorySyscall     Category = "syscall"
)

// Severity orders findings for the operator. Detectors assign it; the UI sorts by it.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityLow
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

func (s Severity) String() string {
	switch s {
	case SeverityCritical:
		return "critical"
	case SeverityHigh:
		return "high"
	case SeverityMedium:
		return "medium"
	case SeverityLow:
		return "low"
	default:
		return "info"
	}
}

// Subject is WHO an event is about — resolved to service identity ONCE, identically across
// every source. This is the moat: a network flow and a process exec on the same PID/service
// land on the same Subject, so their evidence correlates automatically.
type Subject struct {
	Service      string `json:"service"`                 // service.name — the correlation key
	PID          int    `json:"pid,omitempty"`           // optional
	Container    string `json:"container,omitempty"`     // optional
	WorkloadKind string `json:"workload_kind,omitempty"` // docker | systemd | process
	Instrumented bool   `json:"instrumented,omitempty"`  // already an Odigos Source (has traces)
	Eligible     bool   `json:"eligible,omitempty"`      // instrumentation-eligible (for the pivot)
}

// Object is WHAT an event acted on. Network events use the peer/port fields; runtime events
// (later) use Path/Binary/Syscall. One struct, source-tagged fields, so the UI renders any.
type Object struct {
	// Network (MVP).
	PeerService string  `json:"peer_service,omitempty"` // resolved peer name (or raw IP/host)
	PeerIP      string  `json:"peer_ip,omitempty"`
	Port        int     `json:"port,omitempty"`
	Transport   string  `json:"transport,omitempty"`     // tcp | udp
	External    bool    `json:"external,omitempty"`      // peer is off-host
	BytesPerSec float64 `json:"bytes_per_sec,omitempty"` // throughput on this flow (volumetric/exfil signal)

	// Runtime (reserved).
	Path    string `json:"path,omitempty"`    // file path
	Binary  string `json:"binary,omitempty"`  // executed binary
	Syscall string `json:"syscall,omitempty"` // syscall name
}

// SecurityEvent is the canonical, source-agnostic unit. A Source emits these; Detectors
// inspect them; Findings cite them as evidence.
type SecurityEvent struct {
	Time    time.Time      `json:"time"`
	Source  string         `json:"source"` // "network" (MVP) | "tetragon" | "falco" | "obi-l7" (later)
	Subject Subject        `json:"subject"`
	Cat     Category       `json:"category"`
	Verb    string         `json:"verb"` // connect | listen | exec | open
	Object  Object         `json:"object"`
	Attrs   map[string]any `json:"attrs,omitempty"` // raw source-specific detail
}
