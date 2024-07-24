package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.49

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/frontend/endpoints"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

// K8sActualSource is the resolver for the k8sActualSource field.
func (r *computePlatformResolver) K8sActualSource(ctx context.Context, obj *model.ComputePlatform, name *string, namespace *string, kind *string) (*model.K8sActualSource, error) {
	source, err := endpoints.GetActualSource(ctx, *namespace, *kind, *name)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, nil
	}
	k8sActualSource := k8sSourceToGql(source)

	return k8sActualSource, nil
}

// ApplyDesiredNamespace is the resolver for the applyDesiredNamespace field.
func (r *mutationResolver) ApplyDesiredNamespace(ctx context.Context, cpID string, nsID model.K8sNamespaceID, ns model.K8sDesiredNamespaceInput) (bool, error) {
	panic(fmt.Errorf("not implemented: ApplyDesiredNamespace - applyDesiredNamespace"))
}

// DeleteDesiredNamespace is the resolver for the deleteDesiredNamespace field.
func (r *mutationResolver) DeleteDesiredNamespace(ctx context.Context, cpID string, nsID model.K8sNamespaceID) (bool, error) {
	panic(fmt.Errorf("not implemented: DeleteDesiredNamespace - deleteDesiredNamespace"))
}

// ApplyDesiredSource is the resolver for the applyDesiredSource field.
func (r *mutationResolver) ApplyDesiredSource(ctx context.Context, cpID string, sourceID model.K8sSourceID, source model.K8sDesiredSourceInput) (bool, error) {
	panic(fmt.Errorf("not implemented: ApplyDesiredSource - applyDesiredSource"))
}

// DeleteDesiredSource is the resolver for the deleteDesiredSource field.
func (r *mutationResolver) DeleteDesiredSource(ctx context.Context, cpID string, sourceID model.K8sSourceID) (bool, error) {
	panic(fmt.Errorf("not implemented: DeleteDesiredSource - deleteDesiredSource"))
}

// CreateDesiredDestination is the resolver for the createDesiredDestination field.
func (r *mutationResolver) CreateDesiredDestination(ctx context.Context, destinationType string, destination model.DesiredDestinationInput) (*model.DesiredDestination, error) {
	panic(fmt.Errorf("not implemented: CreateDesiredDestination - createDesiredDestination"))
}

// UpdateDesiredDestination is the resolver for the updateDesiredDestination field.
func (r *mutationResolver) UpdateDesiredDestination(ctx context.Context, destinationID string, destination model.DesiredDestinationInput) (*model.DesiredDestination, error) {
	panic(fmt.Errorf("not implemented: UpdateDesiredDestination - updateDesiredDestination"))
}

// DeleteDesiredDestination is the resolver for the deleteDesiredDestination field.
func (r *mutationResolver) DeleteDesiredDestination(ctx context.Context, destinationID string) (bool, error) {
	panic(fmt.Errorf("not implemented: DeleteDesiredDestination - deleteDesiredDestination"))
}

// ApplyDesiredDestinationToComputePlatform is the resolver for the applyDesiredDestinationToComputePlatform field.
func (r *mutationResolver) ApplyDesiredDestinationToComputePlatform(ctx context.Context, cpID string, destinationID string) (bool, error) {
	panic(fmt.Errorf("not implemented: ApplyDesiredDestinationToComputePlatform - applyDesiredDestinationToComputePlatform"))
}

// RemoveDesiredDestinationFromComputePlatform is the resolver for the removeDesiredDestinationFromComputePlatform field.
func (r *mutationResolver) RemoveDesiredDestinationFromComputePlatform(ctx context.Context, cpID string, destinationID string) (bool, error) {
	panic(fmt.Errorf("not implemented: RemoveDesiredDestinationFromComputePlatform - removeDesiredDestinationFromComputePlatform"))
}

// CreateDesiredAction is the resolver for the createDesiredAction field.
func (r *mutationResolver) CreateDesiredAction(ctx context.Context, cpID string, action model.DesiredActionInput) (bool, error) {
	panic(fmt.Errorf("not implemented: CreateDesiredAction - createDesiredAction"))
}

// UpdateDesiredAction is the resolver for the updateDesiredAction field.
func (r *mutationResolver) UpdateDesiredAction(ctx context.Context, cpID string, actionID string, action model.DesiredActionInput) (bool, error) {
	panic(fmt.Errorf("not implemented: UpdateDesiredAction - updateDesiredAction"))
}

// DeleteActualAction is the resolver for the deleteActualAction field.
func (r *mutationResolver) DeleteActualAction(ctx context.Context, cpID string, actionID string, kind string) (bool, error) {
	panic(fmt.Errorf("not implemented: DeleteActualAction - deleteActualAction"))
}

// ComputePlatform is the resolver for the computePlatform field.
func (r *queryResolver) ComputePlatform(ctx context.Context, cpID string) (*model.ComputePlatform, error) {
	k8sActualSources := endpoints.GetActualSources(ctx, "odigos-system")
	res := make([]*model.K8sActualSource, len(k8sActualSources))
	for i, source := range k8sActualSources {
		res[i] = k8sThinSourceToGql(&source)
	}

	return &model.ComputePlatform{
		K8sActualSources: res,
	}, nil
}

// DestinationTypeCategories is the resolver for the destinationTypeCategories field.
func (r *queryResolver) DestinationTypeCategories(ctx context.Context) ([]*model.DestinationTypeCategory, error) {
	panic(fmt.Errorf("not implemented: DestinationTypeCategories - destinationTypeCategories"))
}

// DesiredDestinations is the resolver for the desiredDestinations field.
func (r *queryResolver) DesiredDestinations(ctx context.Context) ([]*model.DesiredDestination, error) {
	panic(fmt.Errorf("not implemented: DesiredDestinations - desiredDestinations"))
}

// ComputePlatform returns ComputePlatformResolver implementation.
func (r *Resolver) ComputePlatform() ComputePlatformResolver { return &computePlatformResolver{r} }

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type computePlatformResolver struct{ *Resolver }
type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//   - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//     it when you're done.
//   - You have helper methods in this file. Move them out to keep these resolver files clean.
func (r *computePlatformResolver) K8sActualSources(ctx context.Context, obj *model.ComputePlatform) ([]*model.K8sActualSource, error) {
	// thinSource, err := endpoints.GetActualSource(ctx, *namespace, *kind, *name)
	// if err != nil {
	// 	return nil, err
	// }
	// k8sActualSource := k8sSourceToGql(thinSource)

	// return k8sActualSources, nil
	return obj.K8sActualSources, nil
}
