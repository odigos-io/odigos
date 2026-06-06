package securitymetrics

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// Default thresholds for the volumetric detector. Tunable via NewVolumetricDetector.
const (
	defaultExfilFloorBps = 256 * 1024 // 256 KB/s sustained to a NEW external dest = exfil-shaped
	defaultSpikeFactor   = 4.0        // egress > 4× a service's rolling-average = a spike
	volEwmaTau           = 30 * time.Second
)

// VolumetricDetector flags two egress-volume anomalies that are the shape of data
// exfiltration: (1) sustained high-rate egress to a brand-new external destination, and
// (2) a sudden spike in a service's egress rate well above its own learned average. Both
// run purely on the byte rates already present on egress events — no new capture. It keeps
// a small per-service EWMA of egress rate (its only state), guarded by a mutex because the
// engine may call Inspect from one goroutine but the detector is shared.
type VolumetricDetector struct {
	exfilFloorBps float64
	spikeFactor   float64

	mu   sync.Mutex
	avg  map[string]float64   // service -> EWMA of total egress bytes/sec
	last map[string]time.Time // service -> last update (for time-aware EWMA)
}

// NewVolumetricDetector builds the detector. Zero args use sensible defaults; pass overrides
// for tuning. floorBps is the sustained-rate floor that makes egress to a new destination
// suspicious; spike is the multiple of a service's average that counts as a spike.
func NewVolumetricDetector(floorBps, spike float64) *VolumetricDetector {
	if floorBps <= 0 {
		floorBps = defaultExfilFloorBps
	}
	if spike <= 0 {
		spike = defaultSpikeFactor
	}
	return &VolumetricDetector{
		exfilFloorBps: floorBps,
		spikeFactor:   spike,
		avg:           map[string]float64{},
		last:          map[string]time.Time{},
	}
}

func (*VolumetricDetector) Name() string { return "volumetric" }

func (d *VolumetricDetector) Inspect(ev SecurityEvent, b *Baseline) []Finding {
	if ev.Cat != CategoryEgress || ev.Object.BytesPerSec <= 0 {
		return nil
	}
	svc := ev.Subject.Service
	bps := ev.Object.BytesPerSec

	// update the per-service egress-rate EWMA (time-aware)
	d.mu.Lock()
	prevAvg := d.avg[svc]
	alpha := 1.0
	if t, ok := d.last[svc]; ok {
		if gap := ev.Time.Sub(t).Seconds(); gap > 0 {
			alpha = 1 - expNeg(gap/volEwmaTau.Seconds())
		}
	}
	newAvg := prevAvg + alpha*(bps-prevAvg)
	d.avg[svc] = newAvg
	d.last[svc] = ev.Time
	d.mu.Unlock()

	var out []Finding

	// (1) sustained high-rate egress to a NEW external destination = exfil-shaped. We reuse
	// the drift baseline's first-seen knowledge: only fire when this dest is new-ish AND the
	// rate is over the floor. (The baseline already recorded it via the drift detector, so we
	// check membership without re-flagging — high volume to an established dest is normal.)
	if ev.Object.External && bps >= d.exfilFloorBps {
		dest := fmt.Sprintf("%s:%d", peerOrIP(ev.Object), ev.Object.Port)
		// only when the destination is recent (seen within the warm window): treat a
		// just-appeared high-volume external flow as the exfil signal.
		if first, _ := b.SeenExternalDest(svc, dest, ev.Time); ev.Time.Sub(first) < 2*time.Minute {
			out = append(out, Finding{
				ID:       findingID(CategoryEgress, ev.Subject, "exfil:"+dest),
				Time:     ev.Time,
				Severity: SeverityHigh,
				Cat:      CategoryEgress,
				Subject:  ev.Subject,
				Title:    fmt.Sprintf("possible exfiltration: %s → %s at %s", svc, dest, humanBps(bps)),
				Detail:   "sustained high-rate egress to a recently-first-seen external destination",
				Evidence: []SecurityEvent{ev},
				Actions:  pivotActions(ev.Subject),
			})
		}
	}

	// (2) egress spike vs the service's own learned average (a service suddenly sending far
	// more than usual). Require a meaningful prior average so we don't fire on cold start.
	if prevAvg >= 16*1024 && bps > d.spikeFactor*prevAvg {
		out = append(out, Finding{
			ID:       findingID(CategoryEgress, ev.Subject, "spike"),
			Time:     ev.Time,
			Severity: SeverityMedium,
			Cat:      CategoryEgress,
			Subject:  ev.Subject,
			Title:    fmt.Sprintf("egress spike: %s at %s (avg %s)", svc, humanBps(bps), humanBps(prevAvg)),
			Detail:   fmt.Sprintf("egress rate is %.1f× this service's recent average", bps/prevAvg),
			Evidence: []SecurityEvent{ev},
		})
	}
	return out
}

func humanBps(b float64) string {
	const k = 1024.0
	switch {
	case b < k:
		return fmt.Sprintf("%.0f B/s", b)
	case b < k*k:
		return fmt.Sprintf("%.1f KB/s", b/k)
	default:
		return fmt.Sprintf("%.1f MB/s", b/(k*k))
	}
}

func expNeg(x float64) float64 { return math.Exp(-x) }
