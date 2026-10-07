package tracesurge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

// archiver keeps the trace surges that are over, as the history of the rules.
type archiver interface {
	put(ctx context.Context, surge *odigosv1.TraceSurge) error
}

// insightsArchive keeps ended trace surges in odigos insights. A put of the same surge replaces
// the one kept, so a retry after a failed status write is harmless.
type insightsArchive struct {
	baseURL string
	client  *http.Client
}

// archivedSurge is the body of odigos insights' PUT /api/v1/trace-surges/{name}: the surge, and
// the fields insights lists them by. It ended at its last change, which is the time the telemetry
// activity lists it at.
type archivedSurge struct {
	Name         string               `json:"name"`
	SamplingName string               `json:"sampling_name"`
	RuleID       string               `json:"rule_id"`
	Namespace    string               `json:"namespace"`
	WorkloadKind string               `json:"workload_kind"`
	WorkloadName string               `json:"workload_name"`
	StartedAt    time.Time            `json:"started_at"`
	EndedAt      time.Time            `json:"ended_at"`
	Record       *odigosv1.TraceSurge `json:"record"`
}

func newInsightsArchive(odigosNamespace string) *insightsArchive {
	return &insightsArchive{
		baseURL: k8sconsts.InsightsHTTPEndpoint(odigosNamespace),
		client:  &http.Client{Timeout: 3 * time.Second},
	}
}

func (a *insightsArchive) put(ctx context.Context, surge *odigosv1.TraceSurge) error {
	archived := archivedSurge{
		Name:         surge.Name,
		SamplingName: surge.Spec.SamplingName,
		RuleID:       surge.Spec.RuleID,
		Namespace:    surge.Spec.Service.Namespace,
		WorkloadKind: string(surge.Spec.Service.Kind),
		WorkloadName: surge.Spec.Service.Name,
		StartedAt:    surge.StartedAt.Time,
		EndedAt:      lastChange(surge),
		Record:       surge,
	}
	body, err := json.Marshal(archived)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, a.baseURL+"/api/v1/trace-surges/"+url.PathEscape(surge.Name), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("put trace surge to insights: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("put trace surge to insights: %s", resp.Status)
	}
	return nil
}

// lastChange is when the surge last changed: when sampling was restored, or, later, when its
// targets were confirmed back. The telemetry activity lists surges by it.
func lastChange(surge *odigosv1.TraceSurge) time.Time {
	last := surge.StartedAt.Time
	if surge.Status.RestoredAt != nil && surge.Status.RestoredAt.After(last) {
		last = surge.Status.RestoredAt.Time
	}
	for _, event := range surge.Status.Timeline {
		if event.At.After(last) {
			last = event.At.Time
		}
	}
	return last
}
