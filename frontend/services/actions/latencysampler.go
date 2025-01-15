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

type LatencySamplerDetails struct {
	EndpointsFilters []v1alpha1.HttpRouteFilter `json:"endpoints_filters"`
}

// CreateLatencySampler creates a new LatencySampler action in Kubernetes
func CreateLatencySampler(ctx context.Context, action model.ActionInput) (model.Action, error) {
	var details LatencySamplerDetails
	err := json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for LatencySampler: %v", err)
	}

	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	latencySamplerAction := &v1alpha1.LatencySampler{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "ls-",
		},
		Spec: v1alpha1.LatencySamplerSpec{
			ActionName:       services.DerefString(action.Name),
			Notes:            services.DerefString(action.Notes),
			Disabled:         action.Disable,
			Signals:          signals,
			EndpointsFilters: details.EndpointsFilters,
		},
	}

	ns := env.GetCurrentNamespace()

	generatedAction, err := kube.DefaultClient.ActionsClient.LatencySamplers(ns).Create(ctx, latencySamplerAction, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create LatencySampler: %v", err)
	}

	// Convert Endpoint Filters to JSON string for response details
	detailsJSON, err := json.Marshal(details.EndpointsFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal endpoint filters: %v", err)
	}
	responseDetails := []*string{services.StringPtr(string(detailsJSON))}

	response := &model.LatencySamplerAction{
		ID:      generatedAction.Name,
		Type:    ActionTypeLatencySampler,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: responseDetails,
	}

	return response, nil
}

// UpdateLatencySampler updates an existing LatencySampler action in Kubernetes
func UpdateLatencySampler(ctx context.Context, id string, action model.ActionInput) (model.Action, error) {
	ns := env.GetCurrentNamespace()

	existingAction, err := kube.DefaultClient.ActionsClient.LatencySamplers(ns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch LatencySampler: %v", err)
	}

	var details LatencySamplerDetails
	err = json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for LatencySampler: %v", err)
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
	existingAction.Spec.EndpointsFilters = details.EndpointsFilters

	updatedAction, err := kube.DefaultClient.ActionsClient.LatencySamplers(ns).Update(ctx, existingAction, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update LatencySampler: %v", err)
	}

	// Convert Endpoint Filters to JSON string for response details
	detailsJSON, err := json.Marshal(details.EndpointsFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal endpoint filters: %v", err)
	}
	responseDetails := []*string{services.StringPtr(string(detailsJSON))}

	response := &model.LatencySamplerAction{
		ID:      updatedAction.Name,
		Type:    ActionTypeLatencySampler,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: responseDetails,
	}

	return response, nil
}

// DeleteLatencySampler deletes an existing LatencySampler action from Kubernetes
func DeleteLatencySampler(ctx context.Context, id string) error {
	ns := env.GetCurrentNamespace()

	err := kube.DefaultClient.ActionsClient.LatencySamplers(ns).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("LatencySampler action with ID %s not found", id)
		}
		return fmt.Errorf("failed to delete LatencySampler action: %v", err)
	}

	return nil
}
