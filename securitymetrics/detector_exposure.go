package securitymetrics

import "fmt"

// sensitivePlaintextPorts are well-known cleartext services that are notable when listening
// on a wildcard (externally reachable) address — the classic posture finding.
var sensitivePlaintextPorts = map[int]string{
	23:    "telnet",
	21:    "ftp",
	80:    "http",
	3306:  "mysql",
	5432:  "postgres",
	6379:  "redis",
	9200:  "elasticsearch",
	11211: "memcached",
	27017: "mongodb",
}

// ExposureDetector turns LISTEN events into attack-surface findings. A service listening on
// a wildcard address (0.0.0.0 / ::) is externally reachable — higher severity than a
// loopback-only listener — and a cleartext service on a wildcard is higher still. This is
// the "what's exposed that shouldn't be" posture pillar, computed purely from listen sockets
// the endpoint resolver already reads.
type ExposureDetector struct{}

func (ExposureDetector) Name() string { return "exposure" }

func (ExposureDetector) Inspect(ev SecurityEvent, _ *Baseline) []Finding {
	if ev.Cat != CategoryExposure {
		return nil
	}
	wildcard, _ := ev.Attrs["wildcard"].(bool)
	port := ev.Object.Port

	sev := SeverityInfo
	title := fmt.Sprintf("%s listens on :%d (loopback)", ev.Subject.Service, port)
	detail := "loopback-only listener — not externally reachable"
	if wildcard {
		sev = SeverityLow
		title = fmt.Sprintf("%s exposed on 0.0.0.0:%d", ev.Subject.Service, port)
		detail = "listening on a wildcard address — reachable from off-host"
		if svc, ok := sensitivePlaintextPorts[port]; ok {
			sev = SeverityMedium
			detail = fmt.Sprintf("cleartext %s exposed on a wildcard address — reachable from off-host", svc)
		}
	}
	return []Finding{{
		ID:       findingID(CategoryExposure, ev.Subject, itoa(port)),
		Time:     ev.Time,
		Severity: sev,
		Cat:      CategoryExposure,
		Subject:  ev.Subject,
		Title:    title,
		Detail:   detail,
		Evidence: []SecurityEvent{ev},
	}}
}
