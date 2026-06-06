package securitymetrics

import "fmt"

// DriftDetector is the core threat-detection primitive: it flags activity that is NEW
// relative to the learned baseline. In a service graph that is normally stable, a brand-new
// service→service edge, a first-seen external destination, or a new listening port is how
// lateral movement, C2 callbacks, supply-chain beacons, and misconfig actually show up.
//
// It is the same code for runtime drift later: a "new binary executed" event is just another
// SecurityEvent whose Subject+Object was never baselined. The detector does not change.
type DriftDetector struct{}

func (DriftDetector) Name() string { return "drift" }

func (DriftDetector) Inspect(ev SecurityEvent, b *Baseline) []Finding {
	switch ev.Cat {
	case CategoryEgress:
		// New external destination for this service (the exfil/C2-shaped signal).
		if ev.Object.External {
			dest := fmt.Sprintf("%s:%d", peerOrIP(ev.Object), ev.Object.Port)
			if _, isNew := b.SeenExternalDest(ev.Subject.Service, dest, ev.Time); isNew {
				return []Finding{{
					ID:       findingID(CategoryFlowNew, ev.Subject, "extdest:"+dest),
					Time:     ev.Time,
					Severity: SeverityMedium,
					Cat:      CategoryFlowNew,
					Subject:  ev.Subject,
					Title:    fmt.Sprintf("new external egress: %s → %s", ev.Subject.Service, dest),
					Detail:   "first time this service has connected to this external destination",
					Evidence: []SecurityEvent{ev},
					Actions:  pivotActions(ev.Subject),
				}}
			}
		} else {
			// New internal service→service edge (the lateral-movement-shaped signal).
			peer := ev.Object.PeerService
			if peer != "" {
				if _, isNew := b.SeenEdge(ev.Subject.Service, peer, ev.Time); isNew {
					return []Finding{{
						ID:       findingID(CategoryFlowNew, ev.Subject, "edge:"+peer),
						Time:     ev.Time,
						Severity: SeverityLow,
						Cat:      CategoryFlowNew,
						Subject:  ev.Subject,
						Title:    fmt.Sprintf("new internal edge: %s → %s", ev.Subject.Service, peer),
						Detail:   "first time this service has talked to this peer",
						Evidence: []SecurityEvent{ev},
						Actions:  pivotActions(ev.Subject),
					}}
				}
			}
		}
	case CategoryExposure:
		// New listening port for this service (new surface appearing at runtime).
		if _, isNew := b.SeenListen(ev.Subject.Service, ev.Object.Port, ev.Time); isNew {
			return []Finding{{
				ID:       findingID(CategoryFlowNew, ev.Subject, "listen:"+itoa(ev.Object.Port)),
				Time:     ev.Time,
				Severity: SeverityLow,
				Cat:      CategoryFlowNew,
				Subject:  ev.Subject,
				Title:    fmt.Sprintf("new listening port: %s :%d", ev.Subject.Service, ev.Object.Port),
				Detail:   "first time this service has been observed listening on this port",
				Evidence: []SecurityEvent{ev},
			}}
		}
	}
	return nil
}

// pivotActions offers the Odigos move — instrument the subject to see the actual request —
// only when it is an eligible, not-yet-instrumented local service.
func pivotActions(s Subject) []string {
	if s.Eligible && !s.Instrumented && s.Service != "" {
		return []string{"instrument"}
	}
	return nil
}

func peerOrIP(o Object) string {
	if o.PeerService != "" {
		return o.PeerService
	}
	return o.PeerIP
}
