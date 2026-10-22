// Package telemetry serves the telemetry activity: the changes Odigos makes to the telemetry it
// collects. Trace surges are its source: the instrumentor keeps the open ones, and those that ended
// recently, in the status of the Sampling objects, and moves the others to odigos insights.
package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services/insights"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/vektah/gqlparser/v2/gqlerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// History keeps the trace surges that ended and left the Sampling status: odigos insights.
type History interface {
	ListTraceSurges(ctx context.Context, from, to time.Time, ruleID string, limit int) (*insights.TraceSurgeList, error)
	GetTraceSurge(ctx context.Context, name string) (*insights.TraceSurge, error)
}

const (
	sourceTraceSurge = "trace_surge"

	activityPageSize = 20

	// the most ended surges read from the history for one listing: insights lists the latest
	// ended first, at most this many, and has no paging.
	historyLimit   = 1000
	historyDefault = 7 * 24 * time.Hour
	notFound       = "This change no longer exists. Ended trace surges are kept for 30 days."

	// a read of the history serves the next pages of the same listing for this long, so paging
	// does not read it again.
	historyCacheFor  = 30 * time.Second
	historyCacheSize = 8

	// the requests of the targets when the rule that raised them no longer exists.
	matchingRequests = "Matching requests"
)

// GetActivity lists the trace surges of the filters. The active view reads the Sampling status
// only; the history view also the ended surges history keeps, when it is set.
func GetActivity(ctx context.Context, history History, filters model.TelemetryActivityFiltersInput, cursor *string) (*model.TelemetryActivityPage, error) {
	now := time.Now()
	page := &model.TelemetryActivityPage{Items: []*model.TelemetryActivitySummary{}, AsOf: now.UTC().Format(time.RFC3339)}
	if filters.Source != nil && *filters.Source != sourceTraceSurge {
		return page, nil
	}
	surges, err := statusSurges(ctx)
	if err != nil {
		return nil, err
	}
	from, fromErr := time.Parse(time.RFC3339, filters.From)
	to, toErr := time.Parse(time.RFC3339, filters.To)
	if filters.View != "active" && history != nil {
		start, end := now.Add(-historyDefault), now
		if fromErr == nil {
			start = from
		}
		if toErr == nil {
			end = to
		}
		ruleID := ""
		if filters.SourceID != nil {
			ruleID = *filters.SourceID
		}
		if start.Before(end) {
			ended, err := cachedHistorySurges(ctx, history, start, end, ruleID, now)
			if err != nil {
				return nil, err
			}
			if ended.truncated {
				limit := historyLimit
				page.HistoryLimit = &limit
			}
			surges = mergeSurges(surges, ended.surges)
		}
	}

	var matched []*model.TelemetryActivitySummary
	for i := range surges {
		surge := &surges[i]
		if filters.SourceID != nil && *filters.SourceID != surge.Spec.RuleID {
			continue
		}
		summary := summarize(surge)
		if filters.View == "active" {
			if !surge.Active() {
				continue
			}
		} else if updated := updatedAt(surge); (fromErr == nil && updated.Before(from)) || (toErr == nil && updated.After(to)) {
			continue
		}
		if filters.Query != nil && *filters.Query != "" && !matchesQuery(summary, *filters.Query) {
			continue
		}
		matched = append(matched, summary)
	}
	slices.SortFunc(matched, func(a, b *model.TelemetryActivitySummary) int {
		return strings.Compare(b.UpdatedAt, a.UpdatedAt)
	})

	page.Items, page.NextCursor, err = pageOf(matched, cursor)
	if err != nil {
		return nil, err
	}
	return page, nil
}

// pageOf returns the page of matched that starts at cursor, an offset, and the cursor of the next.
func pageOf(matched []*model.TelemetryActivitySummary, cursor *string) ([]*model.TelemetryActivitySummary, *string, error) {
	offset := 0
	if cursor != nil {
		n, err := strconv.Atoi(*cursor)
		if err != nil || n < 0 {
			return nil, nil, &gqlerror.Error{Message: fmt.Sprintf("invalid cursor %q", *cursor), Extensions: map[string]any{"code": "BAD_USER_INPUT"}}
		}
		offset = n
	}
	if offset >= len(matched) {
		return []*model.TelemetryActivitySummary{}, nil, nil
	}
	end := len(matched)
	if end-offset > activityPageSize {
		end = offset + activityPageSize
	}
	if end == len(matched) {
		return matched[offset:end], nil, nil
	}
	next := strconv.Itoa(end)
	return matched[offset:end], &next, nil
}

func GetChange(ctx context.Context, history History, id string) (*model.TelemetryActivityDetail, error) {
	var list v1alpha1.SamplingList
	if err := kube.CacheClient.List(ctx, &list, client.InNamespace(env.GetCurrentNamespace())); err != nil {
		return nil, fmt.Errorf("failed to list sampling CRs: %w", err)
	}
	surge, recordedIn, err := getSurge(ctx, history, list.Items, id)
	if err != nil {
		return nil, err
	}
	return changeDetail(surge, recordedIn, ruleRequests(list.Items, surge.Spec.RuleID)), nil
}

// changeDetail describes a surge; recordedIn says where its record is kept, and requests which
// requests of the targets the rule samples.
func changeDetail(surge *v1alpha1.TraceSurge, recordedIn, requests string) *model.TelemetryActivityDetail {
	s := summarize(surge)
	detail := &model.TelemetryActivityDetail{
		ID: s.ID, UpdatedAt: s.UpdatedAt, Title: s.Title, Before: s.Before, After: s.After,
		TargetSummary: s.TargetSummary, ScopeSummary: s.ScopeSummary, Source: s.Source, Reason: s.Reason,
		Trigger: s.Trigger, Status: s.Status, LimitReason: s.LimitReason,
	}

	for _, target := range surge.Spec.Targets {
		scope := scopeOf(target.Workload)
		scope.Operation = &requests
		change := &model.TelemetryTargetChange{
			ID:     workloadID(target.Workload),
			Scope:  scope,
			Before: s.Before,
			After:  s.After,
		}
		if target.BroadScopeReason != "" {
			change.BroadScopeReason = &target.BroadScopeReason
		}
		detail.Targets = append(detail.Targets, change)
	}
	if detail.Targets == nil {
		detail.Targets = []*model.TelemetryTargetChange{}
	}

	for i, event := range surge.Status.Timeline {
		detail.Timeline = append(detail.Timeline, &model.TelemetryTimelineEvent{
			ID: strconv.Itoa(i), At: timestamp(&event.At), Title: event.Title, Description: optional(event.Description),
		})
	}
	settings := surge.Spec.Settings
	switch {
	case surge.Active():
		recovery := fmt.Sprintf("Sampling returns to %s once %s stays at or below %s for %ds, and at least %ds after it was raised.",
			percent(surge.Spec.NormalPercent), metricTitle(string(settings.Metric)), formatValue(string(settings.Metric), settings.RecoveryThreshold),
			settings.RecoverySeconds, settings.MinimumBoostSeconds)
		detail.Timeline = append(detail.Timeline, &model.TelemetryTimelineEvent{
			ID: "next", Title: "Return to normal", Description: &recovery,
		})
	case surge.Status.Phase == v1alpha1.TraceSurgePhaseLimited:
		next := fmt.Sprintf("Sampling rises to %s if the trace surge limits allow it while %s is still above %s.",
			percent(settings.BoostPercent), metricTitle(string(settings.Metric)), formatValue(string(settings.Metric), settings.Threshold))
		detail.Timeline = append(detail.Timeline, &model.TelemetryTimelineEvent{
			ID: "next", Title: "Raise sampling within the limits", Description: &next,
		})
	}
	if detail.Timeline == nil {
		detail.Timeline = []*model.TelemetryTimelineEvent{}
	}

	detail.Evidence = []*model.TelemetryEvidence{
		{Label: "Service", Value: workloadID(surge.Spec.Service)},
		{Label: "Rule", Value: ruleName(surge)},
		{Label: "Phase", Value: string(surge.Status.Phase)},
		{Label: "Source data", Value: "Span metrics of server spans, recorded by the agent before sampling"},
		{Label: "Trace surge", Value: surge.Name},
		{Label: "Recorded in", Value: recordedIn},
	}
	if last := surge.Status.LastObservation; last != nil {
		detail.Evidence = append(detail.Evidence,
			&model.TelemetryEvidence{Label: "Latest value", Value: formatValue(string(settings.Metric), last.Value)},
			&model.TelemetryEvidence{Label: "Calls in the last " + windowText(last.WindowSeconds), Value: strconv.FormatInt(last.Requests, 10)})
		if last.Operation != "" {
			detail.Evidence = append(detail.Evidence, &model.TelemetryEvidence{Label: "Operation", Value: last.Operation})
		}
	}
	if surge.Status.LimitReason != "" {
		detail.Evidence = append(detail.Evidence, &model.TelemetryEvidence{Label: "Not raised because", Value: surge.Status.LimitReason})
	}
	if surge.Status.RestoreReason != "" {
		detail.Evidence = append(detail.Evidence, &model.TelemetryEvidence{Label: "Ended because", Value: surge.Status.RestoreReason})
	}
	return detail
}

// statusSurges returns the trace surges in the status of the Sampling objects: the open ones, and
// those that ended recently.
func statusSurges(ctx context.Context) ([]v1alpha1.TraceSurge, error) {
	var list v1alpha1.SamplingList
	if err := kube.CacheClient.List(ctx, &list, client.InNamespace(env.GetCurrentNamespace())); err != nil {
		return nil, fmt.Errorf("failed to list sampling CRs: %w", err)
	}
	var surges []v1alpha1.TraceSurge
	for i := range list.Items {
		surges = append(surges, list.Items[i].Status.TraceSurges...)
	}
	return surges, nil
}

// historySurges returns the trace surges of a list the history returned. A record that isn't a
// trace surge is skipped.
func historySurges(list *insights.TraceSurgeList) []v1alpha1.TraceSurge {
	surges := make([]v1alpha1.TraceSurge, 0, len(list.Items))
	for _, item := range list.Items {
		var surge v1alpha1.TraceSurge
		if json.Unmarshal(item.Record, &surge) != nil || surge.Name != item.Name {
			continue
		}
		surges = append(surges, surge)
	}
	return surges
}

type historyRead struct {
	surges    []v1alpha1.TraceSurge
	truncated bool
	readAt    time.Time
}

type historyKey struct {
	history  History
	from, to int64
	ruleID   string
}

var historyCache = struct {
	sync.Mutex
	reads map[historyKey]historyRead
}{reads: map[historyKey]historyRead{}}

// cachedHistorySurges returns the ended surges history keeps that overlap [from, to], from a read
// of the same range made less than historyCacheFor before now when there is one.
func cachedHistorySurges(ctx context.Context, history History, from, to time.Time, ruleID string, now time.Time) (historyRead, error) {
	key := historyKey{history: history, from: from.UnixNano(), to: to.UnixNano(), ruleID: ruleID}
	historyCache.Lock()
	read, ok := historyCache.reads[key]
	historyCache.Unlock()
	if ok && now.Sub(read.readAt) < historyCacheFor {
		return read, nil
	}

	list, err := history.ListTraceSurges(ctx, from, to, ruleID, historyLimit)
	if err != nil {
		return historyRead{}, fmt.Errorf("failed to read ended trace surges from odigos insights: %w", err)
	}
	read = historyRead{surges: historySurges(list), truncated: list.Truncated, readAt: now}

	historyCache.Lock()
	defer historyCache.Unlock()
	for k, r := range historyCache.reads {
		if now.Sub(r.readAt) >= historyCacheFor {
			delete(historyCache.reads, k)
		}
	}
	if _, ok := historyCache.reads[key]; !ok && len(historyCache.reads) >= historyCacheSize {
		var oldest historyKey
		first := true
		for k, r := range historyCache.reads {
			if first || r.readAt.Before(historyCache.reads[oldest].readAt) {
				oldest, first = k, false
			}
		}
		delete(historyCache.reads, oldest)
	}
	historyCache.reads[key] = read
	return read, nil
}

// mergeSurges adds the ended surges to those of the status. The instrumentor puts a surge in the
// history before removing it from the status, so a surge can be in both for a moment: the
// status's is kept.
func mergeSurges(status, ended []v1alpha1.TraceSurge) []v1alpha1.TraceSurge {
	inStatus := make(map[string]bool, len(status))
	for i := range status {
		inStatus[status[i].Name] = true
	}
	for i := range ended {
		if !inStatus[ended[i].Name] {
			status = append(status, ended[i])
		}
	}
	return status
}

// getSurge returns the surge named id, from the status of samplings or the history, and where it
// is recorded.
func getSurge(ctx context.Context, history History, samplings []v1alpha1.Sampling, id string) (*v1alpha1.TraceSurge, string, error) {
	for i := range samplings {
		for j := range samplings[i].Status.TraceSurges {
			if surge := &samplings[i].Status.TraceSurges[j]; surge.Name == id {
				return surge, "The status of Sampling " + samplings[i].Name, nil
			}
		}
	}
	if history == nil {
		return nil, "", errors.New(notFound)
	}
	return historySurge(ctx, history, id)
}

func historySurge(ctx context.Context, history History, id string) (*v1alpha1.TraceSurge, string, error) {
	kept, err := history.GetTraceSurge(ctx, id)
	if errors.Is(err, insights.ErrNotFound) {
		return nil, "", errors.New(notFound)
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to read the trace surge from odigos insights: %w", err)
	}
	var surge v1alpha1.TraceSurge
	if err := json.Unmarshal(kept.Record, &surge); err != nil {
		return nil, "", fmt.Errorf("odigos insights holds an unreadable trace surge %s: %w", id, err)
	}
	return &surge, "Odigos Insights, which keeps ended trace surges for 30 days", nil
}

// ruleRequests describes the requests the noisy operation rule ruleID samples. The id is a hash of
// the rule's scope and operation, so any rule with it has the same operation.
func ruleRequests(samplings []v1alpha1.Sampling, ruleID string) string {
	for i := range samplings {
		for j := range samplings[i].Spec.NoisyOperations {
			if rule := &samplings[i].Spec.NoisyOperations[j]; v1alpha1.ComputeNoisyOperationHash(rule) == ruleID {
				return operationText(rule.Operation)
			}
		}
	}
	return matchingRequests
}

func operationText(op *commonapisampling.HeadSamplingOperationMatcher) string {
	switch {
	case op == nil:
		return "All requests"
	case op.HttpServer != nil:
		route := op.HttpServer.Route
		if route == "" && op.HttpServer.RoutePrefix != "" {
			route = op.HttpServer.RoutePrefix + "*"
		}
		return orDefault(op.HttpServer.Method, "Any method") + " " + orDefault(route, "any route")
	case op.HttpClient != nil:
		path := op.HttpClient.TemplatedPath
		if path == "" && op.HttpClient.TemplatedPathPrefix != "" {
			path = op.HttpClient.TemplatedPathPrefix + "*"
		}
		return "Outgoing " + orDefault(op.HttpClient.Method, "any method") + " to " + orDefault(op.HttpClient.ServerAddress, "any server") + path
	case op.GrpcServer != nil:
		return "gRPC " + orDefault(op.GrpcServer.Service, "any service") + " " + orDefault(op.GrpcServer.Method, "any method")
	case op.GrpcClient != nil:
		return "Outgoing gRPC " + orDefault(op.GrpcClient.Service, "any service") + " " + orDefault(op.GrpcClient.Method, "any method") + " to " + orDefault(op.GrpcClient.ServerAddress, "any server")
	}
	return "All requests"
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func summarize(surge *v1alpha1.TraceSurge) *model.TelemetryActivitySummary {
	settings := surge.Spec.Settings
	metric := string(settings.Metric)

	summary := &model.TelemetryActivitySummary{
		ID:            surge.Name,
		UpdatedAt:     updatedAt(surge).UTC().Format(time.RFC3339),
		Title:         "Head sampling",
		Before:        percent(surge.Spec.NormalPercent),
		After:         percent(settings.BoostPercent),
		TargetSummary: targetSummary(surge.Spec.Targets),
		ScopeSummary:  fmt.Sprintf("New traces of %s, for a spike in %s", ruleName(surge), workloadID(surge.Spec.Service)),
		Source: &model.TelemetrySourceRef{
			Kind:    sourceTraceSurge,
			ID:      surge.Spec.RuleID,
			Name:    ruleName(surge),
			Version: optional(settings.Version),
		},
		Status: string(surge.Status.Phase),
	}

	if trigger := surge.Status.Trigger; trigger != nil {
		requests := int(trigger.Requests)
		measurement := "server spans over the last " + windowText(trigger.WindowSeconds) + ", recorded before sampling"
		scope := scopeOf(surge.Spec.Service)
		scope.Operation = optional(trigger.Operation)
		summary.Trigger = &model.TelemetryMetricTrigger{
			Scope:            scope,
			Metric:           metric,
			ObservedValue:    trigger.Value,
			Threshold:        settings.Threshold,
			SustainedSeconds: settings.SustainedSeconds,
			ObservedAt:       trigger.At.UTC().Format(time.RFC3339),
			RequestCount:     &requests,
			Measurement:      &measurement,
		}
		for _, sample := range surge.Status.TriggerSamples {
			summary.Trigger.Samples = append(summary.Trigger.Samples, &model.TelemetrySample{At: sample.At.UTC().Format(time.RFC3339), Value: sample.Value})
		}
		summary.Reason = fmt.Sprintf("%s in %s stayed above %s for %ds, reaching %s.",
			metricTitle(metric), observedID(surge.Spec.Service, trigger.Operation), formatValue(metric, settings.Threshold), settings.SustainedSeconds, formatValue(metric, trigger.Value))
	}

	switch {
	case surge.Status.BoostedAt == nil:
		// Limited, or ended while Limited: the boost is what the surge asked for, and never got.
		summary.LimitReason = optional(surge.Status.LimitReason)
	case !surge.Active():
		summary.Before, summary.After = percent(settings.BoostPercent), percent(surge.Spec.NormalPercent)
	}
	if surge.Status.Phase == v1alpha1.TraceSurgePhaseRestored && surge.Status.RestoreReason != "" {
		summary.Reason = surge.Status.RestoreReason
	}
	return summary
}

func updatedAt(surge *v1alpha1.TraceSurge) time.Time {
	latest := surge.StartedAt.Time
	for _, event := range surge.Status.Timeline {
		if event.At.After(latest) {
			latest = event.At.Time
		}
	}
	return latest
}

func matchesQuery(s *model.TelemetryActivitySummary, query string) bool {
	query = strings.ToLower(query)
	fields := []string{s.Title, s.TargetSummary, s.ScopeSummary, s.Reason, s.Source.Name}
	if s.Trigger != nil {
		fields = append(fields, s.Trigger.Scope.Service)
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func ruleName(surge *v1alpha1.TraceSurge) string {
	if surge.Spec.RuleName != "" {
		return surge.Spec.RuleName
	}
	return "Collection rule " + surge.Spec.RuleID
}

func targetSummary(targets []v1alpha1.TraceSurgeTarget) string {
	names := make([]string, len(targets))
	for i, t := range targets {
		names[i] = t.Workload.Name
	}
	if len(names) > 3 {
		return fmt.Sprintf("%s and %d more", strings.Join(names[:3], ", "), len(names)-3)
	}
	return strings.Join(names, ", ")
}

// windowText names an evaluation window. Observations recorded before the window was
// configurable have none: they were measured over a minute.
func windowText(seconds int) string {
	if seconds == 0 || seconds == 60 {
		return "minute"
	}
	return (time.Duration(seconds) * time.Second).String()
}

// observedID names what an observation measured: a service, or one of its operations.
func observedID(pw k8sconsts.PodWorkload, operation string) string {
	if operation == "" {
		return workloadID(pw)
	}
	return workloadID(pw) + " " + operation
}

func workloadID(pw k8sconsts.PodWorkload) string {
	return pw.Namespace + "/" + pw.Name
}

func scopeOf(pw k8sconsts.PodWorkload) *model.TelemetryScope {
	return &model.TelemetryScope{ServiceID: workloadID(pw), Service: pw.Name, Namespace: pw.Namespace}
}

func timestamp(t *metav1.Time) *string {
	if t == nil || t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func percent(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64) + "%"
}

func metricTitle(metric string) string {
	switch metric {
	case string(v1alpha1.TraceSurgeMetricErrorRate):
		return "Error rate"
	case string(v1alpha1.TraceSurgeMetricLatencyP95):
		return "p95 latency"
	case string(v1alpha1.TraceSurgeMetricRequestRate):
		return "Request rate"
	}
	return metric
}

func formatValue(metric string, v float64) string {
	switch metric {
	case string(v1alpha1.TraceSurgeMetricErrorRate):
		return percent(v)
	case string(v1alpha1.TraceSurgeMetricLatencyP95):
		return strconv.FormatFloat(v, 'f', -1, 64) + " ms"
	case string(v1alpha1.TraceSurgeMetricRequestRate):
		return strconv.FormatFloat(v, 'f', -1, 64) + " req/s"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
