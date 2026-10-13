// Package tracesurge raises the head sampling of a noisy operation rule while a service its traces
// reach has a RED metric spike, and restores it once the metric recovers. It reads the metrics from
// odigos insights and records the surges in the Sampling status, which agentenabled applies.
package tracesurge

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/scope"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"go.opentelemetry.io/otel/metric"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// missing metrics never count as recovery, but end an active surge after this long.
	staleAfter = 10 * time.Minute
	// the most ended surges a Sampling status keeps while odigos insights is unreachable.
	maxEndedInStatus = 20
	// the most Limited surges a Sampling status keeps, the newest. A failing dependency can make
	// every service that calls it surge at once; the others are kept in memory only.
	maxLimitedInStatus = 10
	// the most the trace surges of a Sampling status take, well under the object size etcd
	// accepts (1.5 MiB), so that writing the status never fails for its size.
	maxStatusBytes = 512 << 10
	// when the full status cannot be written, the compact one keeps this many of the latest
	// events of a surge, and this many targets of an ended one.
	compactTimeline = 3
	compactTargets  = 10
	// back-off after a failed put to odigos insights, so an outage doesn't slow every evaluation.
	archiveRetryAfter = 30 * time.Second
	// the most each evaluation spends moving ended surges to odigos insights.
	archiveBudget   = 2 * time.Second
	maxTimeline     = 30
	maxTriggerTrail = 30
	// the most bytes of an event's description and of an observed operation's name.
	maxDescription = 512
	maxOperation   = 256
	// the most targets an event names.
	namedTargets = 10
)

// Evaluator runs on the leader instrumentor.
type Evaluator struct {
	Client client.Client
	Logger logr.Logger
	// nil records no metrics.
	Meter metric.Meter

	namespace string
	metrics   metricsReader
	// services whose metric is above the threshold of a rule, without an active surge yet.
	breaches map[surgeKey]*breach
	// surges already logged as having no workload to raise.
	noTargets      map[surgeKey]bool
	lastQueryErr   string
	lastArchiveErr string
	archiveRetryAt time.Time
	// the open surges (limited or active) of the current evaluation.
	active map[surgeKey]*odigosv1.TraceSurge
	limits limits
	// the window the current evaluation measures, and the time until the next one.
	window, interval time.Duration
	// services whose surge ended at the maximum duration: they must recover before surging again.
	latched map[surgeKey]bool
	// the latest ended surge of each rule and service, of the current evaluation.
	ended     map[surgeKey]*odigosv1.TraceSurge
	samplings map[string]*samplingState
	// the surges each Sampling status had no room for: evaluated with the ones in it, while this
	// instrumentor is the leader.
	unwritten map[string][]*odigosv1.TraceSurge
	// the bytes of the surges each Sampling status must keep in the current evaluation: the
	// active ones, including those it restores.
	requiredBytes map[string]int
	archive       archiver
	telemetry     *evaluatorMetrics
	// what the current evaluation read from odigos insights.
	servicesRead int
	queryErr     error
}

// samplingState is a Sampling and its surges during one evaluation; flush writes the dirty ones.
type samplingState struct {
	obj    *odigosv1.Sampling
	surges []*odigosv1.TraceSurge
	dirty  bool
}

var _ manager.LeaderElectionRunnable = &Evaluator{}

// metricsReader reads the span metrics and the service graph: see insightsMetrics.read.
type metricsReader interface {
	read(ctx context.Context, window time.Duration, minOperationCalls int) (map[k8sconsts.PodWorkload]*serviceMetrics, map[string][]string, error)
}

type surgeKey struct {
	sampling string
	ruleID   string
	service  k8sconsts.PodWorkload
}

type breach struct {
	since   time.Time
	samples []odigosv1.TraceSurgeObservation
}

type surgeRule struct {
	sampling string
	id       string
	op       odigosv1.NoisyOperation
}

func (r surgeRule) normalPercent() float64 {
	if r.op.PercentageAtMost == nil {
		return 0
	}
	return *r.op.PercentageAtMost
}

func (e *Evaluator) NeedLeaderElection() bool {
	return true
}

func (e *Evaluator) Start(ctx context.Context) error {
	e.init(env.GetCurrentNamespace(), newInsightsMetrics(env.GetCurrentNamespace()), newInsightsArchive(env.GetCurrentNamespace()))

	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}
		e.evaluateSafely(ctx, time.Now())
		// a timer, not a ticker: every evaluation may change the interval.
		timer.Reset(e.interval)
	}
}

// evaluateSafely runs one evaluation. A panic in it is logged: the evaluator runs in the
// instrumentor's manager, whose controllers and webhooks would all stop with it.
func (e *Evaluator) evaluateSafely(ctx context.Context, now time.Time) {
	defer func() {
		if r := recover(); r != nil {
			e.Logger.Error(fmt.Errorf("%v", r), "trace surge evaluation panicked", "stack", string(debug.Stack()))
		}
	}()
	if err := e.evaluate(ctx, now); err != nil {
		e.Logger.Error(err, "trace surge evaluation failed")
	}
}

func (e *Evaluator) init(namespace string, metrics metricsReader, archive archiver) {
	e.namespace = namespace
	e.metrics = metrics
	e.archive = archive
	e.breaches = map[surgeKey]*breach{}
	e.noTargets = map[surgeKey]bool{}
	e.window, e.interval = common.TraceSurgeTiming(nil)
	telemetry, err := newEvaluatorMetrics(e.Meter)
	if err != nil {
		e.Logger.Error(err, "failed to create the trace surge evaluator's metrics")
		telemetry, _ = newEvaluatorMetrics(nil)
	}
	e.telemetry = telemetry
}

// evaluate runs one evaluation and records how it went.
func (e *Evaluator) evaluate(ctx context.Context, now time.Time) error {
	began := time.Now()
	result, err := e.evaluateSurges(ctx, now)
	if err != nil {
		result = resultError
	}
	e.telemetry.evaluated(ctx, result, time.Since(began), now)
	return err
}

func (e *Evaluator) evaluateSurges(ctx context.Context, now time.Time) (string, error) {
	e.servicesRead, e.queryErr = 0, nil
	conf, err := k8sutils.GetCurrentOdigosConfiguration(ctx, e.Client)
	if err != nil {
		return "", err
	}
	e.limits = limitsFrom(conf.Sampling)
	e.window, e.interval = common.TraceSurgeTiming(conf.Sampling)

	samplings := &odigosv1.SamplingList{}
	if err := e.Client.List(ctx, samplings, client.InNamespace(e.namespace)); err != nil {
		return "", err
	}
	e.samplings = make(map[string]*samplingState, len(samplings.Items))
	for i := range samplings.Items {
		sampling := &samplings.Items[i]
		state := &samplingState{obj: sampling}
		names := make(map[string]bool, len(sampling.Status.TraceSurges))
		open := map[surgeKey]bool{}
		for j := range sampling.Status.TraceSurges {
			surge := sampling.Status.TraceSurges[j].DeepCopy()
			if repair(surge, now) {
				state.dirty = true
			}
			names[surge.Name] = true
			if surge.Status.Phase != odigosv1.TraceSurgePhaseRestored {
				open[keyOf(surge)] = true
			}
			state.surges = append(state.surges, surge)
		}
		for _, surge := range e.unwritten[sampling.Name] {
			if !names[surge.Name] && (surge.Status.Phase == odigosv1.TraceSurgePhaseRestored || !open[keyOf(surge)]) {
				state.surges = append(state.surges, surge)
			}
		}
		e.samplings[sampling.Name] = state
	}
	e.requiredBytes = make(map[string]int, len(e.samplings))
	for name, state := range e.samplings {
		for _, surge := range state.surges {
			if surge.Active() {
				e.requiredBytes[name] += surgeSize(surge)
			}
		}
	}

	rules := map[[2]string]surgeRule{}
	for _, sampling := range samplings.Items {
		for _, op := range sampling.Spec.NoisyOperations {
			if op.Surge != nil && !op.Disabled {
				id := odigosv1.ComputeNoisyOperationHash(&op)
				rules[[2]string{sampling.Name, id}] = surgeRule{sampling: sampling.Name, id: id, op: op}
			}
		}
	}

	active := map[surgeKey]*odigosv1.TraceSurge{}
	var ended []*odigosv1.TraceSurge
	for _, state := range e.samplings {
		for _, surge := range state.surges {
			if surge.Status.Phase == odigosv1.TraceSurgePhaseRestored {
				ended = append(ended, surge)
			} else if surge.Status.Phase != "" {
				active[keyOf(surge)] = surge
			}
		}
	}
	e.ended = latestEnded(ended)
	if e.latched == nil {
		e.latched = latchedFrom(e.ended, active)
	}

	e.active = active
	result := resultOK
	switch {
	case !common.TraceSurgesEnabled(conf.Sampling):
		clear(e.breaches)
		clear(rules)
		for _, surge := range active {
			e.restore(ctx, surge, now, causeTurnedOff, "Trace surges were turned off.")
		}
		for key := range e.latched {
			e.releaseHold(ctx, now, key, "Trace surges were turned off.")
		}
	case !common.InsightsPipelineActive(conf.Insights):
		result = resultInsightsOff
		clear(e.breaches)
		for _, surge := range active {
			e.restore(ctx, surge, now, causeInsightsOff, "Odigos Insights was turned off. Trace surges evaluate the RED metrics it stores.")
		}
		e.releaseHolds(ctx, now, rules, nil)
	case len(rules) > 0 || len(active) > 0:
		if err := e.evaluateRules(ctx, now, rules, active); err != nil {
			return "", err
		}
		switch {
		case errors.Is(e.queryErr, errInsightsTraceSurgesOff):
			result = resultInsightsSurgesOff
		case e.queryErr != nil:
			result = resultMetricsError
		}
	default:
		clear(e.breaches)
		e.releaseHolds(ctx, now, rules, nil)
	}

	// sampling is restored before ended surges move to odigos insights, however long that takes.
	e.flush(ctx)
	if e.archiveEnded(ctx, now) {
		e.flush(ctx)
	}
	e.telemetry.snapshot(rules, e.samplings, e.servicesRead, e.limits)
	return result, nil
}

func (e *Evaluator) evaluateRules(ctx context.Context, now time.Time, rules map[[2]string]surgeRule, active map[surgeKey]*odigosv1.TraceSurge) error {
	e.enforceLimits(ctx, now, active)

	// an operation is evaluated on its own only with the calls a rule requires, so fewer are not read.
	minOperationCalls := math.MaxInt
	for _, rule := range rules {
		minOperationCalls = min(minOperationCalls, rule.op.Surge.MinimumRequests)
	}
	for _, surge := range active {
		minOperationCalls = min(minOperationCalls, surge.Spec.Settings.MinimumRequests)
	}
	queryBegan := time.Now()
	services, callers, err := e.metrics.read(ctx, e.window, max(minOperationCalls, 1))
	e.telemetry.queried(ctx, time.Since(queryBegan), err)
	e.queryErr = err
	if err != nil {
		// without metrics no service is evaluated; active surges hold until they are stale.
		if err.Error() != e.lastQueryErr {
			if errors.Is(err, errInsightsTraceSurgesOff) {
				e.Logger.Error(err, "odigos insights has trace surges turned off while sampling.traceSurge.enabled is on: no service is evaluated until insights is deployed with them on")
			} else {
				e.Logger.Error(err, "failed to read span metrics from odigos insights")
			}
			e.lastQueryErr = err.Error()
		}
		services, callers = nil, nil
	} else {
		e.lastQueryErr = ""
		e.servicesRead = len(services)
	}

	languages, err := e.instrumentedWorkloads(ctx)
	if err != nil {
		return err
	}
	e.releaseHolds(ctx, now, rules, languages)
	byServiceName := map[string]k8sconsts.PodWorkload{}
	for pw, m := range services {
		byServiceName[m.key()] = pw
	}
	for pw := range languages {
		if key := serviceKey(pw.Namespace, pw.Name); byServiceName[key] == (k8sconsts.PodWorkload{}) {
			byServiceName[key] = pw
		}
	}
	callees := map[string][]string{}
	for server, clients := range callers {
		for _, client := range clients {
			callees[client] = append(callees[client], server)
		}
	}

	evaluated := map[surgeKey]bool{}
	for _, rule := range rules {
		inScope := map[k8sconsts.PodWorkload]bool{}
		for pw, langs := range languages {
			if slices.ContainsFunc(langs, func(l common.ProgrammingLanguage) bool {
				return scope.SourceScopeMatchesContainer(rule.op.SourceScopes, pw, l)
			}) {
				inScope[pw] = true
			}
		}
		for pw := range reachable(inScope, services, callees, byServiceName) {
			key := surgeKey{rule.sampling, rule.id, pw}
			evaluated[key] = true
			e.evaluateService(ctx, now, key, rule, services[pw], active[key], inScope, callers, services, byServiceName)
		}
	}

	for key, surge := range active {
		rule, ok := rules[[2]string{key.sampling, key.ruleID}]
		switch {
		case !ok:
			e.restore(ctx, surge, now, causeRuleRemoved, "The rule was removed, disabled, or no longer has a surge.")
		case !evaluated[key]:
			e.evaluateService(ctx, now, key, rule, nil, surge, nil, nil, nil, nil)
		}
	}
	for key := range e.breaches {
		if !evaluated[key] {
			delete(e.breaches, key)
		}
	}
	return nil
}

// evaluateService advances the surge of one rule for one service. m is nil when the service has no metrics.
func (e *Evaluator) evaluateService(ctx context.Context, now time.Time, key surgeKey, rule surgeRule, m *serviceMetrics, surge *odigosv1.TraceSurge,
	inScope map[k8sconsts.PodWorkload]bool, callers map[string][]string, services map[k8sconsts.PodWorkload]*serviceMetrics, byServiceName map[string]k8sconsts.PodWorkload) {
	settings := *rule.op.Surge
	if surge != nil {
		settings = surge.Spec.Settings
		if e.syncRule(ctx, surge, rule, now) {
			settings = surge.Spec.Settings
		}
	}

	var observation *odigosv1.TraceSurgeObservation
	if m != nil {
		if value, operation, requests, ok := m.observe(string(settings.Metric), e.window, settings.MinimumRequests); ok {
			observation = &odigosv1.TraceSurgeObservation{At: metav1.NewTime(now), Value: round2(value), Requests: requests, Operation: truncate(operation, maxOperation),
				WindowSeconds: int(m.windowOr(e.window).Seconds())}
		}
	}

	if surge == nil {
		if e.latched[key] {
			// the last surge ended at the maximum duration: the service must recover first.
			if observation != nil && observation.Value <= settings.RecoveryThreshold {
				e.recordRecovery(ctx, key, string(settings.Metric), observation, now)
				delete(e.latched, key)
			}
			return
		}
		if observation == nil || observation.Value <= settings.Threshold {
			delete(e.breaches, key)
			return
		}
		b := e.breaches[key]
		if b == nil {
			b = &breach{since: now}
			e.breaches[key] = b
		}
		b.samples = appendCapped(b.samples, *observation, maxTriggerTrail)
		if now.Sub(b.since) < time.Duration(settings.SustainedSeconds)*time.Second {
			return
		}
		targets := targetsFor(key.service, inScope, callers, services, byServiceName)
		if len(targets) == 0 {
			if !e.noTargets[key] {
				e.Logger.Info("trace surge has no workload in the rule's scope that leads to the service", "rule", rule.id, "service", key.service)
				e.noTargets[key] = true
			}
			return
		}
		delete(e.noTargets, key)
		e.start(ctx, now, key, rule, b, *observation, targets)
		delete(e.breaches, key)
		return
	}

	if surge.Active() && now.Sub(surge.Status.BoostedAt.Time) >= e.limits.maxDuration {
		if surge.Status.Phase == odigosv1.TraceSurgePhaseRecovering {
			// the metric already recovered: the surge ends without holding the service back.
			metric := string(settings.Metric)
			e.restore(ctx, surge, now, causeMaxDuration, fmt.Sprintf("%s was at or below %s when the maximum surge duration of %s was reached.",
				metricTitle(metric), formatValue(metric, settings.RecoveryThreshold), e.limits.maxDuration))
			return
		}
		e.latched[key] = true
		e.ended[key] = surge
		e.restore(ctx, surge, now, causeMaxDuration, fmt.Sprintf("%s %s. %s must recover before it can surge again.",
			maxDurationReason, e.limits.maxDuration, workloadLabel(key.service)))
		return
	}

	if observation == nil {
		last := surge.Status.LastObservation
		if last != nil && now.Sub(last.At.Time) > staleAfter {
			e.restore(ctx, surge, now, causeStaleMetrics, fmt.Sprintf("No metrics from %s for %s. Missing metrics never count as recovery, so the surge ended without it.", workloadLabel(key.service), staleAfter))
		}
		return
	}

	previous := surge.Status.LastObservation
	surge.Status.LastObservation = observation
	if targets := targetsFor(key.service, inScope, callers, services, byServiceName); len(targets) > 0 {
		e.addTargets(ctx, surge, targets, now)
	}
	metric := string(settings.Metric)
	switch surge.Status.Phase {
	case odigosv1.TraceSurgePhaseLimited:
		if observation.Value <= settings.RecoveryThreshold {
			e.restore(ctx, surge, now, causeRecovered, fmt.Sprintf("%s went back to %s before the trace surge limits let sampling rise.",
				metricTitle(metric), formatValue(metric, observation.Value)))
			return
		}
		if observation.Value <= settings.Threshold {
			break
		}
		reason := e.limitReason(surge.Spec.Targets, surge)
		if reason == "" {
			reason = e.roomReason(surge)
		}
		if reason == "" {
			at := metav1.NewTime(now)
			surge.Status.Phase = odigosv1.TraceSurgePhaseBoosting
			surge.Status.BoostedAt = &at
			addEvent(surge, now, fmt.Sprintf("Sampling raised to %s", formatPercent(settings.BoostPercent)),
				fmt.Sprintf("Within the trace surge limits again, with %s still above %s (%s now). New traces starting at %s are sampled at %s instead of %s.",
					metricTitle(metric), formatValue(metric, settings.Threshold), formatValue(metric, observation.Value),
					someTargetNames(surge.Spec.Targets, namedTargets), formatPercent(settings.BoostPercent), formatPercent(surge.Spec.NormalPercent)))
			e.requiredBytes[surge.Spec.SamplingName] += surgeSize(surge)
			e.telemetry.transition(ctx, surge, eventPromoted, "")
			e.Logger.Info("trace surge started", "surge", surge.Name, "rule", rule.id, "service", key.service, "value", observation.Value)
		} else if reason != surge.Status.LimitReason {
			surge.Status.LimitReason = reason
		}
	case odigosv1.TraceSurgePhaseBoosting:
		if observation.Value <= settings.RecoveryThreshold {
			surge.Status.Phase = odigosv1.TraceSurgePhaseRecovering
			surge.Status.RecoveryStartedAt = &observation.At
			e.telemetry.transition(ctx, surge, eventRecovering, "")
			addEvent(surge, now, fmt.Sprintf("%s back at %s", metricTitle(metric), formatValue(metric, observation.Value)),
				fmt.Sprintf("At or below %s: the %ds recovery window started.", formatValue(metric, settings.RecoveryThreshold), settings.RecoverySeconds))
		}
	case odigosv1.TraceSurgePhaseRecovering:
		if observation.Value > settings.RecoveryThreshold {
			surge.Status.Phase = odigosv1.TraceSurgePhaseBoosting
			surge.Status.RecoveryStartedAt = nil
			e.telemetry.transition(ctx, surge, eventRebounded, "")
			addEvent(surge, now, fmt.Sprintf("%s rose to %s", metricTitle(metric), formatValue(metric, observation.Value)),
				"Above the recovery threshold again: the recovery window restarts once it is back.")
		} else if previous == nil || observation.At.Sub(previous.At.Time) > 2*e.interval {
			// missing observations never count as recovery, e.g. while odigos insights was down.
			surge.Status.RecoveryStartedAt = &observation.At
			addEvent(surge, now, "Recovery window restarted",
				fmt.Sprintf("No metrics from %s for more than %s. Missing metrics never count as recovery: the %ds recovery window starts again.",
					workloadLabel(key.service), 2*e.interval, settings.RecoverySeconds))
		} else if recovered, boosted := now.Sub(surge.Status.RecoveryStartedAt.Time), now.Sub(surge.Status.BoostedAt.Time); recovered >= time.Duration(settings.RecoverySeconds)*time.Second &&
			boosted >= time.Duration(settings.MinimumBoostSeconds)*time.Second {
			e.restore(ctx, surge, now, causeRecovered, fmt.Sprintf("%s stayed at or below %s for %ds.", metricTitle(metric), formatValue(metric, settings.RecoveryThreshold), settings.RecoverySeconds))
			return
		}
	}
	e.updateStatus(ctx, surge)
}

// start records a surge of the rule for the service. It raises the sampling of the nearest
// maxBoostedWorkloads targets, unless a trace surge limit keeps it from doing so.
func (e *Evaluator) start(ctx context.Context, now time.Time, key surgeKey, rule surgeRule, b *breach, trigger odigosv1.TraceSurgeObservation, targets []odigosv1.TraceSurgeTarget) {
	settings := *rule.op.Surge
	state := e.samplings[rule.sampling]
	if state == nil {
		return
	}
	var leftOut []odigosv1.TraceSurgeTarget
	if limit := max(e.limits.maxBoosted, 1); len(targets) > limit {
		targets, leftOut = targets[:limit], targets[limit:]
	}
	limitReason := e.limitReason(targets, nil)
	surge := &odigosv1.TraceSurge{
		Name:      surgeName(key, now),
		StartedAt: metav1.NewTime(now),
		Spec: odigosv1.TraceSurgeSpec{
			SamplingName:  rule.sampling,
			RuleID:        rule.id,
			RuleName:      rule.op.Name,
			Settings:      settings,
			NormalPercent: rule.normalPercent(),
			Service:       key.service,
			Targets:       targets,
		},
	}
	state.surges = append(state.surges, surge)
	state.dirty = true
	e.active[key] = surge

	metric := string(settings.Metric)
	at := metav1.NewTime(now)
	surge.Status = odigosv1.TraceSurgeStatus{
		Phase:           odigosv1.TraceSurgePhaseBoosting,
		Trigger:         &trigger,
		TriggerSamples:  b.samples,
		LastObservation: &trigger,
		BoostedAt:       &at,
	}
	addEvent(surge, b.since, fmt.Sprintf("%s crossed %s", metricTitle(metric), formatValue(metric, settings.Threshold)),
		fmt.Sprintf("%s: %s over %d calls in the last %s.", observedLabel(key.service, b.samples[0].Operation), formatValue(metric, b.samples[0].Value), b.samples[0].Requests, windowText(time.Duration(b.samples[0].WindowSeconds)*time.Second)))
	if limitReason == "" {
		limitReason = e.roomReason(surge)
	}
	if limitReason != "" {
		surge.Status.Phase = odigosv1.TraceSurgePhaseLimited
		surge.Status.BoostedAt = nil
		surge.Status.LimitReason = limitReason
		addEvent(surge, now, "Sampling not raised", limitReason)
		e.telemetry.transition(ctx, surge, eventLimited, "")
		e.Logger.Info("trace surge limited", "surge", surge.Name, "rule", rule.id, "service", key.service, "reason", limitReason)
	} else {
		addEvent(surge, now, fmt.Sprintf("Sampling raised to %s", formatPercent(settings.BoostPercent)),
			fmt.Sprintf("%s stayed above %s for %ds (%s now). New traces starting at %s are sampled at %s instead of %s.",
				metricTitle(metric), formatValue(metric, settings.Threshold), settings.SustainedSeconds, formatValue(metric, trigger.Value),
				someTargetNames(targets, namedTargets), formatPercent(settings.BoostPercent), formatPercent(rule.normalPercent())))
		e.requiredBytes[rule.sampling] += surgeSize(surge)
		e.telemetry.transition(ctx, surge, eventStarted, "")
		e.Logger.Info("trace surge started", "surge", surge.Name, "rule", rule.id, "service", key.service, "value", trigger.Value, "targets", len(targets))
	}
	if len(leftOut) > 0 {
		e.noteLeftOut(ctx, surge, leftOut, now)
	}
	e.updateStatus(ctx, surge)
}

func (e *Evaluator) restore(ctx context.Context, surge *odigosv1.TraceSurge, now time.Time, cause restoreCause, reason string) {
	e.telemetry.transition(ctx, surge, eventRestored, cause)
	if surge.Status.BoostedAt == nil {
		// a limited surge never raised sampling: there is nothing to restore.
		at := metav1.NewTime(now)
		surge.Status.Phase = odigosv1.TraceSurgePhaseRestored
		surge.Status.RestoredAt = &at
		surge.Status.RestoreReason = reason
		addEvent(surge, now, "Surge ended", reason)
		e.updateStatus(ctx, surge)
		return
	}
	at := metav1.NewTime(now)
	surge.Status.Phase = odigosv1.TraceSurgePhaseRestored
	surge.Status.RestoredAt = &at
	surge.Status.RecoveryStartedAt = nil
	surge.Status.RestoreReason = reason
	addEvent(surge, now, fmt.Sprintf("Sampling restored to %s", formatPercent(surge.Spec.NormalPercent)), reason)
	e.updateStatus(ctx, surge)
	e.Logger.Info("trace surge ended", "surge", surge.Name, "reason", reason)
}

// syncRule applies edits of the rule to an active surge, and reports whether its settings changed.
func (e *Evaluator) syncRule(ctx context.Context, surge *odigosv1.TraceSurge, rule surgeRule, now time.Time) bool {
	if *rule.op.Surge == surge.Spec.Settings && rule.normalPercent() == surge.Spec.NormalPercent && rule.op.Name == surge.Spec.RuleName {
		return false
	}
	before := surge.Spec.Settings.BoostPercent
	surge.Spec.Settings = *rule.op.Surge
	surge.Spec.NormalPercent = rule.normalPercent()
	surge.Spec.RuleName = rule.op.Name
	e.updateStatus(ctx, surge)
	description := "The rule's surge settings changed; the surge continues with them."
	if before != surge.Spec.Settings.BoostPercent {
		description = fmt.Sprintf("The rule's boost changed from %s to %s.", formatPercent(before), formatPercent(surge.Spec.Settings.BoostPercent))
	}
	addEvent(surge, now, "Rule updated", description)
	return true
}

// addTargets adds the workloads that lead to the service since the surge started, nearest first, as
// far as maxBoostedWorkloads allows: an active surge adds the ones that fit the limit with the
// workloads every active surge raises, and a limited one keeps at most the limit.
func (e *Evaluator) addTargets(ctx context.Context, surge *odigosv1.TraceSurge, targets []odigosv1.TraceSurgeTarget, now time.Time) {
	has := make(map[k8sconsts.PodWorkload]bool, len(surge.Spec.Targets))
	for _, t := range surge.Spec.Targets {
		has[t.Workload] = true
	}
	active, limit := surge.Active(), max(e.limits.maxBoosted, 1)
	var boosted map[k8sconsts.PodWorkload]bool
	var added, leftOut []odigosv1.TraceSurgeTarget
	if active {
		boosted = e.boostedWorkloads()
	} else if len(surge.Spec.Targets) > limit {
		// the limit was lowered: a limited surge keeps the nearest targets, so that it can still raise them.
		leftOut = append(leftOut, surge.Spec.Targets[limit:]...)
		surge.Spec.Targets = surge.Spec.Targets[:limit]
		e.updateStatus(ctx, surge)
	}
	for _, t := range targets {
		switch {
		case has[t.Workload]:
		case active && (boosted[t.Workload] || len(boosted) < e.limits.maxBoosted):
			boosted[t.Workload] = true
			added = append(added, t)
		case !active && len(surge.Spec.Targets)+len(added) < limit:
			added = append(added, t)
		default:
			leftOut = append(leftOut, t)
		}
	}
	if len(added) > 0 {
		surge.Spec.Targets = append(surge.Spec.Targets, added...)
		addEvent(surge, now, "More services raised", fmt.Sprintf("%s also lead to %s.", someTargetNames(added, namedTargets), workloadLabel(surge.Spec.Service)))
		e.updateStatus(ctx, surge)
	}
	if len(leftOut) > 0 {
		e.noteLeftOut(ctx, surge, leftOut, now)
	}
}

const leftOutTitle = "Services left out"

// noteLeftOut records the workloads that lead to the service but that maxBoostedWorkloads keeps
// the surge from raising, unless the latest such event names the same ones.
func (e *Evaluator) noteLeftOut(ctx context.Context, surge *odigosv1.TraceSurge, leftOut []odigosv1.TraceSurgeTarget, now time.Time) {
	description := fmt.Sprintf("%s also lead to %s. Raising them would sample more than %s at a raised percentage, the limit (sampling.traceSurge.maxBoostedWorkloads).",
		someTargetNames(leftOut, 5), workloadLabel(surge.Spec.Service), countOf(e.limits.maxBoosted, "workload"))
	for i := len(surge.Status.Timeline) - 1; i >= 0; i-- {
		if surge.Status.Timeline[i].Title == leftOutTitle {
			if surge.Status.Timeline[i].Description == description {
				return
			}
			break
		}
	}
	addEvent(surge, now, leftOutTitle, description)
	e.updateStatus(ctx, surge)
}

// updateStatus marks the surge's Sampling for writing at the end of the evaluation.
func (e *Evaluator) updateStatus(_ context.Context, surge *odigosv1.TraceSurge) {
	if state := e.samplings[surge.Spec.SamplingName]; state != nil {
		state.dirty = true
	}
}

// archiveEnded moves the surges that are over from the Sampling status to odigos insights, for at
// most archiveBudget, and reports whether it removed any surge. A surge that ended at the maximum
// duration stays while it holds its service back.
func (e *Evaluator) archiveEnded(ctx context.Context, now time.Time) bool {
	ctx, cancel := context.WithTimeout(ctx, archiveBudget)
	defer cancel()
	archiving := !now.Before(e.archiveRetryAt)
	moved, dropped := 0, false
	for _, state := range e.samplings {
		kept := state.surges[:0]
		var ended []*odigosv1.TraceSurge
		for _, surge := range state.surges {
			over := surge.Status.Phase == odigosv1.TraceSurgePhaseRestored && surge.Status.RestoredAt != nil && !e.holdsBack(surge)
			if over && archiving && ctx.Err() == nil {
				err := e.archive.put(ctx, surge)
				if err == nil {
					moved++
					state.dirty = true
					continue
				}
				archiving = false
				// out of time after moving some: the others move in the next evaluation.
				if ctx.Err() == nil || moved == 0 {
					e.archiveRetryAt = now.Add(archiveRetryAfter)
					if err.Error() != e.lastArchiveErr {
						e.Logger.Error(err, "failed to move ended trace surges to odigos insights; they wait in the Sampling status", "surge", surge.Name)
						e.lastArchiveErr = err.Error()
					}
				}
			}
			kept = append(kept, surge)
			if surge.Status.Phase == odigosv1.TraceSurgePhaseRestored {
				ended = append(ended, surge)
			}
		}
		state.surges = kept
		// insights is down: keep the newest ended surges, and the ones that hold a service back.
		if len(ended) > maxEndedInStatus {
			slices.SortFunc(ended, func(a, b *odigosv1.TraceSurge) int { return b.StartedAt.Compare(a.StartedAt.Time) })
			drop := map[*odigosv1.TraceSurge]bool{}
			for _, surge := range ended[maxEndedInStatus:] {
				if !e.holdsBack(surge) {
					drop[surge] = true
				}
			}
			state.surges = slices.DeleteFunc(state.surges, func(surge *odigosv1.TraceSurge) bool { return drop[surge] })
			state.dirty = state.dirty || len(drop) > 0
			dropped = dropped || len(drop) > 0
		}
	}
	return moved > 0 || dropped
}

// flush writes the status of each Sampling whose surges changed, with the surges it has room for
// (statusSurges), and keeps the others in memory. The instrumentor leader is the only writer of the
// trace surges in it, so the status is patched without a version check, and a change of the rules
// in between does not fail it.
func (e *Evaluator) flush(ctx context.Context) {
	unwritten := map[string][]*odigosv1.TraceSurge{}
	for name, state := range e.samplings {
		if state.dirty {
			original := state.obj.DeepCopy()
			state.obj.Status.TraceSurges = e.statusSurges(state.surges, original.Status.TraceSurges)
			err := e.Client.Status().Patch(ctx, state.obj, client.MergeFrom(original))
			if err != nil && !apierrors.IsNotFound(err) {
				// e.g. the status is too large: the surges that decide what the agents sample, compact,
				// are written alone, so that a restoration is never held up by the rest.
				e.Logger.Error(err, "failed to write the trace surges of a Sampling; writing the active and restored ones only", "sampling", state.obj.Name)
				state.obj.Status.TraceSurges = compactSurges(state.surges, original.Status.TraceSurges)
				if err = e.Client.Status().Patch(ctx, state.obj, client.MergeFrom(original)); err != nil && !apierrors.IsNotFound(err) {
					e.Logger.Error(err, "failed to write the active and restored trace surges of a Sampling", "sampling", state.obj.Name)
				}
			}
			if err != nil {
				// what the status holds is unchanged: the surges not in it are evaluated again from memory.
				state.obj = original
			}
			state.dirty = false
		}
		written := make(map[string]bool, len(state.obj.Status.TraceSurges))
		for i := range state.obj.Status.TraceSurges {
			written[state.obj.Status.TraceSurges[i].Name] = true
		}
		for _, surge := range state.surges {
			if !written[surge.Name] {
				unwritten[name] = append(unwritten[name], surge)
			}
		}
	}
	e.unwritten = unwritten
}

// statusSurges returns the surges a Sampling status keeps, in their order. The active surges, and
// the ones the status still has active, are always kept: they decide what the agents sample. Then,
// while the status stays under maxStatusBytes, the ended surges (those holding their service back
// first, then the newest) and the newest maxLimitedInStatus Limited ones.
func (e *Evaluator) statusSurges(surges []*odigosv1.TraceSurge, written []odigosv1.TraceSurge) []odigosv1.TraceSurge {
	wasActive := activeNames(written)
	keep := make(map[*odigosv1.TraceSurge]bool, len(surges))
	size := 0
	var ended, limited []*odigosv1.TraceSurge
	for _, surge := range surges {
		switch {
		case surge.Active() || wasActive[surge.Name]:
			keep[surge] = true
			size += surgeSize(surge)
		case surge.Status.Phase == odigosv1.TraceSurgePhaseLimited:
			limited = append(limited, surge)
		default:
			ended = append(ended, surge)
		}
	}
	newestFirst := func(a, b *odigosv1.TraceSurge) int { return b.StartedAt.Compare(a.StartedAt.Time) }
	slices.SortFunc(ended, func(a, b *odigosv1.TraceSurge) int {
		if heldA, heldB := e.holdsBack(a), e.holdsBack(b); heldA != heldB {
			if heldA {
				return -1
			}
			return 1
		}
		return newestFirst(a, b)
	})
	slices.SortFunc(limited, newestFirst)
	optional := append(ended, limited[:min(len(limited), maxLimitedInStatus)]...)
	for _, surge := range optional {
		n := surgeSize(surge)
		if size+n > maxStatusBytes {
			break
		}
		keep[surge] = true
		size += n
	}
	kept := make([]odigosv1.TraceSurge, 0, len(keep))
	for _, surge := range surges {
		if keep[surge] {
			kept = append(kept, *surge)
		}
	}
	return kept
}

// compactSurges returns the surges a Sampling status must keep, the active ones and the ones it
// still has active, with the least of each: no trigger samples, the latest events, and the first
// targets of an ended one.
func compactSurges(surges []*odigosv1.TraceSurge, written []odigosv1.TraceSurge) []odigosv1.TraceSurge {
	wasActive := activeNames(written)
	var kept []odigosv1.TraceSurge
	for _, surge := range surges {
		if !surge.Active() && !wasActive[surge.Name] {
			continue
		}
		compact := *surge
		compact.Status.TriggerSamples = nil
		compact.Status.Timeline = compact.Status.Timeline[max(0, len(compact.Status.Timeline)-compactTimeline):]
		if !compact.Active() {
			compact.Spec.Targets = compact.Spec.Targets[:min(len(compact.Spec.Targets), compactTargets)]
		}
		kept = append(kept, compact)
	}
	return kept
}

// activeNames returns the names of the surges a status has active.
func activeNames(written []odigosv1.TraceSurge) map[string]bool {
	active := map[string]bool{}
	for i := range written {
		if written[i].Active() {
			active[written[i].Name] = true
		}
	}
	return active
}

func surgeSize(surge *odigosv1.TraceSurge) int {
	data, _ := json.Marshal(surge)
	return len(data)
}

// repair sets the times a surge's phase relies on that its status lacks, e.g. after an edit by
// hand, as if the phase started now.
func repair(surge *odigosv1.TraceSurge, now time.Time) bool {
	at := func() *metav1.Time {
		t := metav1.NewTime(now)
		return &t
	}
	repaired := false
	if surge.Active() && surge.Status.BoostedAt == nil {
		surge.Status.BoostedAt = at()
		repaired = true
	}
	if surge.Status.Phase == odigosv1.TraceSurgePhaseRecovering && surge.Status.RecoveryStartedAt == nil {
		surge.Status.RecoveryStartedAt = at()
		repaired = true
	}
	if surge.Status.Phase == odigosv1.TraceSurgePhaseRestored && surge.Status.RestoredAt == nil {
		surge.Status.RestoredAt = at()
		repaired = true
	}
	return repaired
}

// instrumentedWorkloads returns the languages of each instrumented workload's containers.
func (e *Evaluator) instrumentedWorkloads(ctx context.Context) (map[k8sconsts.PodWorkload][]common.ProgrammingLanguage, error) {
	configs := &odigosv1.InstrumentationConfigList{}
	// read only: the cache's objects are not copied.
	if err := e.Client.List(ctx, configs, client.UnsafeDisableDeepCopy); err != nil {
		return nil, err
	}
	workloads := map[k8sconsts.PodWorkload][]common.ProgrammingLanguage{}
	for _, ic := range configs.Items {
		pw, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(ic.Name, ic.Namespace)
		if err != nil {
			continue
		}
		langs := make([]common.ProgrammingLanguage, 0, len(ic.Status.RuntimeDetailsByContainer))
		for _, rd := range ic.Status.RuntimeDetailsByContainer {
			langs = append(langs, rd.Language)
		}
		workloads[pw] = langs
	}
	return workloads, nil
}

// reachable returns the workloads with metrics that the rule evaluates: those in its scope, and
// every workload they call, directly or not.
func reachable(inScope map[k8sconsts.PodWorkload]bool, services map[k8sconsts.PodWorkload]*serviceMetrics, callees map[string][]string, byServiceName map[string]k8sconsts.PodWorkload) map[k8sconsts.PodWorkload]bool {
	seen := map[k8sconsts.PodWorkload]bool{}
	var queue []k8sconsts.PodWorkload
	for pw := range inScope {
		seen[pw] = true
		queue = append(queue, pw)
	}
	for len(queue) > 0 {
		pw := queue[0]
		queue = queue[1:]
		name := serviceKey(pw.Namespace, pw.Name)
		if m := services[pw]; m != nil {
			name = m.key()
		}
		for _, callee := range callees[name] {
			if next, ok := byServiceName[callee]; ok && !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	for pw := range seen {
		if services[pw] == nil {
			delete(seen, pw)
		}
	}
	return seen
}

// targetsFor returns the workloads in the rule's scope whose traces lead to the service, nearest
// first: the service itself, and every service that calls it, directly or not.
func targetsFor(service k8sconsts.PodWorkload, inScope map[k8sconsts.PodWorkload]bool, callers map[string][]string, services map[k8sconsts.PodWorkload]*serviceMetrics, byServiceName map[string]k8sconsts.PodWorkload) []odigosv1.TraceSurgeTarget {
	hops := map[k8sconsts.PodWorkload]int{service: 0}
	queue := []k8sconsts.PodWorkload{service}
	for len(queue) > 0 {
		pw := queue[0]
		queue = queue[1:]
		name := serviceKey(pw.Namespace, pw.Name)
		if m := services[pw]; m != nil {
			name = m.key()
		}
		for _, caller := range callers[name] {
			if next, ok := byServiceName[caller]; ok {
				if _, seen := hops[next]; !seen {
					hops[next] = hops[pw] + 1
					queue = append(queue, next)
				}
			}
		}
	}
	var targets []odigosv1.TraceSurgeTarget
	for pw := range hops {
		if inScope[pw] {
			targets = append(targets, odigosv1.TraceSurgeTarget{Workload: pw})
		}
	}
	slices.SortFunc(targets, func(a, b odigosv1.TraceSurgeTarget) int {
		if c := cmp.Compare(hops[a.Workload], hops[b.Workload]); c != 0 {
			return c
		}
		return strings.Compare(workloadLabel(a.Workload), workloadLabel(b.Workload))
	})
	return targets
}

func addEvent(surge *odigosv1.TraceSurge, at time.Time, title, description string) {
	event := odigosv1.TraceSurgeEvent{At: metav1.NewTime(at), Title: title, Description: truncate(description, maxDescription)}
	surge.Status.Timeline = appendCapped(surge.Status.Timeline, event, maxTimeline)
}

// truncate cuts s to at most n bytes, at a rune boundary, marking the cut.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	const mark = "…"
	cut := n - len(mark)
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + mark
}

func appendCapped[T any](s []T, v T, limit int) []T {
	s = append(s, v)
	if len(s) > limit {
		s = s[len(s)-limit:]
	}
	return s
}

func keyOf(surge *odigosv1.TraceSurge) surgeKey {
	return surgeKey{surge.Spec.SamplingName, surge.Spec.RuleID, surge.Spec.Service}
}

// surgeName identifies a surge, also in odigos insights: readable, with a 64-bit hash of the
// Sampling, rule, service and start that tells apart services of the same name in other namespaces.
func surgeName(key surgeKey, now time.Time) string {
	service := key.service.Name
	if len(service) > 32 {
		service = strings.TrimRight(service[:32], "-.")
	}
	identity := sha256.Sum256([]byte(fmt.Sprintf("%s/%s/%s/%s/%s/%d", key.sampling, key.ruleID, key.service.Namespace, key.service.Kind, key.service.Name, now.UnixNano())))
	return fmt.Sprintf("%s-%s-%s-%s", key.ruleID[:min(8, len(key.ruleID))], service, strconv.FormatInt(now.Unix(), 36), hex.EncodeToString(identity[:8]))
}

func workloadLabel(pw k8sconsts.PodWorkload) string {
	return pw.Namespace + "/" + pw.Name
}

// observedLabel names what an observation measured: a service, or one of its operations.
func observedLabel(pw k8sconsts.PodWorkload, operation string) string {
	if operation == "" {
		return workloadLabel(pw)
	}
	return workloadLabel(pw) + " " + operation
}

func targetNames(targets []odigosv1.TraceSurgeTarget) string {
	names := make([]string, len(targets))
	for i, t := range targets {
		names[i] = workloadLabel(t.Workload)
	}
	return strings.Join(names, ", ")
}

// someTargetNames names the first n targets, and counts the others.
func someTargetNames(targets []odigosv1.TraceSurgeTarget, n int) string {
	if len(targets) <= n {
		return targetNames(targets)
	}
	return fmt.Sprintf("%s and %d more", targetNames(targets[:n]), len(targets)-n)
}

func windowText(d time.Duration) string {
	if d == time.Minute {
		return "minute"
	}
	return d.String()
}

func metricTitle(metric string) string {
	switch metric {
	case "error_rate":
		return "Error rate"
	case "latency_p95":
		return "p95 latency"
	case "request_rate":
		return "Request rate"
	}
	return metric
}

func formatValue(metric string, v float64) string {
	switch metric {
	case "error_rate":
		return formatPercent(v)
	case "latency_p95":
		return strconv.FormatFloat(round2(v), 'f', -1, 64) + " ms"
	case "request_rate":
		return strconv.FormatFloat(round2(v), 'f', -1, 64) + " req/s"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func formatPercent(v float64) string {
	return strconv.FormatFloat(round2(v), 'f', -1, 64) + "%"
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
