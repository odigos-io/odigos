package graph

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/frontend/services/insights"
	"github.com/odigos-io/odigos/frontend/services/telemetry"
)

// insightsClient is the gate every insights resolver goes through: insights is
// an optional feature, and the odigos-insights service it talks to only runs in
// the cluster when the effective configuration enables it. Returns an error that
// is ready to hand back from a resolver.
func (r *Resolver) insightsClient(ctx context.Context) (*insights.Client, error) {
	enabled, err := insights.IsEnabled(ctx, r.K8sCacheClient)
	if err != nil {
		return nil, insights.GraphQLError(ctx, fmt.Errorf("reading effective config for insights: %w", err))
	}
	if !enabled {
		return nil, insights.GraphQLError(ctx, insights.ErrNotEnabled)
	}
	if r.InsightsClient == nil {
		return nil, insights.GraphQLError(ctx, fmt.Errorf("%w: insights client was not initialized", insights.ErrInternal))
	}
	return r.InsightsClient, nil
}

// traceSurgeHistory is where the telemetry activity reads ended trace surges: odigos insights, or
// nil when it is not enabled. Trace surges evaluate the span metrics insights stores, so without
// it there are none.
func (r *Resolver) traceSurgeHistory(ctx context.Context) (telemetry.History, error) {
	enabled, err := insights.IsEnabled(ctx, r.K8sCacheClient)
	if err != nil {
		return nil, insights.GraphQLError(ctx, fmt.Errorf("reading effective config for insights: %w", err))
	}
	if !enabled {
		return nil, nil
	}
	if r.InsightsClient == nil {
		return nil, insights.GraphQLError(ctx, fmt.Errorf("%w: insights client was not initialized", insights.ErrInternal))
	}
	return r.InsightsClient, nil
}
