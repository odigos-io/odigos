package profiles

import "sync/atomic"

// IngestGate toggles whether OTLP profile batches are written to the in-memory profile store.
type IngestGate struct {
	enabled atomic.Bool
}

func NewProfilesIngestGate(val bool) *IngestGate {
	g := &IngestGate{}
	g.enabled.Store(val)
	return g
}

func (g *IngestGate) Set(val bool) {
	g.enabled.Store(val)
}

func (g *IngestGate) IsEnabled() bool {
	return g.enabled.Load()
}
