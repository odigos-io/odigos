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

type ProbabilisticSamplerDetails struct {
	SamplingPercentage string `json:"sampling_percentage"`
}

// CreateProbabilisticSampler creates a new ProbabilisticSampler action in Kubernetes
func CreateProbabilisticSampler(ctx context.Context, action model.ActionInput) (model.Action, error) {
	var details ProbabilisticSamplerDetails
	err := json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for ProbabilisticSampler: %v", err)
	}

	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	probabilisticSamplerAction := &v1alpha1.ProbabilisticSampler{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "ps-",
		},
		Spec: v1alpha1.ProbabilisticSamplerSpec{
			ActionName:         services.DerefString(action.Name),
			Notes:              services.DerefString(action.Notes),
			Disabled:           action.Disable,
			Signals:            signals,
			SamplingPercentage: details.SamplingPercentage,
		},
	}

	ns := env.GetCurrentNamespace()

	generatedAction, err := kube.DefaultClient.ActionsClient.ProbabilisticSamplers(ns).Create(ctx, probabilisticSamplerAction, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create ProbabilisticSampler: %v", err)
	}

	// Convert SamplingPercentage to JSON string and assign it as the first element of []*string
	detailsJSON, err := json.Marshal(details.SamplingPercentage)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sampling percentage: %v", err)
	}
	responseDetails := string(detailsJSON)

	response := &model.ProbabilisticSamplerAction{
		ID:      generatedAction.Name,
		Type:    ActionTypeProbabilisticSampler,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: responseDetails,
	}

	return response, nil
}

// UpdateProbabilisticSampler updates an existing ProbabilisticSampler action in Kubernetes
func UpdateProbabilisticSampler(ctx context.Context, id string, action model.ActionInput) (model.Action, error) {
	ns := env.GetCurrentNamespace()

	existingAction, err := kube.DefaultClient.ActionsClient.ProbabilisticSamplers(ns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ProbabilisticSampler: %v", err)
	}

	var details ProbabilisticSamplerDetails
	err = json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for ProbabilisticSampler: %v", err)
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
	existingAction.Spec.SamplingPercentage = details.SamplingPercentage

	updatedAction, err := kube.DefaultClient.ActionsClient.ProbabilisticSamplers(ns).Update(ctx, existingAction, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update ProbabilisticSampler: %v", err)
	}

	// Convert SamplingPercentage to JSON string and assign it as the first element of []*string
	detailsJSON, err := json.Marshal(details.SamplingPercentage)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sampling percentage: %v", err)
	}
	responseDetails := string(detailsJSON)

	response := &model.ProbabilisticSamplerAction{
		ID:      updatedAction.Name,
		Type:    ActionTypeProbabilisticSampler,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: responseDetails,
	}

	return response, nil
}

// DeleteProbabilisticSampler deletes an existing ProbabilisticSampler action from Kubernetes
func DeleteProbabilisticSampler(ctx context.Context, id string) error {
	ns := env.GetCurrentNamespace()

	err := kube.DefaultClient.ActionsClient.ProbabilisticSamplers(ns).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("ProbabilisticSampler action with ID %s not found", id)
		}
		return fmt.Errorf("failed to delete ProbabilisticSampler action: %v", err)
	}

	return nil
}
