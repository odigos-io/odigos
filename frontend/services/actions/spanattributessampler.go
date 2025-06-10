package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Operation string

const (
	Exists             Operation = "exists"
	Equals             Operation = "equals"
	NotEquals          Operation = "not_equals"
	Contains           Operation = "contains"
	NotContains        Operation = "not_contains"
	Regex              Operation = "regex"
	GreaterThan        Operation = "greater_than"
	LessThan           Operation = "less_than"
	GreaterThanOrEqual Operation = "greater_than_or_equal"
	LessThanOrEqual    Operation = "less_than_or_equal"
	IsValidJson        Operation = "is_valid_json"
	IsInvalidJson      Operation = "is_invalid_json"
	JsonPathExists     Operation = "jsonpath_exists"
	KeyEquals          Operation = "key_equals"
	KeyNotEquals       Operation = "key_not_equals"
)

type SpanAttributeSamplerDetails struct {
	AttributeFilters []v1alpha1.SpanAttributeFilter `json:"attribute_filters"`
}

func CreateSpanAttributeSampler(ctx context.Context, action model.ActionInput) (model.Action, error) {
	var details SpanAttributeSamplerDetails
	err := json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for SpanAttributeSampler: %v", err)
	}

	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	serviceNameSamplerAction := &v1alpha1.SpanAttributeSampler{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "sas-",
		},
		Spec: v1alpha1.SpanAttributeSamplerSpec{
			ActionName:       services.DerefString(action.Name),
			Notes:            services.DerefString(action.Notes),
			Disabled:         action.Disable,
			Signals:          signals,
			AttributeFilters: details.AttributeFilters,
		},
	}

	ns := env.GetCurrentNamespace()
	generatedAction, err := kube.DefaultClient.ActionsClient.SpanAttributeSamplers(ns).Create(ctx, serviceNameSamplerAction, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create SpanAttributeSampler: %v", err)
	}

	response := &model.SpanAttributeSamplerAction{
		ID:      generatedAction.Name,
		Type:    ActionTypeSpanAttributeSampler,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: convertAttributeFiltersForResponse(generatedAction.Spec.AttributeFilters),
	}

	return response, nil
}

func UpdateSpanAttributeSampler(ctx context.Context, id string, action model.ActionInput) (model.Action, error) {
	ns := env.GetCurrentNamespace()

	existingAction, err := kube.DefaultClient.ActionsClient.SpanAttributeSamplers(ns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch SpanAttributeSampler: %v", err)
	}

	var details SpanAttributeSamplerDetails
	err = json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for SpanAttributeSampler: %v", err)
	}

	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	// Update the existing action with new values
	existingAction.Spec.ActionName = services.DerefString(action.Name)
	existingAction.Spec.Notes = services.DerefString(action.Notes)
	existingAction.Spec.Disabled = action.Disable
	existingAction.Spec.Signals = signals
	existingAction.Spec.AttributeFilters = details.AttributeFilters

	updatedAction, err := kube.DefaultClient.ActionsClient.SpanAttributeSamplers(ns).Update(ctx, existingAction, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update SpanAttributeSampler: %v", err)
	}

	response := &model.SpanAttributeSamplerAction{
		ID:      updatedAction.Name,
		Type:    ActionTypeSpanAttributeSampler,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: convertAttributeFiltersForResponse(updatedAction.Spec.AttributeFilters),
	}

	return response, nil
}

func DeleteSpanAttributeSampler(ctx context.Context, id string) error {
	ns := env.GetCurrentNamespace()

	err := kube.DefaultClient.ActionsClient.SpanAttributeSamplers(ns).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("SpanAttributeSampler action with ID %s not found", id)
		}
		return fmt.Errorf("failed to delete SpanAttributeSampler action: %v", err)
	}

	return nil
}

func convertAttributeFiltersForResponse(attributeFilters []v1alpha1.SpanAttributeFilter) []*model.AttributeFilters {
	var result []*model.AttributeFilters

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

		result = append(result, &model.AttributeFilters{
			ServiceName:           f.ServiceName,
			AttributeKey:          f.AttributeKey,
			FallbackSamplingRatio: f.FallbackSamplingRatio,
			Condition:             cond,
		})
	}

	return result
}
