package tracesurge

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Default trace surge limits: they keep a broad rule, or one that triggers on a traffic peak,
// from raising sampling across the whole cluster.
const (
	defaultMaxActiveSurges     = 10
	defaultMaxBoostedWorkloads = 50
	defaultMaxDuration         = 30 * time.Minute
	// higher limits are taken as these: every active surge is written to the Sampling status.
	maxMaxActiveSurges     = 100
	maxMaxBoostedWorkloads = 1000

	maxDurationReason = "Raised for the maximum surge duration of"
	// a surge that ended at the maximum duration holds its service back at most this long.
	maxHoldBack = 24 * time.Hour
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
		l.maxActive = min(*cfg.MaxActiveSurges, maxMaxActiveSurges)
	}
	if cfg.MaxBoostedWorkloads != nil && *cfg.MaxBoostedWorkloads >= 0 {
		l.maxBoosted = min(*cfg.MaxBoostedWorkloads, maxMaxBoostedWorkloads)
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
			someTargetNames(targets, namedTargets), countOf(n, "workload"), e.limits.maxBoosted)
	}
	return ""
}

// boostedWorkloads returns the workloads the active surges raise.
func (e *Evaluator) boostedWorkloads() map[k8sconsts.PodWorkload]bool {
	boosted := map[k8sconsts.PodWorkload]bool{}
	for _, s := range e.active {
		if s.Active() {
			for _, t := range s.Spec.Targets {
				boosted[t.Workload] = true
			}
		}
	}
	return boosted
}

// enforceLimits restores the newest active surges beyond maxActiveSurges, then the newest ones
// until the workloads they raise fit maxBoostedWorkloads: either limit may have been lowered since
// the surges started.
func (e *Evaluator) enforceLimits(ctx context.Context, now time.Time, active map[surgeKey]*odigosv1.TraceSurge) {
	var raising []surgeKey
	for key, s := range active {
		if s.Active() {
			raising = append(raising, key)
		}
	}
	slices.SortFunc(raising, func(a, b surgeKey) int {
		if c := active[a].Status.BoostedAt.Compare(active[b].Status.BoostedAt.Time); c != 0 {
			return c
		}
		return strings.Compare(active[a].Name, active[b].Name)
	})
	end := func(reason string) {
		key := raising[len(raising)-1]
		raising = raising[:len(raising)-1]
		e.restore(ctx, active[key], now, reason)
		delete(active, key)
	}

	reason := fmt.Sprintf("sampling.traceSurge.maxActiveSurges was lowered to %d while more surges raised sampling: the newest ones ended.", e.limits.maxActive)
	if e.limits.maxActive == 0 {
		reason = "Trace surges are turned off: sampling.traceSurge.maxActiveSurges is 0."
	}
	for len(raising) > e.limits.maxActive {
		end(reason)
	}

	// the number of surges that raise each workload.
	raisedBy := map[k8sconsts.PodWorkload]int{}
	for _, key := range raising {
		for _, t := range active[key].Spec.Targets {
			raisedBy[t.Workload]++
		}
	}
	reason = fmt.Sprintf("sampling.traceSurge.maxBoostedWorkloads was lowered to %d while the surges raised more workloads: the newest ones ended.", e.limits.maxBoosted)
	for len(raisedBy) > e.limits.maxBoosted {
		for _, t := range active[raising[len(raising)-1]].Spec.Targets {
			if raisedBy[t.Workload]--; raisedBy[t.Workload] == 0 {
				delete(raisedBy, t.Workload)
			}
		}
		end(reason)
	}
}

// roomReason returns why the Sampling status has no room for the surge to raise sampling, or "":
// the status always keeps the active surges, and they must fit in maxStatusBytes.
func (e *Evaluator) roomReason(surge *odigosv1.TraceSurge) string {
	if e.requiredBytes[surge.Spec.SamplingName]+surgeSize(surge) > maxStatusBytes {
		return fmt.Sprintf("The Sampling status has no room for another surge that raises sampling: the surges raising sampling would take more than %d KiB of it.", maxStatusBytes>>10)
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
		if _, ok := open[key]; !ok && endedAtMaxDuration(s) {
			latched[key] = true
		}
	}
	return latched
}

// endedAtMaxDuration reports whether the surge ended at the maximum duration, and its service has
// neither recovered nor been released since.
func endedAtMaxDuration(s *odigosv1.TraceSurge) bool {
	return s.Status.Phase == odigosv1.TraceSurgePhaseRestored && s.Status.RecoveredAt == nil && s.Status.HoldReleasedAt == nil &&
		strings.HasPrefix(s.Status.RestoreReason, maxDurationReason)
}

// holdsBack reports whether the surge ended at the maximum duration and still holds its service
// back: the service must recover before it can surge again.
func (e *Evaluator) holdsBack(s *odigosv1.TraceSurge) bool {
	return endedAtMaxDuration(s) && e.latched[keyOf(s)]
}

// releaseHolds lets the services that a surge which ended at the maximum duration holds back surge
// again without recovering, once the rule or the service is gone, or after maxHoldBack. workloads
// are the instrumented workloads, or nil when not known.
func (e *Evaluator) releaseHolds(ctx context.Context, now time.Time, rules map[[2]string]surgeRule, workloads map[k8sconsts.PodWorkload][]common.ProgrammingLanguage) {
	for key := range e.latched {
		_, instrumented := workloads[key.service]
		surge := e.ended[key]
		switch {
		case !hasRule(rules, key):
			e.releaseHold(ctx, now, key, "The rule was removed, disabled, or no longer has a surge.")
		case workloads != nil && !instrumented:
			e.releaseHold(ctx, now, key, fmt.Sprintf("%s is no longer instrumented.", workloadLabel(key.service)))
		case surge != nil && surge.Status.RestoredAt != nil && now.Sub(surge.Status.RestoredAt.Time) >= maxHoldBack:
			e.releaseHold(ctx, now, key, fmt.Sprintf("%s has not recovered in the %s since the surge ended.", workloadLabel(key.service), maxHoldBack))
		}
	}
}

func hasRule(rules map[[2]string]surgeRule, key surgeKey) bool {
	_, ok := rules[[2]string{key.sampling, key.ruleID}]
	return ok
}

// releaseHold lets the service surge again without recovering first, also after a restart, and the
// surge that held it back leave the Sampling status.
func (e *Evaluator) releaseHold(ctx context.Context, now time.Time, key surgeKey, reason string) {
	delete(e.latched, key)
	if surge := e.ended[key]; surge != nil {
		at := metav1.NewTime(now)
		surge.Status.HoldReleasedAt = &at
		addEvent(surge, now, "No longer held back", fmt.Sprintf("%s %s can surge again without recovering first.", reason, workloadLabel(key.service)))
		e.updateStatus(ctx, surge)
	}
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
