package securitymetrics

import "fmt"

// EgressDetector raises informational findings for every external destination a service
// talks to. It is the attack-surface inventory rendered as findings: "service X egresses to
// Y:port". Low-noise by design (info severity, deduped per destination) — the value is the
// complete, service-named picture of where data flows out.
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
