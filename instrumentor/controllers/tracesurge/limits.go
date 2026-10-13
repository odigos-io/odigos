package tracesurge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Trace surge limits bound how much more data surges send to the destinations, whatever the rules
// and spikes: a broad rule, or one that triggers on a traffic peak, must not raise sampling
// across a whole cluster.
const (
	defaultMaxActiveSurges     = 10
	defaultMaxBoostedWorkloads = 50
	defaultMaxDuration         = 30 * time.Minute

	maxDurationReason = "Raised for the maximum surge duration of"
)

type limits struct {
	maxActive   int
	maxBoosted  int
	maxDuration time.Duration
}

func limitsFrom(sampling *common.SamplingConfiguration) limits {
	l := limits{maxActive: defaultMaxActiveSurges, maxBoosted: defaultMaxBoostedWorkloads, maxDuration: defaultMaxDuration}
	if sampling == nil || sampling.TraceSurge == nil {
		return l
	}
	cfg := sampling.TraceSurge
	if cfg.MaxActiveSurges != nil && *cfg.MaxActiveSurges >= 0 {
		l.maxActive = *cfg.MaxActiveSurges
	}
	if cfg.MaxBoostedWorkloads != nil && *cfg.MaxBoostedWorkloads >= 0 {
		l.maxBoosted = *cfg.MaxBoostedWorkloads
	}
	if d, err := time.ParseDuration(cfg.MaxDuration); err == nil && d > 0 {
		l.maxDuration = d
	}
	return l
}

// limitReason returns which limit keeps a surge from raising the targets' sampling, or "".
// self is the surge asking, when it already exists.
func (e *Evaluator) limitReason(targets []odigosv1.TraceSurgeTarget, self *odigosv1.TraceSurge) string {
	active := 0
	boosted := map[k8sconsts.PodWorkload]bool{}
	for _, s := range e.active {
		if s == self || !s.Active() {
			continue
		}
		active++
		for _, t := range s.Spec.Targets {
			boosted[t.Workload] = true
		}
	}
	if e.limits.maxActive == 0 {
		return "Trace surges are turned off: sampling.traceSurge.maxActiveSurges is 0."
	}
	if active >= e.limits.maxActive {
		return fmt.Sprintf("%s already raising sampling, the limit (sampling.traceSurge.maxActiveSurges).", countOf(active, "trace surge"))
	}
	n := len(boosted)
	for _, t := range targets {
		if !boosted[t.Workload] {
			n++
		}
	}
	if n > e.limits.maxBoosted {
		return fmt.Sprintf("Raising %s would sample %s at a raised percentage, above the limit of %d (sampling.traceSurge.maxBoostedWorkloads).",
			targetNames(targets), countOf(n, "workload"), e.limits.maxBoosted)
	}
	return ""
}

func countOf(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// latestEnded returns the latest ended surge of each rule and service.
func latestEnded(ended []*odigosv1.TraceSurge) map[surgeKey]*odigosv1.TraceSurge {
	latest := map[surgeKey]*odigosv1.TraceSurge{}
	for _, s := range ended {
		key := surgeKey{s.Spec.SamplingName, s.Spec.RuleID, s.Spec.Service}
		if l := latest[key]; l == nil || s.StartedAt.After(l.StartedAt.Time) {
			latest[key] = s
		}
	}
	return latest
}

// latchedFrom returns the services whose latest surge ended at the maximum duration and that have
// not recovered since, so that a restarted evaluator keeps them from surging again before they do.
func latchedFrom(latest map[surgeKey]*odigosv1.TraceSurge, open map[surgeKey]*odigosv1.TraceSurge) map[surgeKey]bool {
	latched := map[surgeKey]bool{}
	for key, s := range latest {
		if _, ok := open[key]; !ok && s.Status.RecoveredAt == nil && strings.HasPrefix(s.Status.RestoreReason, maxDurationReason) {
			latched[key] = true
		}
	}
	return latched
}

// recordRecovery records on the surge that ended at the maximum duration that its service
// recovered, so that a restarted evaluator does not hold the service back again, and the surge
// can move to odigos insights.
func (e *Evaluator) recordRecovery(ctx context.Context, key surgeKey, metric string, observation *odigosv1.TraceSurgeObservation, now time.Time) {
	surge := e.ended[key]
	if surge == nil || surge.Status.RecoveredAt != nil {
		return
	}
	at := metav1.NewTime(now)
	surge.Status.RecoveredAt = &at
	addEvent(surge, now, fmt.Sprintf("%s back at %s", metricTitle(metric), formatValue(metric, observation.Value)),
		fmt.Sprintf("%s recovered after the surge ended at the maximum duration, and can surge again.", workloadLabel(key.service)))
	e.updateStatus(ctx, surge)
}
