package securitymetrics

import (
	"sort"
	"sync"
	"time"
)

// EgressItem is one external destination a service talks to — the unit of the egress
// inventory, the #1 artifact a security/attack-surface review wants: "what does each
// service call out to, off-host."
type EgressItem struct {
	Service   string    `json:"service"`
	Peer      string    `json:"peer"` // resolved name or host
	PeerIP    string    `json:"peer_ip,omitempty"`
	Port      int       `json:"port"`
	Transport string    `json:"transport"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// Inventory accumulates the distinct external destinations per service from the event
// stream. It is the egress attack-surface map; the egress detector reads from the same
// observations to raise findings.
type Inventory struct {
	mu    sync.RWMutex
	items map[string]*EgressItem // key: service\x00peer\x00port\x00transport
}

func NewInventory() *Inventory { return &Inventory{items: map[string]*EgressItem{}} }

// Observe folds an egress event into the inventory (no-op for non-external / non-egress).
func (in *Inventory) Observe(ev SecurityEvent) {
	if ev.Cat != CategoryEgress || !ev.Object.External {
		return
	}
	key := ev.Subject.Service + "\x00" + ev.Object.PeerService + "\x00" + itoa(ev.Object.Port) + "\x00" + ev.Object.Transport
	in.mu.Lock()
	defer in.mu.Unlock()
	if it, ok := in.items[key]; ok {
		it.LastSeen = ev.Time
		return
	}
	in.items[key] = &EgressItem{
		Service:   ev.Subject.Service,
		Peer:      ev.Object.PeerService,
		PeerIP:    ev.Object.PeerIP,
		Port:      ev.Object.Port,
		Transport: ev.Object.Transport,
		FirstSeen: ev.Time,
		LastSeen:  ev.Time,
	}
}

// Items returns the inventory sorted by service then peer, for stable rendering.
func (in *Inventory) Items() []EgressItem {
	in.mu.RLock()
	defer in.mu.RUnlock()
	out := make([]EgressItem, 0, len(in.items))
	for _, it := range in.items {
		out = append(out, *it)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Service != out[j].Service {
			return out[i].Service < out[j].Service
		}
		if out[i].Peer != out[j].Peer {
			return out[i].Peer < out[j].Peer
		}
		return out[i].Port < out[j].Port
	})
	return out
}
