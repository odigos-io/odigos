package graph

import (
	"context"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/services/insights"
)

func (r *Resolver) resolveIntegrationCatalog(ctx context.Context) ([]*model.InsightsIntegrationDescriptor, error) {
	client, err := r.insightsClient(ctx)
	if err != nil {
		return nil, err
	}
	values, err := client.ListIntegrationCatalog(ctx)
	if err != nil {
		return nil, insights.GraphQLError(ctx, err)
	}
	result := make([]*model.InsightsIntegrationDescriptor, 0, len(values))
	for _, value := range values {
		result = append(result, insights.IntegrationDescriptorToModel(value))
	}
	return result, nil
}
func (r *Resolver) resolveIntegrations(ctx context.Context) ([]*model.InsightsIntegration, error) {
	client, err := r.insightsClient(ctx)
	if err != nil {
		return nil, err
	}
	values, err := client.ListIntegrations(ctx)
	if err != nil {
		return nil, insights.GraphQLError(ctx, err)
	}
	result := make([]*model.InsightsIntegration, 0, len(values))
	for _, value := range values {
		result = append(result, insights.IntegrationToModel(value))
	}
	return result, nil
}
func (r *Resolver) resolveIntegration(ctx context.Context, id string) (*model.InsightsIntegration, error) {
	client, err := r.insightsClient(ctx)
	if err != nil {
		return nil, err
	}
	values, err := client.GetIntegration(ctx, id)
	if err != nil {
		return nil, insights.GraphQLError(ctx, err)
	}
	return insights.IntegrationToModel(*values), nil
}
func (r *Resolver) resolveIntegrationDeliveries(ctx context.Context, integrationID string, state *model.InsightsDeliveryState, limit *int) ([]*model.InsightsDelivery, error) {
	client, err := r.insightsClient(ctx)
	if err != nil {
		return nil, err
	}
	var filter *string
	if state != nil {
		value := string(*state)
		filter = &value
	}
	values, err := client.ListIntegrationDeliveries(ctx, integrationID, filter, limit)
	if err != nil {
		return nil, insights.GraphQLError(ctx, err)
	}
	result := make([]*model.InsightsDelivery, 0, len(values))
	for _, value := range values {
		result = append(result, insights.DeliveryToModel(value))
	}
	return result, nil
}
func (r *Resolver) resolveIntegrationAttempts(ctx context.Context, integrationID string, limit *int) ([]*model.InsightsDeliveryAttempt, error) {
	client, err := r.insightsClient(ctx)
	if err != nil {
		return nil, err
	}
	values, err := client.ListIntegrationAttempts(ctx, integrationID, limit)
	if err != nil {
		return nil, insights.GraphQLError(ctx, err)
	}
	result := make([]*model.InsightsDeliveryAttempt, 0, len(values))
	for _, value := range values {
		result = append(result, insights.DeliveryAttemptToModel(value))
	}
	return result, nil
}
func (r *Resolver) resolveFindingDeliveries(ctx context.Context, findingKey string) ([]*model.InsightsDelivery, error) {
	client, err := r.insightsClient(ctx)
	if err != nil {
		return nil, err
	}
	values, err := client.GetFindingDeliveries(ctx, findingKey)
	if err != nil {
		return nil, insights.GraphQLError(ctx, err)
	}
	result := make([]*model.InsightsDelivery, 0, len(values))
	for _, value := range values {
		result = append(result, insights.DeliveryToModel(value))
	}
	return result, nil
}
func (r *Resolver) resolveCreateInsightsIntegration(ctx context.Context, integration model.InsightsIntegrationInput) (*model.InsightsIntegration, error) {
	client, err := r.insightsClient(ctx)
	if err != nil {
		return nil, err
	}
	input, err := insights.IntegrationFromInput(integration)
	if err != nil {
		return nil, insights.GraphQLError(ctx, err)
	}
	values, err := client.CreateIntegration(ctx, input)
	if err != nil {
		return nil, insights.GraphQLError(ctx, err)
	}
	return insights.IntegrationToModel(*values), nil
}
func (r *Resolver) resolveUpdateInsightsIntegration(ctx context.Context, id string, integration model.InsightsIntegrationInput) (*model.InsightsIntegration, error) {
	client, err := r.insightsClient(ctx)
	if err != nil {
		return nil, err
	}
	input, err := insights.IntegrationFromInput(integration)
	if err != nil {
		return nil, insights.GraphQLError(ctx, err)
	}
	values, err := client.UpdateIntegration(ctx, id, input)
	if err != nil {
		return nil, insights.GraphQLError(ctx, err)
	}
	return insights.IntegrationToModel(*values), nil
}
func (r *Resolver) resolveDeleteInsightsIntegration(ctx context.Context, id string) (bool, error) {
	client, err := r.insightsClient(ctx)
	if err != nil {
		return false, err
	}
	if err := client.DeleteIntegration(ctx, id); err != nil {
		return false, insights.GraphQLError(ctx, err)
	}
	return true, nil
}
func (r *Resolver) resolveTestInsightsIntegration(ctx context.Context, id string) (bool, error) {
	client, err := r.insightsClient(ctx)
	if err != nil {
		return false, err
	}
	if err := client.TestIntegration(ctx, id); err != nil {
		return false, insights.GraphQLError(ctx, err)
	}
	return true, nil
}
func (r *Resolver) resolveRetryInsightsIntegrationDeliveries(ctx context.Context, id string, findingKeys []string) (bool, error) {
	client, err := r.insightsClient(ctx)
	if err != nil {
		return false, err
	}
	if err := client.RetryIntegrationDeliveries(ctx, id, findingKeys); err != nil {
		return false, insights.GraphQLError(ctx, err)
	}
	return true, nil
}
