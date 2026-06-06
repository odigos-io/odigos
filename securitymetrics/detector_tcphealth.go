package securitymetrics

import "fmt"

// Thresholds for the TCP-health detector (tunable via NewTCPHealthDetector).
const (
	defaultFailedConnScan = 20  // failed connections per poll from one service = scanning shape
	defaultRetxStorm      = 100 // retransmits per poll on one edge = a RST/retransmit storm
)

// TCPHealthDetector turns OBI's TCP-health deltas into security signals:
//   - a burst of FAILED connections from one service = port/host scanning or a service
//     hammering a dead dependency (recon / misconfig shape);
//   - a retransmit storm on an edge = network abuse / RST flood / saturation.
//
// These are the classic "health metrics as security signal" detections, riding entirely on
// stats OBI already computes — no new capture. Latency is surfaced (info) for ops context.
type TCPHealthDetector struct {
	failedConnScan float64
	retxStorm      float64
}

func NewTCPHealthDetector(failedConnScan, retxStorm float64) *TCPHealthDetector {
	if failedConnScan <= 0 {
		failedConnScan = defaultFailedConnScan
	}
	if retxStorm <= 0 {
		retxStorm = defaultRetxStorm
	}
	return &TCPHealthDetector{failedConnScan: failedConnScan, retxStorm: retxStorm}
}

func (*TCPHealthDetector) Name() string { return "tcp-health" }

func (d *TCPHealthDetector) Inspect(ev SecurityEvent, _ *Baseline) []Finding {
	if ev.Cat != CategoryTCPHealth {
		return nil
	}
	failed, _ := ev.Attrs["failed_conns_delta"].(float64)
	retx, _ := ev.Attrs["retransmits_delta"].(float64)

	var out []Finding

	// Failed-connection burst from a service = scanning / dead-dependency hammering.
	if failed >= d.failedConnScan {
		out = append(out, Finding{
			ID:       findingID(CategoryTCPHealth, ev.Subject, "scan"),
			Time:     ev.Time,
			Severity: SeverityMedium,
			Cat:      CategoryTCPHealth,
			Subject:  ev.Subject,
			Title:    fmt.Sprintf("connection-failure burst: %s (%.0f failed/poll)", ev.Subject.Service, failed),
			Detail:   "a spike of failed TCP connections — port/host scanning or a service hammering an unreachable dependency",
			Evidence: []SecurityEvent{ev},
			Actions:  pivotActions(ev.Subject),
		})
	}

	// Retransmit storm on an edge = network abuse / saturation / RST flood.
	if retx >= d.retxStorm {
		out = append(out, Finding{
			ID:       findingID(CategoryTCPHealth, ev.Subject, "retx:"+ev.Object.PeerService),
			Time:     ev.Time,
			Severity: SeverityLow,
			Cat:      CategoryTCPHealth,
			Subject:  ev.Subject,
			Title:    fmt.Sprintf("retransmit storm: %s → %s (%.0f/poll)", ev.Subject.Service, ev.Object.PeerService, retx),
			Detail:   "high TCP retransmits on this edge — saturation, packet loss, or RST flooding",
			Evidence: []SecurityEvent{ev},
		})
	}
	return out
}
