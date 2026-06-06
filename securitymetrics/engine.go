package securitymetrics

import (
	"context"
	"sort"
	"sync"
	"time"
)

// Engine is the source-agnostic security pipeline: it fans events from every registered
// Source through every registered Detector, and maintains a deduplicated, aggregated store
// of current Findings plus an egress inventory. Adding a runtime source later is just
// AddSource(...) — the detectors and store are untouched.
type Engine struct {
	sources   []Source
	detectors []Detector
	baseline  *Baseline
	onFinding func(Finding) // sink for new / severity-escalated findings (export)

	mu        sync.RWMutex
	findings  map[string]*Finding // keyed by Finding.ID
	inventory *Inventory          // egress inventory (built by the egress detector)
}

// NewEngine builds an engine over a baseline. Sources and detectors are registered before Run.
func NewEngine(baseline *Baseline) *Engine {
	return &Engine{
		baseline:  baseline,
		findings:  map[string]*Finding{},
		inventory: NewInventory(),
	}
}

// AddSource registers an event source (NetworkSource in the MVP; runtime sources later).
func (e *Engine) AddSource(s Source) *Engine { e.sources = append(e.sources, s); return e }

// AddDetector registers a detector. Order is preserved but detectors are independent.
func (e *Engine) AddDetector(d Detector) *Engine { e.detectors = append(e.detectors, d); return e }

// OnFinding registers a sink called once when a finding is first created and again only when
// its severity escalates — so the host can export findings (structured log + a JSONL file a
// SIEM tails) without re-emitting on every repeat sighting. The sink runs under the engine
// lock, so it must be quick and non-blocking.
func (e *Engine) OnFinding(fn func(Finding)) *Engine { e.onFinding = fn; return e }

// Baseline exposes the learned baseline (for persistence by the host).
func (e *Engine) Baseline() *Baseline { return e.baseline }

// Inventory exposes the egress inventory accumulated from events.
func (e *Engine) Inventory() *Inventory { return e.inventory }

// Run starts every source and processes their events through the detectors until ctx is
// cancelled. Each source runs in its own goroutine; events are merged and handled serially
// so detector/baseline access needs no extra locking beyond the engine's own.
func (e *Engine) Run(ctx context.Context) {
	merged := make(chan SecurityEvent, 256)
	var wg sync.WaitGroup
	for _, s := range e.sources {
		wg.Add(1)
		go func(s Source) {
			defer wg.Done()
			for ev := range s.Events(ctx) {
				if ev.Source == "" {
					ev.Source = s.Name()
				}
				select {
				case merged <- ev:
				case <-ctx.Done():
					return
				}
			}
		}(s)
	}
	go func() { wg.Wait(); close(merged) }()

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-merged:
			if !ok {
				return
			}
			e.handle(ev)
		}
	}
}

// handle feeds one event to the inventory and every detector, folding resulting findings
// into the store (dedup by ID, bump Count + Time, merge bounded evidence).
func (e *Engine) handle(ev SecurityEvent) {
	e.inventory.Observe(ev)
	for _, d := range e.detectors {
		for _, f := range d.Inspect(ev, e.baseline) {
			e.upsert(f)
		}
	}
}

const maxEvidencePerFinding = 10

func (e *Engine) upsert(f Finding) {
	e.mu.Lock()
	var emit *Finding // set to a finding to export after the lock is released
	if existing, ok := e.findings[f.ID]; ok {
		existing.Count++
		existing.Time = f.Time
		if f.Severity > existing.Severity {
			existing.Severity = f.Severity
			cp := *existing
			emit = &cp // export on severity escalation
		}
		if len(existing.Evidence) < maxEvidencePerFinding {
			existing.Evidence = append(existing.Evidence, f.Evidence...)
			if len(existing.Evidence) > maxEvidencePerFinding {
				existing.Evidence = existing.Evidence[:maxEvidencePerFinding]
			}
		}
	} else {
		f.Count = 1
		fc := f
		e.findings[f.ID] = &fc
		emit = &fc // export on first sighting
	}
	e.mu.Unlock()
	if emit != nil && e.onFinding != nil {
		e.onFinding(*emit)
	}
}

// Report is the security snapshot the host serves (over /api/security) and the TUI renders.
type Report struct {
	Time      time.Time    `json:"time"`
	Warming   bool         `json:"warming"`   // baseline still learning (drift muted)
	Findings  []Finding    `json:"findings"`  // severity desc, then most-recent
	Inventory []EgressItem `json:"inventory"` // per-service external destinations
	Totals    ReportTotals `json:"totals"`
}

// ReportTotals is the at-a-glance header for the security view.
type ReportTotals struct {
	Findings     int `json:"findings"`
	Critical     int `json:"critical"`
	High         int `json:"high"`
	Medium       int `json:"medium"`
	ExternalDeps int `json:"external_deps"`
}

// Report builds the current security report: sorted findings + egress inventory + totals.
func (e *Engine) Report() Report {
	e.mu.RLock()
	out := make([]Finding, 0, len(e.findings))
	for _, f := range e.findings {
		out = append(out, *f)
	}
	e.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity > out[j].Severity
		}
		return out[i].Time.After(out[j].Time)
	})

	var t ReportTotals
	t.Findings = len(out)
	for _, f := range out {
		switch f.Severity {
		case SeverityCritical:
			t.Critical++
		case SeverityHigh:
			t.High++
		case SeverityMedium:
			t.Medium++
		}
	}
	inv := e.inventory.Items()
	t.ExternalDeps = len(inv)

	return Report{
		Time:      time.Now(),
		Warming:   e.baseline.Warming(),
		Findings:  out,
		Inventory: inv,
		Totals:    t,
	}
}
