package securitymetrics

import (
	"fmt"
	"net"
)

// adminPorts are remote-admin / lateral-movement ports that are notable when a service
// egresses to a PUBLIC address (a workload calling out to SSH/RDP/etc. off-network is a
// classic compromise signal).
var adminPorts = map[int]string{
	22:   "ssh",
	23:   "telnet",
	3389: "rdp",
	5900: "vnc",
	445:  "smb",
	135:  "msrpc",
}

// ThreatIntelDetector enriches external egress with IP reputation/geo facts and raises
// findings for: (1) egress to a denylisted address (highest signal), and (2) egress to a
// PUBLIC address on a remote-admin port (lateral-movement / C2 shape). It is GeoIP-ready —
// give it a GeoIP-backed IPIntel and it will additionally carry country/ASN on findings —
// but ships offline by default (denylist + public/private classification), so a security
// agent never makes external calls.
type ThreatIntelDetector struct {
	intel IPIntel
}

// NewThreatIntelDetector builds the detector over an IPIntel provider (NewStaticIntel for the
// offline default, or a GeoIP-backed provider).
func NewThreatIntelDetector(intel IPIntel) *ThreatIntelDetector {
	if intel == nil {
		intel = NewStaticIntel(nil)
	}
	return &ThreatIntelDetector{intel: intel}
}

func (*ThreatIntelDetector) Name() string { return "threatintel" }

func (d *ThreatIntelDetector) Inspect(ev SecurityEvent, _ *Baseline) []Finding {
	if ev.Cat != CategoryEgress || !ev.Object.External {
		return nil
	}
	ip := net.ParseIP(ev.Object.PeerIP)
	if ip == nil {
		// peer is a name (already resolved); reputation lookup needs an IP. Names still
		// flow through the other detectors; threat-intel only acts on routable IPs.
		ip = net.ParseIP(ev.Object.PeerService)
	}
	if ip == nil {
		return nil
	}

	dest := fmt.Sprintf("%s:%d", peerOrIP(ev.Object), ev.Object.Port)
	facts := d.intel.Lookup(ip)

	// (1) denylisted destination — the strongest signal.
	if facts.Denylisted {
		return []Finding{{
			ID:       findingID(CategoryEgress, ev.Subject, "deny:"+dest),
			Time:     ev.Time,
			Severity: SeverityCritical,
			Cat:      CategoryEgress,
			Subject:  ev.Subject,
			Title:    fmt.Sprintf("egress to known-bad address: %s → %s", ev.Subject.Service, dest),
			Detail:   "destination matches the threat-intel denylist (" + facts.Note + ")" + geoSuffix(facts),
			Evidence: []SecurityEvent{ev},
			Actions:  pivotActions(ev.Subject),
		}}
	}

	// (2) remote-admin port to a public address — lateral-movement / C2 shape.
	if svc, ok := adminPorts[ev.Object.Port]; ok && isPublicIP(ip) {
		return []Finding{{
			ID:       findingID(CategoryEgress, ev.Subject, "admin:"+dest),
			Time:     ev.Time,
			Severity: SeverityHigh,
			Cat:      CategoryEgress,
			Subject:  ev.Subject,
			Title:    fmt.Sprintf("%s egress to public %s: %s → %s", ev.Subject.Service, svc, ev.Subject.Service, dest),
			Detail:   fmt.Sprintf("outbound %s to a public address — lateral-movement / C2 shape%s", svc, geoSuffix(facts)),
			Evidence: []SecurityEvent{ev},
			Actions:  pivotActions(ev.Subject),
		}}
	}
	return nil
}

func geoSuffix(f IPFacts) string {
	if f.Country == "" && f.ASN == "" {
		return ""
	}
	s := " ["
	if f.Country != "" {
		s += "country=" + f.Country
	}
	if f.ASN != "" {
		if f.Country != "" {
			s += " "
		}
		s += "asn=" + f.ASN
	}
	return s + "]"
}
