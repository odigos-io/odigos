package securitymetrics

import (
	"sync"
	"time"
)

// Baseline is the learned "normal" the drift detector diffs against: which service→peer
// edges, which external destinations, and which listening ports have been seen, plus when
// each was first observed. It is source-agnostic — runtime detectors later baseline
// processes/files in the same store using the same first-seen semantics.
//
// A warm-up window guards against alert storms on cold start: until learnUntil passes,
// everything observed is recorded as baseline rather than flagged as new (mirrors the
// netmetrics rate sample-ring warm-up). Persisting the maps across restarts (Snapshot/
// Restore) avoids re-learning every boot.
type Baseline struct {
	mu         sync.RWMutex
	edges      map[string]time.Time // "service\x00peerService" -> first seen
	extDest    map[string]time.Time // "service\x00peerHostOrIP:port" -> first seen
	listens    map[string]time.Time // "service\x00port" -> first seen
	learnUntil time.Time            // during warm-up, observations are learned, not flagged
}

// NewBaseline creates an empty baseline with the given warm-up window (e.g. 60s): for that
// long after start, observations are recorded silently so drift only fires on genuinely new
// activity once "normal" is established.
func NewBaseline(warmup time.Duration) *Baseline {
	return &Baseline{
		edges:      map[string]time.Time{},
		extDest:    map[string]time.Time{},
		listens:    map[string]time.Time{},
		learnUntil: time.Now().Add(warmup),
	}
}

// Warming reports whether the baseline is still in its warm-up window.
func (b *Baseline) Warming() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return time.Now().Before(b.learnUntil)
}

// seen checks-or-records a key in the given map. Returns isNew=true only when the key was
// previously unseen AND the warm-up window has passed (so cold-start observations seed the
// baseline instead of alerting). Either way the key ends up recorded with its first-seen time.
func (b *Baseline) seen(m map[string]time.Time, key string, now time.Time) (firstSeen time.Time, isNew bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if t, ok := m[key]; ok {
		return t, false
	}
	m[key] = now
	return now, !now.Before(b.learnUntil)
}

// SeenEdge records a service→peer edge; isNew is true only for a genuinely new edge after warm-up.
func (b *Baseline) SeenEdge(service, peer string, now time.Time) (time.Time, bool) {
	return b.seen(b.edges, service+"\x00"+peer, now)
}

// SeenExternalDest records a service→external destination (host/ip:port).
func (b *Baseline) SeenExternalDest(service, dest string, now time.Time) (time.Time, bool) {
	return b.seen(b.extDest, service+"\x00"+dest, now)
}

// SeenListen records a service listening on a port.
func (b *Baseline) SeenListen(service string, port int, now time.Time) (time.Time, bool) {
	return b.seen(b.listens, service+"\x00"+itoa(port), now)
}

// Snapshot returns copies of the baseline maps for persistence across restarts.
func (b *Baseline) Snapshot() (edges, extDest, listens map[string]time.Time) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return cloneMap(b.edges), cloneMap(b.extDest), cloneMap(b.listens)
}

// Restore loads previously-persisted baseline maps (e.g. on agent restart) so drift does not
// re-flag known-normal activity as new.
func (b *Baseline) Restore(edges, extDest, listens map[string]time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if edges != nil {
		b.edges = cloneMap(edges)
	}
	if extDest != nil {
		b.extDest = cloneMap(extDest)
	}
	if listens != nil {
		b.listens = cloneMap(listens)
	}
}

func cloneMap(m map[string]time.Time) map[string]time.Time {
	out := make(map[string]time.Time, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
