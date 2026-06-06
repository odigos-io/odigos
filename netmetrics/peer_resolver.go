package netmetrics

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

// peerResolver turns a bare remote IP into a friendlier display name and decides
// whether an address belongs to this host. It exists so the network map shows
// "ec2-x.compute.amazonaws.com" or "<hostname>" instead of a wall of raw IPs.
//
//   - Host IPs (every address on this machine's interfaces, plus loopback) are
//     recognized so host-side traffic we could not attribute to a PID collapses
//     under a single host node instead of appearing as a mystery external peer.
//   - Off-host IPs get a reverse-DNS name, resolved in the background and cached,
//     so Build never blocks on DNS: the first sighting shows the IP, later builds
//     show the resolved name.
type peerResolver struct {
	host    string
	hostIPs map[string]struct{}

	mu       sync.Mutex
	rdns     map[string]string   // ip -> resolved name ("" = resolved to nothing)
	inflight map[string]struct{} // ip currently being looked up
}

func newPeerResolver(host string) *peerResolver {
	p := &peerResolver{
		host:     host,
		hostIPs:  map[string]struct{}{},
		rdns:     map[string]string{},
		inflight: map[string]struct{}{},
	}
	// loopback always counts as this host.
	for _, ip := range []string{"127.0.0.1", "::1"} {
		p.hostIPs[ip] = struct{}{}
	}
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				p.hostIPs[ipnet.IP.String()] = struct{}{}
			}
		}
	}
	return p
}

// isHostIP reports whether addr belongs to this machine: an exact interface address, OR
// any loopback (127.0.0.0/8, ::1) — which includes the systemd-resolved stub 127.0.0.53 —
// OR a link-local address (169.254.0.0/16, fe80::/10). Loopback/link-local are never
// "external" peers; treating them as such would flag every localhost service call and the
// cloud metadata endpoint as external egress.
func (p *peerResolver) isHostIP(addr string) bool {
	if _, ok := p.hostIPs[addr]; ok {
		return true
	}
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsLinkLocalUnicast()
}

// pretty returns a display name for an off-host peer address: its cached reverse-DNS
// name if known, otherwise the raw IP (while a background lookup populates the cache
// for next time). Non-IP inputs (already-named peers) are returned unchanged.
func (p *peerResolver) pretty(addr string) string {
	if net.ParseIP(addr) == nil {
		return addr // already a name, not an IP
	}
	p.mu.Lock()
	if name, ok := p.rdns[addr]; ok {
		p.mu.Unlock()
		if name != "" {
			return name
		}
		return addr
	}
	if _, busy := p.inflight[addr]; !busy {
		p.inflight[addr] = struct{}{}
		go p.lookup(addr)
	}
	p.mu.Unlock()
	return addr
}

func (p *peerResolver) lookup(addr string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var name string
	if names, err := net.DefaultResolver.LookupAddr(ctx, addr); err == nil && len(names) > 0 {
		name = strings.TrimSuffix(names[0], ".")
	}
	p.mu.Lock()
	p.rdns[addr] = name
	delete(p.inflight, addr)
	p.mu.Unlock()
}
