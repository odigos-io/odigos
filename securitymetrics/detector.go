package securitymetrics

import (
	"context"
	"time"
)

// Source produces SecurityEvents. The MVP has exactly one (NetworkSource, from the
// netmetrics Snapshot). Runtime sources (Tetragon/Falco/OBI-L7) implement this same
// interface later — that is the entire extension point.
type Source interface {
	// Name identifies the source for the Source field on emitted events.
	Name() string
	// Events streams events until ctx is cancelled. May be poll-derived (network, from
	// successive snapshots) or push-derived (a runtime event stream).
	Events(ctx context.Context) <-chan SecurityEvent
}

// Finding is a security conclusion the operator should see. Evidence is a slice of the
// SecurityEvents that triggered it — and crucially those events may come from DIFFERENT
// sources sharing one Subject, which is how cross-source correlation surfaces with no
// pipeline change. Actions are operator next-steps the UI offers (e.g. "instrument").
type Finding struct {
	ID       string          `json:"id"`   // stable key: dedupes the same finding over time
	Time     time.Time       `json:"time"` // last update
	Severity Severity        `json:"severity"`
	Cat      Category        `json:"category"`
	Subject  Subject         `json:"subject"`
	Title    string          `json:"title"`  // one-line summary
	Detail   string          `json:"detail"` // human explanation
	Evidence []SecurityEvent `json:"evidence,omitempty"`
	Actions  []string        `json:"actions,omitempty"` // e.g. ["instrument"] — the Odigos pivot
	Count    int             `json:"count"`             // how many times observed (aggregation)
}

// Detector inspects one event against the learned Baseline and returns zero or more
// findings. Detectors are PURE functions of (event, baseline) — no I/O, fully unit-testable
// without a VM. A detector that understands network events today understands runtime events
// tomorrow without changing, because both are SecurityEvents.
type Detector interface {
	Name() string
	Inspect(ev SecurityEvent, b *Baseline) []Finding
}

// findingID builds a stable identifier so the same logical finding (same subject + category
// + object) updates in place instead of spawning duplicates as events repeat.
func findingID(cat Category, subj Subject, objKey string) string {
	return string(cat) + "|" + subj.Service + "|" + objKey
}
