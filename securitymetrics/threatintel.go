package securitymetrics

import (
	"net"
	"strings"
)

// IPIntel is the pluggable IP-reputation/geo provider. The MVP ships a lightweight offline
// implementation (CIDR denylist + cloud/bogon classification); a real GeoIP/ASN database
// (MaxMind, etc.) drops in behind this same interface with no detector change — mirroring
// how runtime Sources compose in. Lookups MUST be offline (a security agent should never
// phone home), so implementations are expected to be backed by a bundled DB or static data.
type IPIntel interface {
	// Lookup returns reputation/geo facts for an IP (best-effort; zero value if unknown).
	Lookup(ip net.IP) IPFacts
}

// IPFacts is what an IPIntel provider knows about an address.
type IPFacts struct {
	Denylisted bool   // matches an operator denylist (known-bad)
	Country    string // ISO country code, if known (GeoIP provider)
	ASN        string // autonomous system, if known
	Note       string // human label for the match (e.g. "denylist: C2 feed")
}

// staticIntel is the default offline provider: an operator-supplied denylist of CIDRs plus
// a small set of always-suspicious ranges (bogons reaching the public side, etc). No DB, no
// network. Swap in a GeoIP-backed IPIntel for country/ASN enrichment.
type staticIntel struct {
	deny  []*net.IPNet
	notes map[string]string // cidr string -> note
}

// NewStaticIntel builds the default provider from a list of denylisted CIDRs/IPs (an IP is
// treated as a /32 or /128). Invalid entries are skipped. Pass nil for an empty denylist
// (the detector still flags sensitive-port egress to public IPs).
func NewStaticIntel(denylist []string) IPIntel {
	s := &staticIntel{notes: map[string]string{}}
	for _, e := range denylist {
		cidr := e
		if !strings.Contains(cidr, "/") {
			if strings.Contains(cidr, ":") {
				cidr += "/128"
			} else {
				cidr += "/32"
			}
		}
		if _, n, err := net.ParseCIDR(cidr); err == nil {
			s.deny = append(s.deny, n)
			s.notes[n.String()] = "denylist"
		}
	}
	return s
}

func (s *staticIntel) Lookup(ip net.IP) IPFacts {
	for _, n := range s.deny {
		if n.Contains(ip) {
			return IPFacts{Denylisted: true, Note: "denylist: " + s.notes[n.String()]}
		}
	}
	return IPFacts{}
}

// isPublicIP reports whether an IP is a routable public address (not private/loopback/
// link-local/CGNAT/multicast) — used to decide whether a sensitive-port egress is leaving
// the trusted network.
func isPublicIP(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsPrivate() {
		return false
	}
	// 100.64.0.0/10 CGNAT
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
		return false
	}
	return true
}
