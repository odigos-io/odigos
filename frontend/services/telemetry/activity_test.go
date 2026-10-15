package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/services/insights"
	"github.com/vektah/gqlparser/v2/gqlerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestChangeDetail(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	boostedAt := metav1.NewTime(now.Add(-5 * time.Minute))
	payments := k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "payments"}
	const triggerReason = "Error rate in shop/payments POST /pay stayed above 5% for 0s, reaching 45%."

	tests := []struct {
		name            string
		phase           v1alpha1.TraceSurgePhase
		boostedAt       *metav1.Time
		restoreReason   string
		wantBefore      string
		wantAfter       string
		wantLimitReason bool
		wantReason      string
		wantNext        string
		wantNextText    string
	}{
		{"limited", v1alpha1.TraceSurgePhaseLimited, nil, "", "1%", "50%", true, triggerReason, "Raise sampling within the limits", "Sampling rises to 50% if the trace surge limits allow it"},
		{"ended while limited", v1alpha1.TraceSurgePhaseRestored, nil, "Error rate went back to 1%.", "1%", "50%", true, "Error rate went back to 1%.", "", ""},
		{"boosting", v1alpha1.TraceSurgePhaseBoosting, &boostedAt, "", "1%", "50%", false, triggerReason, "Return to normal", "Sampling returns to 1%"},
		{"recovering", v1alpha1.TraceSurgePhaseRecovering, &boostedAt, "", "1%", "50%", false, triggerReason, "Return to normal", "Sampling returns to 1%"},
		{"restored", v1alpha1.TraceSurgePhaseRestored, &boostedAt, "Error rate stayed at or below 2% for 60s.", "50%", "1%", false, "Error rate stayed at or below 2% for 60s.", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			surge := &v1alpha1.TraceSurge{
				Name:      "surge",
				StartedAt: metav1.NewTime(now.Add(-10 * time.Minute)),
				Spec: v1alpha1.TraceSurgeSpec{
					Settings:      v1alpha1.TraceSurgeSettings{Metric: "error_rate", Threshold: 5, BoostPercent: 50, RecoveryThreshold: 2, RecoverySeconds: 60},
					NormalPercent: 1,
					Service:       payments,
					Targets:       []v1alpha1.TraceSurgeTarget{{Workload: payments}},
				},
				Status: v1alpha1.TraceSurgeStatus{Phase: tt.phase, BoostedAt: tt.boostedAt, LimitReason: "over the limit", RestoreReason: tt.restoreReason,
					Trigger:  &v1alpha1.TraceSurgeObservation{At: metav1.NewTime(now), Value: 45, Requests: 100, Operation: "POST /pay"},
					Timeline: []v1alpha1.TraceSurgeEvent{{At: metav1.NewTime(now), Title: "Error rate crossed 5%"}}},
			}
			detail := changeDetail(surge, "The status of Sampling default", "POST /pay")

			if detail.Status != string(tt.phase) {
				t.Errorf("status = %q, want the phase %q", detail.Status, tt.phase)
			}
			if detail.Before != tt.wantBefore || detail.After != tt.wantAfter {
				t.Errorf("change %s -> %s, want %s -> %s", detail.Before, detail.After, tt.wantBefore, tt.wantAfter)
			}
			if len(detail.Targets) != 1 {
				t.Fatalf("got %d targets, want 1", len(detail.Targets))
			}
			if target := detail.Targets[0]; target.Before != tt.wantBefore || target.After != tt.wantAfter {
				t.Errorf("target %s -> %s, want %s -> %s", target.Before, target.After, tt.wantBefore, tt.wantAfter)
			}
			if op := detail.Targets[0].Scope.Operation; op == nil || *op != "POST /pay" {
				t.Errorf("target requests = %v, want the rule's", op)
			}
			if got := detail.LimitReason != nil && *detail.LimitReason == "over the limit"; got != tt.wantLimitReason {
				t.Errorf("limit reason = %v, want it only while sampling was never raised", detail.LimitReason)
			}
			if detail.Reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", detail.Reason, tt.wantReason)
			}
			if op := detail.Trigger.Scope.Operation; op == nil || *op != "POST /pay" {
				t.Errorf("trigger operation = %v, want POST /pay", op)
			}
			if first := detail.Timeline[0]; first.At == nil || first.Title != "Error rate crossed 5%" {
				t.Errorf("first event %+v, want the recorded one", first)
			}
			last := detail.Timeline[len(detail.Timeline)-1]
			if tt.wantNext == "" && len(detail.Timeline) != 1 {
				t.Errorf("an ended surge has no next step, got %q", last.Title)
			}
			if tt.wantNext != "" && (last.At != nil || last.Title != tt.wantNext || last.Description == nil || !strings.HasPrefix(*last.Description, tt.wantNextText)) {
				t.Errorf("next step %+v, want %q without a time, described as %q", last, tt.wantNext, tt.wantNextText)
			}
		})
	}
}

type fakeHistory struct {
	items     []insights.TraceSurge
	err       error
	truncated bool
	calls     int
	from, to  time.Time
	ruleID    string
	limit     int
}

func (h *fakeHistory) ListTraceSurges(_ context.Context, from, to time.Time, ruleID string, limit int) (*insights.TraceSurgeList, error) {
	h.calls++
	h.from, h.to, h.ruleID, h.limit = from, to, ruleID, limit
	if h.err != nil {
		return nil, h.err
	}
	return &insights.TraceSurgeList{Items: h.items, Truncated: h.truncated}, nil
}

func (h *fakeHistory) GetTraceSurge(_ context.Context, name string) (*insights.TraceSurge, error) {
	for _, item := range h.items {
		if item.Name == name {
			return &item, nil
		}
	}
	return nil, &insights.APIError{StatusCode: 404, Code: insights.ErrorCodeNotFound}
}

func kept(t *testing.T, surge v1alpha1.TraceSurge) insights.TraceSurge {
	t.Helper()
	record, err := json.Marshal(surge)
	if err != nil {
		t.Fatal(err)
	}
	return insights.TraceSurge{Name: surge.Name, Record: record}
}

func TestHistorySurges(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	surge := func(name string, phase v1alpha1.TraceSurgePhase) v1alpha1.TraceSurge {
		return v1alpha1.TraceSurge{Name: name, StartedAt: metav1.NewTime(start), Spec: v1alpha1.TraceSurgeSpec{RuleID: "abc"},
			Status: v1alpha1.TraceSurgeStatus{Phase: phase}}
	}
	history := &fakeHistory{items: []insights.TraceSurge{
		kept(t, surge("ended", v1alpha1.TraceSurgePhaseRestored)),
		kept(t, surge("moving", v1alpha1.TraceSurgePhaseRestored)),
		{Name: "unreadable", Record: json.RawMessage(`"x"`)},
	}}

	read, err := cachedHistorySurges(ctx, history, start, start.Add(time.Hour), "abc", start)
	if err != nil {
		t.Fatal(err)
	}
	ended := read.surges
	if len(ended) != 2 || history.ruleID != "abc" || history.limit != historyLimit || !history.from.Equal(start) {
		t.Errorf("got %d surges, asked rule %q limit %d from %v", len(ended), history.ruleID, history.limit, history.from)
	}

	status := []v1alpha1.TraceSurge{surge("open", v1alpha1.TraceSurgePhaseBoosting), surge("moving", v1alpha1.TraceSurgePhaseRestored)}
	status[1].Status.RestoreReason = "from the status"
	merged := mergeSurges(status, ended)
	var names []string
	for _, s := range merged {
		names = append(names, s.Name)
	}
	if strings.Join(names, ",") != "open,moving,ended" || merged[1].Status.RestoreReason != "from the status" {
		t.Errorf("merged %v, want each surge once, the status's when it is in both", names)
	}

	got, recordedIn, err := historySurge(ctx, history, "ended")
	if err != nil || got.Name != "ended" || got.Status.Phase != v1alpha1.TraceSurgePhaseRestored || !strings.HasPrefix(recordedIn, "Odigos Insights") {
		t.Errorf("history surge %+v %q %v", got, recordedIn, err)
	}
	if _, _, err := historySurge(ctx, history, "missing"); err == nil || err.Error() != notFound {
		t.Errorf("missing surge: %v, want %q", err, notFound)
	}

	history.err = errors.New("connection refused")
	if _, err := cachedHistorySurges(ctx, history, start, start.Add(time.Hour), "", start); err == nil {
		t.Error("an unreachable history is an error, not an empty one")
	}
}

func TestCachedHistorySurges(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	from, to := now.Add(-time.Hour), now
	history := &fakeHistory{truncated: true}

	first, err := cachedHistorySurges(ctx, history, from, to, "", now)
	if err != nil || !first.truncated {
		t.Fatalf("read %+v %v, want the history's truncation", first, err)
	}
	if _, err := cachedHistorySurges(ctx, history, from, to, "", now.Add(historyCacheFor-time.Second)); err != nil || history.calls != 1 {
		t.Errorf("the next page read the history again: %d reads, %v", history.calls, err)
	}
	if _, err := cachedHistorySurges(ctx, history, from, to, "abc", now); err != nil || history.calls != 2 {
		t.Errorf("another rule's listing must read the history: %d reads, %v", history.calls, err)
	}
	if _, err := cachedHistorySurges(ctx, history, from, to, "", now.Add(historyCacheFor)); err != nil || history.calls != 3 {
		t.Errorf("an expired read must not be served: %d reads, %v", history.calls, err)
	}

	for i := range 2 * historyCacheSize {
		if _, err := cachedHistorySurges(ctx, history, from.Add(time.Duration(i)*time.Second), to, "", now); err != nil {
			t.Fatal(err)
		}
	}
	historyCache.Lock()
	size := len(historyCache.reads)
	historyCache.Unlock()
	if size > historyCacheSize {
		t.Errorf("the cache holds %d reads, want at most %d", size, historyCacheSize)
	}
}

func TestPageOf(t *testing.T) {
	matched := make([]*model.TelemetryActivitySummary, activityPageSize+5)
	for i := range matched {
		matched[i] = &model.TelemetryActivitySummary{ID: strconv.Itoa(i)}
	}
	cursor := func(s string) *string { return &s }

	items, next, err := pageOf(matched, nil)
	if err != nil || len(items) != activityPageSize || next == nil || *next != strconv.Itoa(activityPageSize) {
		t.Fatalf("first page: %d items, next %v, %v", len(items), next, err)
	}
	items, next, err = pageOf(matched, next)
	if err != nil || len(items) != 5 || next != nil || items[0].ID != strconv.Itoa(activityPageSize) {
		t.Errorf("last page: %d items, next %v, %v", len(items), next, err)
	}
	if items, next, err := pageOf(matched, cursor("1000")); err != nil || len(items) != 0 || next != nil {
		t.Errorf("past the end: %d items, next %v, %v", len(items), next, err)
	}
	for _, bad := range []string{"-1", "x", "", "9223372036854775807000", "1.5"} {
		_, _, err := pageOf(matched, cursor(bad))
		var gqlErr *gqlerror.Error
		if !errors.As(err, &gqlErr) || gqlErr.Extensions["code"] != "BAD_USER_INPUT" {
			t.Errorf("cursor %q: %v, want an input error", bad, err)
		}
	}
	if items, _, err := pageOf(matched, cursor(strconv.Itoa(math.MaxInt))); err != nil || len(items) != 0 {
		t.Errorf("the largest cursor: %d items, %v", len(items), err)
	}
}

func TestRuleRequests(t *testing.T) {
	checkout := v1alpha1.NoisyOperation{Operation: &commonapisampling.HeadSamplingOperationMatcher{
		HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Method: "POST", Route: "/api/checkout"}}}
	samplings := []v1alpha1.Sampling{{Spec: v1alpha1.SamplingSpec{NoisyOperations: []v1alpha1.NoisyOperation{{}, checkout}}}}

	if got := ruleRequests(samplings, v1alpha1.ComputeNoisyOperationHash(&checkout)); got != "POST /api/checkout" {
		t.Errorf("requests = %q, want the rule's operation", got)
	}
	if got := ruleRequests(samplings, "gone"); got != matchingRequests {
		t.Errorf("requests of a removed rule = %q, want %q", got, matchingRequests)
	}

	for _, tt := range []struct {
		op   *commonapisampling.HeadSamplingOperationMatcher
		want string
	}{
		{nil, "All requests"},
		{&commonapisampling.HeadSamplingOperationMatcher{HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{RoutePrefix: "/api/"}}, "Any method /api/*"},
		{&commonapisampling.HeadSamplingOperationMatcher{HttpClient: &commonapisampling.HeadSamplingHttpClientOperationMatcher{Method: "GET", ServerAddress: "api.example.com", TemplatedPath: "/orders/{id}"}},
			"Outgoing GET to api.example.com/orders/{id}"},
		{&commonapisampling.HeadSamplingOperationMatcher{GrpcServer: &commonapisampling.HeadSamplingGrpcServerOperationMatcher{Service: "shop.Cart"}}, "gRPC shop.Cart any method"},
	} {
		if got := operationText(tt.op); got != tt.want {
			t.Errorf("operationText = %q, want %q", got, tt.want)
		}
	}
}
