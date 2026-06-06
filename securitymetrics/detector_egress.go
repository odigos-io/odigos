package securitymetrics

import "fmt"

// EgressDetector raises an informational finding for every external destination a service
// talks to. NOTE: in the default VM wiring this is NOT registered — the Engine already
// builds an egress Inventory (its own tab in the security view) from the same events, so
// registering this detector would duplicate that inventory into the Findings list as INFO
// noise. The security Findings list is reserved for actual security SIGNAL (drift,
// volumetric, threat-intel, exposure, tcp-health); benign egress is inventory, not a finding.
// The detector is kept for callers that explicitly want per-egress findings.
type EgressDetector struct{}

func (EgressDetector) Name() string { return "egress" }

func (EgressDetector) Inspect(ev SecurityEvent, _ *Baseline) []Finding {
	if ev.Cat != CategoryEgress || !ev.Object.External {
		return nil
	}
	objKey := fmt.Sprintf("%s:%d/%s", ev.Object.PeerService, ev.Object.Port, ev.Object.Transport)
	return []Finding{{
		ID:       findingID(CategoryEgress, ev.Subject, objKey),
		Time:     ev.Time,
		Severity: SeverityInfo,
		Cat:      CategoryEgress,
		Subject:  ev.Subject,
		Title:    fmt.Sprintf("%s → %s:%d", ev.Subject.Service, ev.Object.PeerService, ev.Object.Port),
		Detail:   fmt.Sprintf("external egress (%s) to %s", ev.Object.Transport, objKey),
		Evidence: []SecurityEvent{ev},
	}}
}
