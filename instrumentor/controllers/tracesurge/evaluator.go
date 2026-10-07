// Package tracesurge raises the head sampling of a noisy operation rule while a service its
// traces reach has a spike in a RED metric, and restores it once the metric recovers.
//
// Every evaluation interval it reads, from odigos insights, which stores them in its ClickHouse,
// the span metrics agents record before sampling and the service graph. Without insights no
// surge starts. For each rule with a surge, every service in the rule's
// scope and every service those call is evaluated on its own. A service whose metric stays above
// the rule's threshold starts a surge: an entry in the status of the Sampling object that holds
// the rule, which raises the rule's percentage for the workloads in the rule's scope that lead to
// the service (agentenabled applies it), and records what happened, down to the processes that
// confirm applying it. Once a surge is over, it moves to odigos insights.
package tracesurge

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

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
	// an active surge whose service has had no metrics for this long is ended.
	// missing metrics never count as recovery, but they do not hold the surge forever either.
	staleAfter = 10 * time.Minute
	// after a surge ends, its targets are watched this long for returning to the normal percentage.
	confirmRestoreFor = 10 * time.Minute
	// ended surges wait in the Sampling status at most this many per Sampling, when odigos insights
	// can't take them: beyond it the oldest are dropped, so the status stays small.
	maxEndedInStatus = 20
	// after a failed put to odigos insights, no surge is put for this long.
	archiveRetryAfter = 30 * time.Second
	maxTimeline       = 30
	maxTriggerTrail   = 30
)

// Evaluator runs on the leader instrumentor.
type Evaluator struct {
	Client    client.Client
	APIReader client.Reader
	Logger    logr.Logger
	// the evaluator's own metrics are created with it; nil records none.
	Meter metric.Meter

	namespace string
	metrics   metricsReader
	// services whose metric is above the threshold of a rule, without an active surge yet.
	breaches map[surgeKey]*breach
	// the surges that found no workload to raise, so that it is logged once.
	noTargets      map[surgeKey]bool
	lastQueryErr   string
	lastArchiveErr string
	// after a failed put to insights, ended surges wait in the status until this time, so that
	// insights being down doesn't slow each evaluation.
	archiveRetryAt time.Time
	// the open surges (limited or active) of the current evaluation.
	active map[surgeKey]*odigosv1.TraceSurge
	// the limits of the current evaluation.
	limits limits
	// the window the current evaluation measures, and the time until the next one.
	window, interval time.Duration
	// services whose surge ended at the maximum duration: they must recover before surging again.
	latched map[surgeKey]bool
	// the latest ended surge of each rule and service, of the current evaluation.
	ended map[surgeKey]*odigosv1.TraceSurge
	// the Sampling objects of the current evaluation, with their surges.
	samplings map[string]*samplingState
	// where ended surges go: odigos insights.
	archive   archiver
	telemetry *evaluatorMetrics
	// what the current evaluation read from odigos insights.
	servicesRead int
	queryFailed  bool
}

// samplingState is a Sampling object and its surges as the current evaluation changes them. Its
// status is written once, at the end of the evaluation, when they changed.
type samplingState struct {
	obj    *odigosv1.Sampling
	surges []*odigosv1.TraceSurge
	dirty  bool
}

var _ manager.LeaderElectionRunnable = &Evaluator{}

// metricsReader returns the RED metrics of every workload over the window, with those of its
// operations that had at least minOperationCalls calls, and for each service (serviceKey), the
// services that call it.
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
		if err := e.evaluate(ctx, time.Now()); err != nil {
			e.Logger.Error(err, "trace surge evaluation failed")
		}
		// the interval is configurable, so it is read again after every evaluation.
		timer.Reset(e.interval)
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
	e.servicesRead, e.queryFailed = 0, false
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
		state := &samplingState{obj: &samplings.Items[i]}
		for j := range samplings.Items[i].Status.TraceSurges {
			surge := samplings.Items[i].Status.TraceSurges[j].DeepCopy()
			state.surges = append(state.surges, surge)
		}
		e.samplings[samplings.Items[i].Name] = state
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
				active[surgeKey{surge.Spec.SamplingName, surge.Spec.RuleID, surge.Spec.Service}] = surge
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
	case !common.InsightsPipelineActive(conf.Insights):
		// the RED metrics are stored by insights: without it nothing is evaluated.
		result = resultInsightsOff
		clear(e.breaches)
		for _, surge := range active {
			e.restore(ctx, surge, now, causeInsightsOff, "Odigos Insights was turned off. Trace surges evaluate the RED metrics it stores.")
		}
	case len(rules) > 0 || len(active) > 0:
		if err := e.evaluateRules(ctx, now, rules, active); err != nil {
			return "", err
		}
		if e.queryFailed {
			result = resultMetricsError
		}
	default:
		clear(e.breaches)
	}

	for _, surge := range ended {
		// an ended surge is watched until its targets are confirmed back, then its record is final.
		if surge.Status.BoostedAt != nil && surge.Status.RestoredAt != nil && now.Sub(surge.Status.RestoredAt.Time) < confirmRestoreFor && !confirmed(surge) {
			if e.confirm(ctx, surge, now) {
				e.updateStatus(ctx, surge)
			}
		}
	}
	e.archiveEnded(ctx, now)
	e.flush(ctx)
	e.telemetry.snapshot(rules, e.samplings, e.servicesRead, e.limits)
	return result, nil
}

func (e *Evaluator) evaluateRules(ctx context.Context, now time.Time, rules map[[2]string]surgeRule, active map[surgeKey]*odigosv1.TraceSurge) error {
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
	e.queryFailed = err != nil
	if err != nil {
		// without metrics no service is evaluated; active surges hold until they are stale.
		if err.Error() != e.lastQueryErr {
			e.Logger.Error(err, "failed to read span metrics from odigos insights")
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
			observation = &odigosv1.TraceSurgeObservation{At: metav1.NewTime(now), Value: round2(value), Requests: requests, Operation: operation,
				WindowSeconds: int(e.window.Seconds())}
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
		e.start(ctx, now, key, rule, b, *observation, targets, e.limitReason(targets, nil))
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
		e.restore(ctx, surge, now, causeMaxDuration, fmt.Sprintf("%s %s. %s must recover before it can surge again.",
			maxDurationReason, e.limits.maxDuration, workloadLabel(key.service)))
		return
	}

	if observation == nil {
		last := surge.Status.LastObservation
		if last != nil && now.Sub(last.At.Time) > staleAfter {
			e.restore(ctx, surge, now, causeStaleMetrics, fmt.Sprintf("No metrics from %s for %s. Missing metrics never count as recovery, so the surge ended without it.", workloadLabel(key.service), staleAfter))
			return
		}
		if e.confirm(ctx, surge, now) {
			e.updateStatus(ctx, surge)
		}
		return
	}

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
			at := metav1.NewTime(now)
			surge.Status.Phase = odigosv1.TraceSurgePhaseBoosting
			surge.Status.BoostedAt = &at
			addEvent(surge, now, fmt.Sprintf("Sampling raised to %s", formatPercent(settings.BoostPercent)),
				fmt.Sprintf("Within the trace surge limits again, with %s still above %s (%s now). New traces starting at %s are sampled at %s instead of %s.",
					metricTitle(metric), formatValue(metric, settings.Threshold), formatValue(metric, observation.Value),
					targetNames(surge.Spec.Targets), formatPercent(settings.BoostPercent), formatPercent(surge.Spec.NormalPercent)))
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
		} else if recovered, boosted := now.Sub(surge.Status.RecoveryStartedAt.Time), now.Sub(surge.Status.BoostedAt.Time); recovered >= time.Duration(settings.RecoverySeconds)*time.Second &&
			boosted >= time.Duration(settings.MinimumBoostSeconds)*time.Second {
			e.restore(ctx, surge, now, causeRecovered, fmt.Sprintf("%s stayed at or below %s for %ds.", metricTitle(metric), formatValue(metric, settings.RecoveryThreshold), settings.RecoverySeconds))
			return
		}
	}
	if surge.Active() {
		e.confirm(ctx, surge, now)
	}
	e.updateStatus(ctx, surge)
}

// start records a surge of the rule for the service. It raises sampling, unless limitReason says
// which limit keeps it from doing so.
func (e *Evaluator) start(ctx context.Context, now time.Time, key surgeKey, rule surgeRule, b *breach, trigger odigosv1.TraceSurgeObservation, targets []odigosv1.TraceSurgeTarget, limitReason string) {
	settings := *rule.op.Surge
	state := e.samplings[rule.sampling]
	if state == nil {
		return
	}
	surge := &odigosv1.TraceSurge{
		Name:      surgeName(rule.id, key.service.Name, now),
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
		fmt.Sprintf("%s: %s over %d calls in the last %s.", observedLabel(key.service, b.samples[0].Operation), formatValue(metric, b.samples[0].Value), b.samples[0].Requests, windowText(e.window)))
	if limitReason != "" {
		surge.Status.Phase = odigosv1.TraceSurgePhaseLimited
		surge.Status.BoostedAt = nil
		surge.Status.LimitReason = limitReason
		addEvent(surge, now, "Sampling not raised", limitReason)
		e.updateStatus(ctx, surge)
		e.telemetry.transition(ctx, surge, eventLimited, "")
		e.Logger.Info("trace surge limited", "surge", surge.Name, "rule", rule.id, "service", key.service, "reason", limitReason)
		return
	}
	addEvent(surge, now, fmt.Sprintf("Sampling raised to %s", formatPercent(settings.BoostPercent)),
		fmt.Sprintf("%s stayed above %s for %ds (%s now). New traces starting at %s are sampled at %s instead of %s.",
			metricTitle(metric), formatValue(metric, settings.Threshold), settings.SustainedSeconds, formatValue(metric, trigger.Value),
			targetNames(targets), formatPercent(settings.BoostPercent), formatPercent(rule.normalPercent())))
	e.confirm(ctx, surge, now)
	e.updateStatus(ctx, surge)
	e.telemetry.transition(ctx, surge, eventStarted, "")
	e.Logger.Info("trace surge started", "surge", surge.Name, "rule", rule.id, "service", key.service, "value", trigger.Value, "targets", len(targets))
}

func (e *Evaluator) restore(ctx context.Context, surge *odigosv1.TraceSurge, now time.Time, cause restoreCause, reason string) {
	e.telemetry.transition(ctx, surge, eventRestored, cause)
	if surge.Status.BoostedAt == nil {
		// a limited surge never raised sampling: there is nothing to restore or confirm.
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
	for i := range surge.Status.Targets {
		surge.Status.Targets[i].ConfirmedAt = nil
	}
	addEvent(surge, now, fmt.Sprintf("Sampling restored to %s", formatPercent(surge.Spec.NormalPercent)), reason)
	e.confirm(ctx, surge, now)
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
		for i := range surge.Status.Targets {
			surge.Status.Targets[i].ConfirmedAt = nil
		}
	}
	addEvent(surge, now, "Rule updated", description)
	return true
}

func (e *Evaluator) addTargets(ctx context.Context, surge *odigosv1.TraceSurge, targets []odigosv1.TraceSurgeTarget, now time.Time) {
	var added []odigosv1.TraceSurgeTarget
	for _, t := range targets {
		if !slices.ContainsFunc(surge.Spec.Targets, func(existing odigosv1.TraceSurgeTarget) bool { return existing.Workload == t.Workload }) {
			added = append(added, t)
		}
	}
	if len(added) == 0 {
		return
	}
	surge.Spec.Targets = append(surge.Spec.Targets, added...)
	e.updateStatus(ctx, surge)
	addEvent(surge, now, "More services raised", fmt.Sprintf("%s also lead to %s.", targetNames(added), workloadLabel(surge.Spec.Service)))
}

// updateStatus marks the surge's Sampling for writing at the end of the evaluation.
func (e *Evaluator) updateStatus(_ context.Context, surge *odigosv1.TraceSurge) {
	if state := e.samplings[surge.Spec.SamplingName]; state != nil {
		state.dirty = true
	}
}

// archiveEnded moves the surges that are over from the Sampling status to odigos insights: those
// whose targets are confirmed back, or that were watched long enough for it. A surge that ended
// at the maximum duration stays until its service recovers, since it holds the service back.
func (e *Evaluator) archiveEnded(ctx context.Context, now time.Time) {
	archiving := !now.Before(e.archiveRetryAt)
	for _, state := range e.samplings {
		kept := state.surges[:0]
		var ended []*odigosv1.TraceSurge
		for _, surge := range state.surges {
			over := surge.Status.Phase == odigosv1.TraceSurgePhaseRestored && surge.Status.RestoredAt != nil &&
				(surge.Status.RecoveredAt != nil || !strings.HasPrefix(surge.Status.RestoreReason, maxDurationReason)) &&
				(surge.Status.BoostedAt == nil || confirmed(surge) || now.Sub(surge.Status.RestoredAt.Time) >= confirmRestoreFor)
			if over && archiving {
				err := e.archive.put(ctx, surge)
				if err == nil {
					state.dirty = true
					continue
				}
				archiving = false
				e.archiveRetryAt = now.Add(archiveRetryAfter)
				if err.Error() != e.lastArchiveErr {
					e.Logger.Error(err, "failed to move ended trace surges to odigos insights; they wait in the Sampling status", "surge", surge.Name)
					e.lastArchiveErr = err.Error()
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
				if surge.Status.RecoveredAt != nil || !strings.HasPrefix(surge.Status.RestoreReason, maxDurationReason) {
					drop[surge] = true
				}
			}
			state.surges = slices.DeleteFunc(state.surges, func(surge *odigosv1.TraceSurge) bool { return drop[surge] })
			state.dirty = state.dirty || len(drop) > 0
		}
	}
}

// flush writes the status of each Sampling whose surges changed. The instrumentor leader is the
// only writer of the trace surges in it, so the status is patched without a version check, and a
// change of the rules in between does not fail it.
func (e *Evaluator) flush(ctx context.Context) {
	for _, state := range e.samplings {
		if !state.dirty {
			continue
		}
		original := state.obj.DeepCopy()
		surges := make([]odigosv1.TraceSurge, len(state.surges))
		for i, surge := range state.surges {
			surges[i] = *surge
		}
		state.obj.Status.TraceSurges = surges
		if err := e.Client.Status().Patch(ctx, state.obj, client.MergeFrom(original)); err != nil && !apierrors.IsNotFound(err) {
			e.Logger.Error(err, "failed to write the trace surges of a Sampling", "sampling", state.obj.Name)
		}
		state.dirty = false
	}
}

// instrumentedWorkloads returns the languages of each instrumented workload's containers.
func (e *Evaluator) instrumentedWorkloads(ctx context.Context) (map[k8sconsts.PodWorkload][]common.ProgrammingLanguage, error) {
	configs := &odigosv1.InstrumentationConfigList{}
	if err := e.Client.List(ctx, configs); err != nil {
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

// targetsFor returns the workloads in the rule's scope whose traces lead to the service: the
// service itself, and every service that calls it, directly or not.
func targetsFor(service k8sconsts.PodWorkload, inScope map[k8sconsts.PodWorkload]bool, callers map[string][]string, services map[k8sconsts.PodWorkload]*serviceMetrics, byServiceName map[string]k8sconsts.PodWorkload) []odigosv1.TraceSurgeTarget {
	seen := map[k8sconsts.PodWorkload]bool{service: true}
	queue := []k8sconsts.PodWorkload{service}
	for len(queue) > 0 {
		pw := queue[0]
		queue = queue[1:]
		name := serviceKey(pw.Namespace, pw.Name)
		if m := services[pw]; m != nil {
			name = m.key()
		}
		for _, caller := range callers[name] {
			if next, ok := byServiceName[caller]; ok && !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	var targets []odigosv1.TraceSurgeTarget
	for pw := range seen {
		if inScope[pw] {
			targets = append(targets, odigosv1.TraceSurgeTarget{Workload: pw})
		}
	}
	slices.SortFunc(targets, func(a, b odigosv1.TraceSurgeTarget) int {
		return strings.Compare(a.Workload.Namespace+"/"+a.Workload.Name, b.Workload.Namespace+"/"+b.Workload.Name)
	})
	return targets
}

func addEvent(surge *odigosv1.TraceSurge, at time.Time, title, description string) {
	surge.Status.Timeline = appendCapped(surge.Status.Timeline, odigosv1.TraceSurgeEvent{At: metav1.NewTime(at), Title: title, Description: description}, maxTimeline)
}

func appendCapped[T any](s []T, v T, limit int) []T {
	s = append(s, v)
	if len(s) > limit {
		s = s[len(s)-limit:]
	}
	return s
}

func surgeName(ruleID, service string, now time.Time) string {
	if len(service) > 40 {
		service = strings.TrimRight(service[:40], "-.")
	}
	return fmt.Sprintf("%s-%s-%s", ruleID[:min(8, len(ruleID))], service, strconv.FormatInt(now.Unix(), 36))
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
