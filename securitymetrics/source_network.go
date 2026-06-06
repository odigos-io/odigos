package securitymetrics

import (
	"context"
	"time"

	"github.com/odigos-io/odigos/netmetrics"
)

// SnapshotFunc returns the current network snapshot. The VM agent passes the netmetrics
// SnapshotBuilder.Build here; tests pass a fake. This is the ONLY coupling between the
// security layer and the network layer — everything else is source-agnostic.
type SnapshotFunc func() (netmetrics.Snapshot, error)

// NetworkSource turns successive netmetrics Snapshots into SecurityEvents — the MVP's only
// Source, adding NO new capture. Each poll emits:
//   - one egress event per edge whose CLIENT is the local subject (Object = the server peer)
//   - one exposure event per service that is a server on an edge (it listens on ServerPort)
//
// Runtime sources (Tetragon/Falco/OBI-L7) implement the same Source interface later; this
// file is the template.
type NetworkSource struct {
	snap     SnapshotFunc
	interval time.Duration
}

// NewNetworkSource polls snap every interval and emits derived security events.
func NewNetworkSource(snap SnapshotFunc, interval time.Duration) *NetworkSource {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &NetworkSource{snap: snap, interval: interval}
}

// ephemeralPortFloor is the start of the Linux default ephemeral port range
// (/proc/sys/net/ipv4/ip_local_port_range is 32768–60999). A "listener" at or above this is
// almost certainly an OS-assigned client source port, not a real service, so exposure
// findings are suppressed for it to keep the attack-surface view signal-rich.
const ephemeralPortFloor = 32768

func (s *NetworkSource) Name() string { return "network" }

func (s *NetworkSource) Events(ctx context.Context) <-chan SecurityEvent {
	out := make(chan SecurityEvent, 128)
	go func() {
		defer close(out)
		t := time.NewTicker(s.interval)
		defer t.Stop()
		s.emit(ctx, out) // emit immediately, then on each tick
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.emit(ctx, out)
			}
		}
	}()
	return out
}

func (s *NetworkSource) emit(ctx context.Context, out chan<- SecurityEvent) {
	snap, err := s.snap()
	if err != nil {
		return
	}
	now := snap.Timestamp
	if now.IsZero() {
		now = time.Now()
	}
	byName := make(map[string]netmetrics.ServiceNode, len(snap.Nodes))
	for _, n := range snap.Nodes {
		byName[n.Name] = n
	}

	send := func(ev SecurityEvent) bool {
		select {
		case out <- ev:
			return true
		case <-ctx.Done():
			return false
		}
	}

	// Egress + drift: each edge is client -> server. The client is the local subject making
	// the outbound connection; the server is the Object. Mark External when the server node
	// is an external peer.
	for _, e := range snap.Edges {
		client, ok := byName[e.Client]
		if !ok || isExternalName(byName, e.Client) {
			// only emit egress for a LOCAL client (a process on this host initiating out)
			continue
		}
		server := byName[e.Server]
		external := server.State == netmetrics.StateExternal || server.Name == ""
		ev := SecurityEvent{
			Time:    now,
			Source:  "network",
			Subject: subjectOf(client),
			Cat:     CategoryEgress,
			Verb:    "connect",
			Object: Object{
				PeerService: e.Server,
				Port:        atoiPort(e.ServerPort),
				Transport:   e.Transport,
				External:    external,
			},
		}
		if !send(ev) {
			return
		}
	}

	// Exposure: a node that is the SERVER on any edge is listening. Emit one exposure event
	// per (service, serverPort). Wildcard vs loopback is inferred from the node not being
	// external and the peer reaching it (best-effort; the controller can pass the real
	// wildcard flag from the endpoint table for higher fidelity).
	seenListen := map[string]bool{}
	for _, e := range snap.Edges {
		server, ok := byName[e.Server]
		if !ok || server.State == netmetrics.StateExternal {
			continue
		}
		port := atoiPort(e.ServerPort)
		// Skip ephemeral ports: a "server" on an edge with a high ephemeral port is almost
		// always the OS-assigned source port of an outbound connection, not a real listener.
		// Real services listen on stable ports below the ephemeral range. This removes the
		// bulk of exposure noise (observed live on systemd hosts).
		if port == 0 || port >= ephemeralPortFloor {
			continue
		}
		key := e.Server + ":" + itoa(port)
		if seenListen[key] {
			continue
		}
		seenListen[key] = true
		// If the client is external/off-host, the listener is reachable from off-host →
		// treat as wildcard exposure; otherwise it is at least loopback-reachable.
		wildcard := isExternalName(byName, e.Client)
		ev := SecurityEvent{
			Time:    now,
			Source:  "network",
			Subject: subjectOf(server),
			Cat:     CategoryExposure,
			Verb:    "listen",
			Object:  Object{Port: port, Transport: e.Transport},
			Attrs:   map[string]any{"wildcard": wildcard},
		}
		if !send(ev) {
			return
		}
	}
}

func subjectOf(n netmetrics.ServiceNode) Subject {
	return Subject{
		Service:      n.Name,
		WorkloadKind: n.WorkloadKind,
		Instrumented: n.Instrumented,
		Eligible:     n.Eligible,
	}
}

func isExternalName(byName map[string]netmetrics.ServiceNode, name string) bool {
	n, ok := byName[name]
	return !ok || n.State == netmetrics.StateExternal
}

func atoiPort(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
