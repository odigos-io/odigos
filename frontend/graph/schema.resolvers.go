package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.49

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/endpoints"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services"
	testconnection "github.com/odigos-io/odigos/frontend/services/test_connection"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// K8sActualNamespace is the resolver for the k8sActualNamespace field.
func (r *computePlatformResolver) K8sActualNamespace(ctx context.Context, obj *model.ComputePlatform, name string) (*model.K8sActualNamespace, error) {
	namespaceActualSources, err := services.GetWorkloadsInNamespace(ctx, name, nil)
	if err != nil {
		return nil, err
	}

	// Convert namespaceActualSources to []*model.K8sActualSource
	namespaceActualSourcesPointers := make([]*model.K8sActualSource, len(namespaceActualSources))
	for i, source := range namespaceActualSources {
		namespaceActualSourcesPointers[i] = &source
	}

	namespace, err := kube.DefaultClient.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	nsInstrumented := workload.GetInstrumentationLabelValue(namespace.GetLabels())

	return &model.K8sActualNamespace{
		Name:                        name,
		InstrumentationLabelEnabled: nsInstrumented,
		K8sActualSources:            namespaceActualSourcesPointers,
	}, nil
}

// K8sActualSource is the resolver for the k8sActualSource field.
func (r *computePlatformResolver) K8sActualSource(ctx context.Context, obj *model.ComputePlatform, name *string, namespace *string, kind *string) (*model.K8sActualSource, error) {
	source, err := services.GetActualSource(ctx, *namespace, *kind, *name)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, nil
	}
	k8sActualSource := k8sSourceToGql(source)

	return k8sActualSource, nil
}

// Type is the resolver for the type field.
func (r *destinationResolver) Type(ctx context.Context, obj *model.Destination) (string, error) {
	panic(fmt.Errorf("not implemented: Type - type"))
}

// Fields is the resolver for the fields field.
func (r *destinationResolver) Fields(ctx context.Context, obj *model.Destination) ([]string, error) {
	panic(fmt.Errorf("not implemented: Fields - fields"))
}

// Conditions is the resolver for the conditions field.
func (r *destinationResolver) Conditions(ctx context.Context, obj *model.Destination) ([]*model.Condition, error) {
	panic(fmt.Errorf("not implemented: Conditions - conditions"))
}

// K8sActualSources is the resolver for the k8sActualSources field.
func (r *k8sActualNamespaceResolver) K8sActualSources(ctx context.Context, obj *model.K8sActualNamespace, instrumentationLabeled *bool) ([]*model.K8sActualSource, error) {
	namespaceActualSources, err := services.GetWorkloadsInNamespace(ctx, obj.Name, instrumentationLabeled)
	if err != nil {
		return nil, err
	}

	// Convert namespaceActualSources to []*model.K8sActualSource
	namespaceActualSourcesPointers := make([]*model.K8sActualSource, len(namespaceActualSources))
	for i, source := range namespaceActualSources {
		namespaceActualSourcesPointers[i] = &source
	}

	return namespaceActualSourcesPointers, nil
}

// CreateNewDestination is the resolver for the createNewDestination field.
func (r *mutationResolver) CreateNewDestination(ctx context.Context, destination model.DestinationInput) (*model.Destination, error) {
	odigosns := consts.DefaultOdigosNamespace

	destType := common.DestinationType(destination.Type)
	destName := destination.Name

	destTypeConfig, err := services.GetDestinationTypeConfig(destType)
	if err != nil {
		return nil, fmt.Errorf("destination type %s not found", destType)
	}

	// Convert fields to map[string]string
	fieldsMap := make(map[string]string)
	for _, field := range destination.Fields {
		fieldsMap[field.Key] = field.Value
	}

	errors := services.VerifyDestinationDataScheme(destType, destTypeConfig, fieldsMap)
	if len(errors) > 0 {
		return nil, fmt.Errorf("invalid destination data scheme: %v", errors)
	}

	dataField, secretFields := services.TransformFieldsToDataAndSecrets(destTypeConfig, fieldsMap)
	generateNamePrefix := "odigos.io.dest." + string(destType) + "-"

	k8sDestination := v1alpha1.Destination{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateNamePrefix,
		},
		Spec: v1alpha1.DestinationSpec{
			Type:            destType,
			DestinationName: destName,
			Data:            dataField,
			Signals:         services.ExportedSignalsObjectToSlice(destination.ExportedSignals),
		},
	}

	createSecret := len(secretFields) > 0
	if createSecret {
		secretRef, err := services.CreateDestinationSecret(ctx, destType, secretFields, odigosns)
		if err != nil {
			return nil, err
		}
		k8sDestination.Spec.SecretRef = secretRef
	}

	dest, err := kube.DefaultClient.OdigosClient.Destinations(odigosns).Create(ctx, &k8sDestination, metav1.CreateOptions{})
	if err != nil {
		if createSecret {
			kube.DefaultClient.CoreV1().Secrets(odigosns).Delete(ctx, destName, metav1.DeleteOptions{})
		}
		return nil, err
	}

	if dest.Spec.SecretRef != nil {
		err = services.AddDestinationOwnerReferenceToSecret(ctx, odigosns, dest)
		if err != nil {
			return nil, err
		}
	}

	secretFieldsMap, err := services.GetDestinationSecretFields(ctx, odigosns, dest)
	if err != nil {
		return nil, err
	}

	endpointDest := services.K8sDestinationToEndpointFormat(*dest, secretFieldsMap)
	return &endpointDest, nil
}

// PersistK8sNamespace is the resolver for the persistK8sNamespace field.
func (r *mutationResolver) PersistK8sNamespace(ctx context.Context, namespace model.PersistNamespaceItemInput) (bool, error) {
	jsonMergePayload := services.GetJsonMergePatchForInstrumentationLabel(namespace.FutureSelected)
	_, err := kube.DefaultClient.CoreV1().Namespaces().Patch(ctx, namespace.Name, types.MergePatchType, jsonMergePayload, metav1.PatchOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to patch namespace: %v", err)
	}

	return true, nil
}

// PersistK8sSources is the resolver for the persistK8sSources field.
func (r *mutationResolver) PersistK8sSources(ctx context.Context, namespace string, sources []*model.PersistNamespaceSourceInput) (bool, error) {
	var persistObjects []model.PersistNamespaceSourceInput
	for _, source := range sources {
		persistObjects = append(persistObjects, model.PersistNamespaceSourceInput{
			Name:     source.Name,
			Kind:     source.Kind,
			Selected: source.Selected,
		})
	}

	err := services.SyncWorkloadsInNamespace(ctx, namespace, persistObjects)
	if err != nil {
		return false, fmt.Errorf("failed to sync workloads: %v", err)
	}

	return true, nil
}

// TestConnectionForDestination is the resolver for the testConnectionForDestination field.
func (r *mutationResolver) TestConnectionForDestination(ctx context.Context, destination model.DestinationInput) (*model.TestConnectionResponse, error) {
	destType := common.DestinationType(destination.Type)

	destConfig, err := services.GetDestinationTypeConfig(destType)
	if err != nil {
		return nil, err
	}

	if !destConfig.Spec.TestConnectionSupported {
		return nil, fmt.Errorf("destination type %s does not support test connection", destination.Type)
	}

	configurer, err := testconnection.ConvertDestinationToConfigurer(destination)
	if err != nil {
		return nil, err
	}

	res := testconnection.TestConnection(ctx, configurer)
	if !res.Succeeded {
		return &model.TestConnectionResponse{
			Succeeded:       false,
			StatusCode:      res.StatusCode,
			DestinationType: (*string)(&res.DestinationType),
			Message:         &res.Message,
			Reason:          (*string)(&res.Reason),
		}, nil
	}

	return &model.TestConnectionResponse{
		Succeeded:       true,
		StatusCode:      200,
		DestinationType: (*string)(&res.DestinationType),
	}, nil
}

// ComputePlatform is the resolver for the computePlatform field.
func (r *queryResolver) ComputePlatform(ctx context.Context) (*model.ComputePlatform, error) {
	namespacesResponse := services.GetK8SNamespaces(ctx)

	K8sActualNamespaces := make([]*model.K8sActualNamespace, len(namespacesResponse.Namespaces))
	for i, namespace := range namespacesResponse.Namespaces {

		namespace, err := kube.DefaultClient.CoreV1().Namespaces().Get(ctx, namespace.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		nsInstrumented := workload.GetInstrumentationLabelValue(namespace.GetLabels())

		K8sActualNamespaces[i] = &model.K8sActualNamespace{
			Name:                        namespace.Name,
			InstrumentationLabelEnabled: nsInstrumented,
		}
	}

	return &model.ComputePlatform{
		ComputePlatformType: model.ComputePlatformTypeK8s,
		K8sActualNamespaces: K8sActualNamespaces,
	}, nil
}

// Config is the resolver for the config field.
func (r *queryResolver) Config(ctx context.Context) (*model.GetConfigResponse, error) {
	response := endpoints.GetConfig(ctx)

	gqlResponse := &model.GetConfigResponse{
		Installation: model.InstallationStatus(response.Installation),
	}

	return gqlResponse, nil
}

// DestinationTypes is the resolver for the destinationTypes field.
func (r *queryResolver) DestinationTypes(ctx context.Context) (*model.GetDestinationTypesResponse, error) {
	destTypes := services.GetDestinationTypes()

	return &destTypes, nil
}

// DestinationTypeDetails is the resolver for the destinationTypeDetails field.
func (r *queryResolver) DestinationTypeDetails(ctx context.Context, typeArg string) (*model.GetDestinationDetailsResponse, error) {
	destType := common.DestinationType(typeArg)
	destTypeConfig, err := services.GetDestinationTypeConfig(destType)
	if err != nil {
		return nil, fmt.Errorf("destination type %s not found", destType)
	}

	var resp model.GetDestinationDetailsResponse
	for _, field := range destTypeConfig.Spec.Fields {
		componentPropsJSON, err := json.Marshal(field.ComponentProps)
		if err != nil {
			return nil, fmt.Errorf("error marshalling component properties: %v", err)
		}

		resp.Fields = append(resp.Fields, &model.Field{
			Name:                field.Name,
			DisplayName:         field.DisplayName,
			ComponentType:       field.ComponentType,
			ComponentProperties: string(componentPropsJSON),
			InitialValue:        &field.InitialValue,
		})

	}

	return &resp, nil
}

// PotentialDestinations is the resolver for the potentialDestinations field.
func (r *queryResolver) PotentialDestinations(ctx context.Context) ([]*model.DestinationDetails, error) {
	potentialDestinations := services.PotentialDestinations(ctx)
	if potentialDestinations == nil {
		return nil, fmt.Errorf("failed to fetch potential destinations")
	}

	// Convert []destination_recognition.DestinationDetails to []*DestinationDetails
	var result []*model.DestinationDetails
	for _, dest := range potentialDestinations {

		fieldsString, err := json.Marshal(dest.Fields)
		if err != nil {
			return nil, fmt.Errorf("error marshalling fields: %v", err)
		}

		result = append(result, &model.DestinationDetails{
			Type:   string(dest.Type),
			Fields: string(fieldsString),
		})
	}

	return result, nil
}

// Destinations is the resolver for the destinations field.
func (r *queryResolver) Destinations(ctx context.Context) ([]*model.Destination, error) {
	odigosns := consts.DefaultOdigosNamespace
	dests, err := kube.DefaultClient.OdigosClient.Destinations(odigosns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var destinations []*model.Destination
	for _, dest := range dests.Items {
		secretFields, err := services.GetDestinationSecretFields(ctx, odigosns, &dest)
		if err != nil {
			return nil, err
		}

		// Convert the k8s destination format to the expected endpoint format
		endpointDest := services.K8sDestinationToEndpointFormat(dest, secretFields)
		destinations = append(destinations, &endpointDest)
	}

	return destinations, nil
}

// ActualSources is the resolver for the actualSources field.
func (r *queryResolver) ActualSources(ctx context.Context) ([]*model.K8sActualSource, error) {
	instrumentedApplications, err := kube.DefaultClient.OdigosClient.InstrumentedApplications("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Initialize an empty list of K8sActualSource
	var actualSources []*model.K8sActualSource

	// Convert each instrumented application to the K8sActualSource type
	for _, app := range instrumentedApplications.Items {
		actualSource := instrumentedApplicationToActualSource(app)
		actualSources = append(actualSources, actualSource)
	}

	return actualSources, nil
}

// ComputePlatform returns ComputePlatformResolver implementation.
func (r *Resolver) ComputePlatform() ComputePlatformResolver { return &computePlatformResolver{r} }

// Destination returns DestinationResolver implementation.
func (r *Resolver) Destination() DestinationResolver { return &destinationResolver{r} }

// K8sActualNamespace returns K8sActualNamespaceResolver implementation.
func (r *Resolver) K8sActualNamespace() K8sActualNamespaceResolver {
	return &k8sActualNamespaceResolver{r}
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type computePlatformResolver struct{ *Resolver }
type destinationResolver struct{ *Resolver }
type k8sActualNamespaceResolver struct{ *Resolver }
type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
