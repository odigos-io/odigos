package services

import (
	"context"
	"encoding/json"
	"fmt"

	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func deriveTypeFromAction(action *model.Action) model.ActionType {
	if action.Fields.CollectContainerAttributes != nil || action.Fields.CollectReplicaSetAttributes != nil || action.Fields.CollectWorkloadID != nil || action.Fields.CollectClusterID != nil || action.Fields.LabelsAttributes != nil || action.Fields.AnnotationsAttributes != nil {
		return model.ActionTypeK8sAttributesResolver
	}
	if action.Fields.ClusterAttributes != nil || action.Fields.OverwriteExistingValues != nil {
		return model.ActionTypeAddClusterInfo
	}
	if action.Fields.AttributeNamesToDelete != nil {
		return model.ActionTypeDeleteAttribute
	}
	if action.Fields.Renames != nil {
		return model.ActionTypeRenameAttribute
	}
	if action.Fields.PiiCategories != nil {
		return model.ActionTypePiiMasking
	}
	if action.Fields.SamplingPercentage != nil {
		return model.ActionTypeProbabilisticSampler
	}
	if action.Fields.FallbackSamplingRatio != nil {
		return model.ActionTypeErrorSampler
	}
	if len(action.Fields.EndpointsFilters) > 0 {
		return model.ActionTypeLatencySampler
	}
	if len(action.Fields.ServicesNameFilters) > 0 {
		return model.ActionTypeServiceNameSampler
	}
	if len(action.Fields.AttributeFilters) > 0 {
		return model.ActionTypeSpanAttributeSampler
	}

	return model.ActionTypeUnknownType
}

func GetAction(ctx context.Context, id string) (*model.Action, error) {
	odigosNs := env.GetCurrentNamespace()

	action, err := kube.DefaultClient.OdigosClient.Actions(odigosNs).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("action with ID %s not found", id)
		}
		return nil, fmt.Errorf("failed to get action: %v", err)
	}

	convertedAction, err := convertActionToModel(action)
	if err != nil {
		return nil, fmt.Errorf("failed to convert action to model: %v", err)
	}
	return convertedAction, nil
}

func GetActions(ctx context.Context) ([]*model.Action, error) {
	odigosNs := env.GetCurrentNamespace()

	actions, err := kube.DefaultClient.OdigosClient.Actions(odigosNs).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get actions: %v", err)
	}

	var response []*model.Action
	for _, action := range actions.Items {
		convertedAction, err := convertActionToModel(&action)
		if err != nil {
			return nil, fmt.Errorf("failed to convert action to model: %v", err)
		}
		response = append(response, convertedAction)
	}

	return response, nil
}

func CreateAction(ctx context.Context, input model.ActionInput) (*model.Action, error) {
	odigosNs := env.GetCurrentNamespace()

	spec, err := getSpecFromInput(input, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get spec from input: %v", err)
	}

	payload := &v1alpha1.Action{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "action-",
		},
		Spec: *spec,
	}

	createdAction, err := kube.DefaultClient.OdigosClient.Actions(odigosNs).Create(ctx, payload, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create action: %v", err)
	}

	response, err := convertActionToModel(createdAction)
	if err != nil {
		return nil, fmt.Errorf("failed to convert action to model: %v", err)
	}

	return response, nil
}

func UpdateAction(ctx context.Context, id string, input model.ActionInput) (*model.Action, error) {
	odigosNs := env.GetCurrentNamespace()

	existingAction, err := kube.DefaultClient.OdigosClient.Actions(odigosNs).Get(ctx, id, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to fetch action: %v", err)
	}

	spec, err := getSpecFromInput(input, existingAction)
	if err != nil {
		return nil, fmt.Errorf("failed to get spec from input: %v", err)
	}
	existingAction.Spec = *spec

	updatedAction, err := kube.DefaultClient.OdigosClient.Actions(odigosNs).Update(ctx, existingAction, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update action: %v", err)
	}

	response, err := convertActionToModel(updatedAction)
	if err != nil {
		return nil, fmt.Errorf("failed to convert action to model: %v", err)
	}

	return response, nil
}

func DeleteAction(ctx context.Context, id string) (bool, error) {
	odigosNs := env.GetCurrentNamespace()

	err := kube.DefaultClient.OdigosClient.Actions(odigosNs).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, fmt.Errorf("action with ID %s not found", id)
		}
		return false, fmt.Errorf("failed to delete action: %v", err)
	}

	return true, nil
}

func getSpecFromInput(input model.ActionInput, existingAction *v1alpha1.Action) (*v1alpha1.ActionSpec, error) {
	var spec v1alpha1.ActionSpec
	if existingAction != nil {
		spec = existingAction.Spec
	} else {
		spec = v1alpha1.ActionSpec{}
	}

	signals, err := ConvertSignals(input.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	spec.ActionName = DerefString(input.Name)
	spec.Notes = DerefString(input.Notes)
	spec.Disabled = input.Disabled
	spec.Signals = signals

	spec.K8sAttributes = convertK8sAttributesFromInput(input.Fields, existingAction)
	spec.AddClusterInfo = convertAddClusterInfoFromInput(input.Fields, existingAction)
	spec.DeleteAttribute = convertDeleteAttributeFromInput(input.Fields, existingAction)

	renameAttribute, err := convertRenameAttributeFromInput(input.Fields, existingAction)
	if err != nil {
		return nil, fmt.Errorf("failed to convert rename attribute: %v", err)
	}
	spec.RenameAttribute = renameAttribute

	spec.PiiMasking = convertPiiMaskingFromInput(input.Fields, existingAction)
	spec.Samplers = convertSamplersFromInput(input.Fields, existingAction)

	return &spec, nil
}

func convertK8sAttributesFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *actionsv1.K8sAttributesConfig {
	withK8sAttributes := false
	var config *actionsv1.K8sAttributesConfig

	if details.CollectContainerAttributes != nil ||
		details.CollectReplicaSetAttributes != nil ||
		details.CollectWorkloadID != nil ||
		details.CollectClusterID != nil ||
		details.LabelsAttributes != nil ||
		details.AnnotationsAttributes != nil {

		config = &actionsv1.K8sAttributesConfig{}

		if details.CollectContainerAttributes != nil {
			config.CollectContainerAttributes = *details.CollectContainerAttributes
			withK8sAttributes = true
		}
		if details.CollectReplicaSetAttributes != nil {
			config.CollectReplicaSetAttributes = *details.CollectReplicaSetAttributes
			withK8sAttributes = true
		}
		if details.CollectWorkloadID != nil {
			config.CollectWorkloadUID = *details.CollectWorkloadID
			withK8sAttributes = true
		}
		if details.CollectClusterID != nil {
			config.CollectClusterUID = *details.CollectClusterID
			withK8sAttributes = true
		}
		if details.LabelsAttributes != nil {
			config.LabelsAttributes = make([]actionsv1.K8sLabelAttribute, len(details.LabelsAttributes))
			for i, attr := range details.LabelsAttributes {
				config.LabelsAttributes[i] = actionsv1.K8sLabelAttribute{
					LabelKey:     attr.LabelKey,
					AttributeKey: attr.AttributeKey,
				}
				if attr.From != nil {
					from := actionsv1.K8sAttributeSource(*attr.From)
					config.LabelsAttributes[i].From = &from
				}
				if len(attr.FromSources) > 0 {
					config.LabelsAttributes[i].FromSources = make([]actionsv1.K8sAttributeSource, len(attr.FromSources))
					for j, source := range attr.FromSources {
						config.LabelsAttributes[i].FromSources[j] = actionsv1.K8sAttributeSource(source)
					}
				}
			}
			withK8sAttributes = true
		}
		if details.AnnotationsAttributes != nil {
			config.AnnotationsAttributes = make([]actionsv1.K8sAnnotationAttribute, len(details.AnnotationsAttributes))
			for i, attr := range details.AnnotationsAttributes {
				config.AnnotationsAttributes[i] = actionsv1.K8sAnnotationAttribute{
					AnnotationKey: attr.AnnotationKey,
					AttributeKey:  attr.AttributeKey,
				}
				if attr.From != nil {
					from := string(*attr.From)
					config.AnnotationsAttributes[i].From = &from
				}
				if len(attr.FromSources) > 0 {
					config.AnnotationsAttributes[i].FromSources = make([]actionsv1.K8sAttributeSource, len(attr.FromSources))
					for j, source := range attr.FromSources {
						config.AnnotationsAttributes[i].FromSources[j] = actionsv1.K8sAttributeSource(source)
					}
				}
			}
			withK8sAttributes = true
		}
	}

	if !withK8sAttributes {
		if existingAction != nil && existingAction.Spec.K8sAttributes != nil {
			return existingAction.Spec.K8sAttributes
		}
		return nil
	}

	return config
}

func convertAddClusterInfoFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *actionsv1.AddClusterInfoConfig {
	withAddClusterInfo := false
	var config *actionsv1.AddClusterInfoConfig

	if details.ClusterAttributes != nil || details.OverwriteExistingValues != nil {
		config = &actionsv1.AddClusterInfoConfig{}

		if details.ClusterAttributes != nil {
			config.ClusterAttributes = make([]actionsv1.OtelAttributeWithValue, len(details.ClusterAttributes))
			for i, attr := range details.ClusterAttributes {
				config.ClusterAttributes[i] = actionsv1.OtelAttributeWithValue{
					AttributeName:        attr.AttributeName,
					AttributeStringValue: &attr.AttributeStringValue,
				}
			}
			withAddClusterInfo = true
		}
		if details.OverwriteExistingValues != nil {
			config.OverwriteExistingValues = *details.OverwriteExistingValues
			withAddClusterInfo = true
		}
	}

	if !withAddClusterInfo {
		if existingAction != nil && existingAction.Spec.AddClusterInfo != nil {
			return existingAction.Spec.AddClusterInfo
		}
		return nil
	}

	return config
}

func convertDeleteAttributeFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *actionsv1.DeleteAttributeConfig {
	withDeleteAttribute := false
	var config *actionsv1.DeleteAttributeConfig

	if details.AttributeNamesToDelete != nil {
		config = &actionsv1.DeleteAttributeConfig{
			AttributeNamesToDelete: details.AttributeNamesToDelete,
		}
		withDeleteAttribute = true
	}

	if !withDeleteAttribute {
		if existingAction != nil && existingAction.Spec.DeleteAttribute != nil {
			return existingAction.Spec.DeleteAttribute
		}
		return nil
	}

	return config
}

func convertRenameAttributeFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) (*actionsv1.RenameAttributeConfig, error) {
	withRenameAttribute := false
	var config *actionsv1.RenameAttributeConfig

	if details.Renames != nil {
		config = &actionsv1.RenameAttributeConfig{}

		var renamesMap map[string]string
		err := json.Unmarshal([]byte(*details.Renames), &renamesMap)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal renames: %v", err)
		}
		config.Renames = renamesMap
		withRenameAttribute = true
	}

	if !withRenameAttribute {
		if existingAction != nil && existingAction.Spec.RenameAttribute != nil {
			return existingAction.Spec.RenameAttribute, nil
		}
		return nil, nil
	}

	return config, nil
}

func convertPiiMaskingFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *actionsv1.PiiMaskingConfig {
	withPiiMasking := false
	var config *actionsv1.PiiMaskingConfig

	if details.PiiCategories != nil {
		config = &actionsv1.PiiMaskingConfig{}

		piiCategories := make([]actionsv1.PiiCategory, len(details.PiiCategories))
		for i, cat := range details.PiiCategories {
			piiCategories[i] = actionsv1.PiiCategory(cat)
		}
		config.PiiCategories = piiCategories
		withPiiMasking = true
	}

	if !withPiiMasking {
		if existingAction != nil && existingAction.Spec.PiiMasking != nil {
			return existingAction.Spec.PiiMasking
		}
		return nil
	}

	return config
}

func convertSamplersFromInput(details *model.ActionFieldsInput, existingAction *v1alpha1.Action) *actionsv1.SamplersConfig {
	withAnySampler := false
	var config *actionsv1.SamplersConfig

	// Check if any sampler fields are provided
	if details.SamplingPercentage != nil ||
		details.FallbackSamplingRatio != nil ||
		details.EndpointsFilters != nil ||
		details.ServicesNameFilters != nil ||
		details.AttributeFilters != nil {

		config = &actionsv1.SamplersConfig{}

		// Convert ProbabilisticSampler
		withProbabilisticSampler := false
		if details.SamplingPercentage != nil {
			config.ProbabilisticSampler = &actionsv1.ProbabilisticSamplerConfig{
				SamplingPercentage: *details.SamplingPercentage,
			}
			withProbabilisticSampler = true
			withAnySampler = true
		}
		if !withProbabilisticSampler {
			if existingAction != nil && existingAction.Spec.Samplers != nil && existingAction.Spec.Samplers.ProbabilisticSampler != nil {
				config.ProbabilisticSampler = existingAction.Spec.Samplers.ProbabilisticSampler
			}
		}

		// Convert ErrorSampler
		withErrorSampler := false
		if details.FallbackSamplingRatio != nil {
			config.ErrorSampler = &actionsv1.ErrorSamplerConfig{
				FallbackSamplingRatio: float64(*details.FallbackSamplingRatio),
			}
			withErrorSampler = true
			withAnySampler = true
		}
		if !withErrorSampler {
			if existingAction != nil && existingAction.Spec.Samplers != nil && existingAction.Spec.Samplers.ErrorSampler != nil {
				config.ErrorSampler = existingAction.Spec.Samplers.ErrorSampler
			}
		}

		// Convert LatencySampler
		withLatencySampler := false
		if details.EndpointsFilters != nil {
			config.LatencySampler = &actionsv1.LatencySamplerConfig{}
			config.LatencySampler.EndpointsFilters = make([]actionsv1.HttpRouteFilter, len(details.EndpointsFilters))
			for i, f := range details.EndpointsFilters {
				config.LatencySampler.EndpointsFilters[i] = actionsv1.HttpRouteFilter{
					HttpRoute:               f.HTTPRoute,
					ServiceName:             f.ServiceName,
					MinimumLatencyThreshold: f.MinimumLatencyThreshold,
					FallbackSamplingRatio:   f.FallbackSamplingRatio,
				}
			}
			withLatencySampler = true
			withAnySampler = true
		}
		if !withLatencySampler {
			if existingAction != nil && existingAction.Spec.Samplers != nil && existingAction.Spec.Samplers.LatencySampler != nil {
				config.LatencySampler = existingAction.Spec.Samplers.LatencySampler
			}
		}

		// Convert ServiceNameSampler
		withServiceNameSampler := false
		if details.ServicesNameFilters != nil {
			config.ServiceNameSampler = &actionsv1.ServiceNameSamplerConfig{}
			config.ServiceNameSampler.ServicesNameFilters = make([]actionsv1.ServiceNameFilter, len(details.ServicesNameFilters))
			for i, f := range details.ServicesNameFilters {
				config.ServiceNameSampler.ServicesNameFilters[i] = actionsv1.ServiceNameFilter{
					ServiceName:           f.ServiceName,
					SamplingRatio:         f.SamplingRatio,
					FallbackSamplingRatio: f.FallbackSamplingRatio,
				}
			}
			withServiceNameSampler = true
			withAnySampler = true
		}
		if !withServiceNameSampler {
			if existingAction != nil && existingAction.Spec.Samplers != nil && existingAction.Spec.Samplers.ServiceNameSampler != nil {
				config.ServiceNameSampler = existingAction.Spec.Samplers.ServiceNameSampler
			}
		}

		// Convert SpanAttributeSampler
		withSpanAttributeSampler := false
		if details.AttributeFilters != nil {
			config.SpanAttributeSampler = &actionsv1.SpanAttributeSamplerConfig{}
			config.SpanAttributeSampler.AttributeFilters = make([]actionsv1.SpanAttributeFilter, len(details.AttributeFilters))
			for i, f := range details.AttributeFilters {
				config.SpanAttributeSampler.AttributeFilters[i] = actionsv1.SpanAttributeFilter{
					ServiceName:           f.ServiceName,
					AttributeKey:          f.AttributeKey,
					FallbackSamplingRatio: f.FallbackSamplingRatio,
					Condition:             *convertConditionFromInput(f.Condition),
				}
			}
			withSpanAttributeSampler = true
			withAnySampler = true
		}
		if !withSpanAttributeSampler {
			if existingAction != nil && existingAction.Spec.Samplers != nil && existingAction.Spec.Samplers.SpanAttributeSampler != nil {
				config.SpanAttributeSampler = existingAction.Spec.Samplers.SpanAttributeSampler
			}
		}
	}

	if !withAnySampler {
		if existingAction != nil && existingAction.Spec.Samplers != nil {
			return existingAction.Spec.Samplers
		}
		return nil
	}

	return config
}

func convertConditionFromInput(condition *model.AttributeFiltersConditionInput) *actionsv1.AttributeCondition {
	if condition == nil {
		return nil
	}
	return &actionsv1.AttributeCondition{
		StringCondition:  convertStringConditionFromInput(condition.StringCondition),
		NumberCondition:  convertNumberConditionFromInput(condition.NumberCondition),
		BooleanCondition: convertBooleanConditionFromInput(condition.BooleanCondition),
		JsonCondition:    convertJSONConditionFromInput(condition.JSONCondition),
	}
}

func convertStringConditionFromInput(condition *model.StringConditionInput) *actionsv1.StringAttributeCondition {
	if condition == nil {
		return nil
	}
	return &actionsv1.StringAttributeCondition{
		Operation:     string(condition.Operation),
		ExpectedValue: *condition.ExpectedValue,
	}
}

func convertNumberConditionFromInput(condition *model.NumberConditionInput) *actionsv1.NumberAttributeCondition {
	if condition == nil {
		return nil
	}
	return &actionsv1.NumberAttributeCondition{
		Operation:     string(condition.Operation),
		ExpectedValue: condition.ExpectedValue,
	}
}

func convertBooleanConditionFromInput(condition *model.BooleanConditionInput) *actionsv1.BooleanAttributeCondition {
	if condition == nil {
		return nil
	}
	return &actionsv1.BooleanAttributeCondition{
		Operation:     string(condition.Operation),
		ExpectedValue: condition.ExpectedValue,
	}
}

func convertJSONConditionFromInput(condition *model.JSONConditionInput) *actionsv1.JsonAttributeCondition {
	if condition == nil {
		return nil
	}
	return &actionsv1.JsonAttributeCondition{
		Operation:     string(condition.Operation),
		ExpectedValue: *condition.ExpectedValue,
		JsonPath:      *condition.JSONPath,
	}
}

func convertActionToModel(action *v1alpha1.Action) (*model.Action, error) {
	var labelAttrs []*model.K8sLabelAttribute
	if action.Spec.K8sAttributes != nil {
		labelAttrs = convertLabelsAttributesToModel(action.Spec.K8sAttributes.LabelsAttributes)
	}

	var annotAttrs []*model.K8sAnnotationAttribute
	if action.Spec.K8sAttributes != nil {
		annotAttrs = convertAnnotationsAttributesToModel(action.Spec.K8sAttributes.AnnotationsAttributes)
	}

	var clustAttrs []*model.ClusterAttribute
	if action.Spec.AddClusterInfo != nil {
		clustAttrs = convertClusterAttributesToModel(action.Spec.AddClusterInfo.ClusterAttributes)
	}

	var renames *string
	if action.Spec.RenameAttribute != nil {
		stringified, err := stringifyMap(action.Spec.RenameAttribute.Renames)
		if err != nil {
			return nil, fmt.Errorf("failed to stringify renames: %v", err)
		}
		renames = &stringified
	}

	var piiCategories []string
	if action.Spec.PiiMasking != nil {
		piiCategories = convertPiiCategoriesToModel(action.Spec.PiiMasking.PiiCategories)
	}

	var fallbackSamplingRatio *int
	if action.Spec.Samplers != nil && action.Spec.Samplers.ErrorSampler != nil {
		intified := int(action.Spec.Samplers.ErrorSampler.FallbackSamplingRatio)
		fallbackSamplingRatio = &intified
	}

	var endpointsFilters []*model.HTTPRouteFilter
	if action.Spec.Samplers != nil && action.Spec.Samplers.LatencySampler != nil {
		endpointsFilters = convertEndpointsFiltersToModel(action.Spec.Samplers.LatencySampler.EndpointsFilters)
	}

	var servicesNameFilters []*model.ServiceNameFilter
	if action.Spec.Samplers != nil && action.Spec.Samplers.ServiceNameSampler != nil {
		servicesNameFilters = convertServiceNameFiltersToModel(action.Spec.Samplers.ServiceNameSampler.ServicesNameFilters)
	}

	var attributeFilters []*model.SpanAttributeFilter
	if action.Spec.Samplers != nil && action.Spec.Samplers.SpanAttributeSampler != nil {
		attributeFilters = convertAttributeFiltersToModel(action.Spec.Samplers.SpanAttributeSampler.AttributeFilters)
	}

	responseFields := &model.ActionFields{
		LabelsAttributes:      labelAttrs,
		AnnotationsAttributes: annotAttrs,
		ClusterAttributes:     clustAttrs,
		Renames:               renames,
		PiiCategories:         piiCategories,
		FallbackSamplingRatio: fallbackSamplingRatio,
		EndpointsFilters:      endpointsFilters,
		ServicesNameFilters:   servicesNameFilters,
		AttributeFilters:      attributeFilters,
	}

	// Handle K8sAttributes fields
	if action.Spec.K8sAttributes != nil {
		responseFields.CollectContainerAttributes = &action.Spec.K8sAttributes.CollectContainerAttributes
		responseFields.CollectReplicaSetAttributes = &action.Spec.K8sAttributes.CollectReplicaSetAttributes
		responseFields.CollectWorkloadID = &action.Spec.K8sAttributes.CollectWorkloadUID
		responseFields.CollectClusterID = &action.Spec.K8sAttributes.CollectClusterUID
	}

	// Handle AddClusterInfo fields
	if action.Spec.AddClusterInfo != nil {
		responseFields.OverwriteExistingValues = &action.Spec.AddClusterInfo.OverwriteExistingValues
	}

	// Handle DeleteAttribute fields
	if action.Spec.DeleteAttribute != nil {
		responseFields.AttributeNamesToDelete = action.Spec.DeleteAttribute.AttributeNamesToDelete
	}

	// Handle Samplers fields
	if action.Spec.Samplers != nil && action.Spec.Samplers.ProbabilisticSampler != nil {
		responseFields.SamplingPercentage = &action.Spec.Samplers.ProbabilisticSampler.SamplingPercentage
	}

	signals := []model.SignalType{}
	for _, signal := range action.Spec.Signals {
		signals = append(signals, model.SignalType(signal))
	}

	response := &model.Action{
		ID:       action.Name,
		Name:     &action.Spec.ActionName,
		Notes:    &action.Spec.Notes,
		Disabled: action.Spec.Disabled,
		Signals:  signals,
		Fields:   responseFields,
	}

	response.Type = deriveTypeFromAction(response)
	response.Conditions = ConvertConditions(action.Status.Conditions)

	return response, nil
}

func convertLabelsAttributesToModel(labelsAttributes []actionsv1.K8sLabelAttribute) []*model.K8sLabelAttribute {
	var result []*model.K8sLabelAttribute

	for _, attr := range labelsAttributes {
		var from *model.K8sAttributesFrom
		if attr.From != nil {
			tmp := model.K8sAttributesFrom(*attr.From)
			from = &tmp
		}
		var fromSources []model.K8sAttributesFrom
		if len(attr.FromSources) > 0 {
			fromSources = make([]model.K8sAttributesFrom, len(attr.FromSources))
			for i, source := range attr.FromSources {
				fromSources[i] = model.K8sAttributesFrom(source)
			}
		}
		result = append(result, &model.K8sLabelAttribute{
			LabelKey:     attr.LabelKey,
			AttributeKey: attr.AttributeKey,
			From:         from,
			FromSources:  fromSources,
		})
	}

	return result
}

func convertAnnotationsAttributesToModel(annotationsAttributes []actionsv1.K8sAnnotationAttribute) []*model.K8sAnnotationAttribute {
	var result []*model.K8sAnnotationAttribute

	for _, attr := range annotationsAttributes {
		var from *model.K8sAttributesFrom
		if attr.From != nil {
			tmp := model.K8sAttributesFrom(*attr.From)
			from = &tmp
		}
		var fromSources []model.K8sAttributesFrom
		if len(attr.FromSources) > 0 {
			fromSources = make([]model.K8sAttributesFrom, len(attr.FromSources))
			for i, source := range attr.FromSources {
				fromSources[i] = model.K8sAttributesFrom(source)
			}
		}
		result = append(result, &model.K8sAnnotationAttribute{
			AnnotationKey: attr.AnnotationKey,
			AttributeKey:  attr.AttributeKey,
			From:          from,
			FromSources:   fromSources,
		})
	}

	return result
}

func convertClusterAttributesToModel(clusterAttributes []actionsv1.OtelAttributeWithValue) []*model.ClusterAttribute {
	var result []*model.ClusterAttribute

	for _, attr := range clusterAttributes {
		var stringValue string
		if attr.AttributeStringValue != nil {
			stringValue = *attr.AttributeStringValue
		}
		result = append(result, &model.ClusterAttribute{
			AttributeName:        attr.AttributeName,
			AttributeStringValue: stringValue,
		})
	}

	return result
}

func convertPiiCategoriesToModel(piiCategories []actionsv1.PiiCategory) []string {
	var result []string

	for _, category := range piiCategories {
		result = append(result, string(category))
	}

	return result
}

func convertEndpointsFiltersToModel(endpointsFilters []actionsv1.HttpRouteFilter) []*model.HTTPRouteFilter {
	var result []*model.HTTPRouteFilter

	for _, f := range endpointsFilters {
		result = append(result, &model.HTTPRouteFilter{
			HTTPRoute:               f.HttpRoute,
			ServiceName:             f.ServiceName,
			MinimumLatencyThreshold: f.MinimumLatencyThreshold,
			FallbackSamplingRatio:   f.FallbackSamplingRatio,
		})
	}

	return result
}

func convertServiceNameFiltersToModel(serviceNameFilters []actionsv1.ServiceNameFilter) []*model.ServiceNameFilter {
	var result []*model.ServiceNameFilter

	for _, f := range serviceNameFilters {
		result = append(result, &model.ServiceNameFilter{
			ServiceName:           f.ServiceName,
			SamplingRatio:         f.SamplingRatio,
			FallbackSamplingRatio: f.FallbackSamplingRatio,
		})
	}

	return result
}

func convertAttributeFiltersToModel(attributeFilters []actionsv1.SpanAttributeFilter) []*model.SpanAttributeFilter {
	var result []*model.SpanAttributeFilter

	for _, f := range attributeFilters {
		cond := &model.AttributeFiltersCondition{}

		if f.Condition.StringCondition != nil {
			cond.StringCondition = &model.StringCondition{
				Operation:     model.StringOperation(f.Condition.StringCondition.Operation),
				ExpectedValue: &f.Condition.StringCondition.ExpectedValue,
			}
		}

		if f.Condition.NumberCondition != nil {
			cond.NumberCondition = &model.NumberCondition{
				Operation:     model.NumberOperation(f.Condition.NumberCondition.Operation),
				ExpectedValue: f.Condition.NumberCondition.ExpectedValue,
			}
		}

		if f.Condition.BooleanCondition != nil {
			cond.BooleanCondition = &model.BooleanCondition{
				Operation:     model.BooleanOperation(f.Condition.BooleanCondition.Operation),
				ExpectedValue: f.Condition.BooleanCondition.ExpectedValue,
			}
		}

		if f.Condition.JsonCondition != nil {
			cond.JSONCondition = &model.JSONCondition{
				Operation:     model.JSONOperation(f.Condition.JsonCondition.Operation),
				ExpectedValue: &f.Condition.JsonCondition.ExpectedValue,
				JSONPath:      &f.Condition.JsonCondition.JsonPath,
			}
		}

		result = append(result, &model.SpanAttributeFilter{
			ServiceName:           f.ServiceName,
			AttributeKey:          f.AttributeKey,
			FallbackSamplingRatio: f.FallbackSamplingRatio,
			Condition:             cond,
		})
	}

	return result
}

func stringifyMap(m map[string]string) (string, error) {
	json, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to marshal map: %v", err)
	}
	return string(json), nil
}
