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

type ServiceNameSamplerDetails struct {
	ServiceNameFilters []v1alpha1.ServiceNameFilter `json:"services_name_filters"`
}

func CreateServiceNameSampler(ctx context.Context, action model.ActionInput) (model.Action, error) {
	var details ServiceNameSamplerDetails
	err := json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for ServiceNameSampler: %v", err)
	}

	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	serviceNameSamplerAction := &v1alpha1.ServiceNameSampler{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "sns-",
		},
		Spec: v1alpha1.ServiceNameSamplerSpec{
			ActionName:          services.DerefString(action.Name),
			Notes:               services.DerefString(action.Notes),
			Disabled:            action.Disable,
			Signals:             signals,
			ServicesNameFilters: details.ServiceNameFilters,
		},
	}

	ns := env.GetCurrentNamespace()
	generatedAction, err := kube.DefaultClient.ActionsClient.ServiceNameSamplers(ns).Create(ctx, serviceNameSamplerAction, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create ServiceNameSampler: %v", err)
	}

	response := &model.ServiceNameSamplerAction{
		ID:      generatedAction.Name,
		Type:    ActionTypeServiceNameSampler,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: convertServiceNameFiltersForResponse(generatedAction.Spec.ServicesNameFilters),
	}

	return response, nil
}

func UpdateServiceNameSampler(ctx context.Context, id string, action model.ActionInput) (model.Action, error) {
	ns := env.GetCurrentNamespace()

	existingAction, err := kube.DefaultClient.ActionsClient.ServiceNameSamplers(ns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ServiceNameSampler: %v", err)
	}

	var details ServiceNameSamplerDetails
	err = json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for ServiceNameSampler: %v", err)
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
	existingAction.Spec.ServicesNameFilters = details.ServiceNameFilters

	updatedAction, err := kube.DefaultClient.ActionsClient.ServiceNameSamplers(ns).Update(ctx, existingAction, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update ServiceNameSampler: %v", err)
	}

	response := &model.ServiceNameSamplerAction{
		ID:      updatedAction.Name,
		Type:    ActionTypeServiceNameSampler,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: convertServiceNameFiltersForResponse(updatedAction.Spec.ServicesNameFilters),
	}

	return response, nil
}

func DeleteServiceNameSampler(ctx context.Context, id string) error {
	ns := env.GetCurrentNamespace()

	err := kube.DefaultClient.ActionsClient.ServiceNameSamplers(ns).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("ServiceNameSampler action with ID %s not found", id)
		}
		return fmt.Errorf("failed to delete ServiceNameSampler action: %v", err)
	}

	return nil
}

func convertServiceNameFiltersForResponse(serviceNameFilters []v1alpha1.ServiceNameFilter) []*model.ServiceNameFilters {
	var result []*model.ServiceNameFilters

	for _, f := range serviceNameFilters {
		result = append(result, &model.ServiceNameFilters{
			ServiceName:           f.ServiceName,
			SamplingRatio:         f.SamplingRatio,
			FallbackSamplingRatio: f.FallbackSamplingRatio,
		})
	}

	return result
}
