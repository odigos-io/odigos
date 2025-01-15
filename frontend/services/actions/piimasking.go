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

type PiiMaskingDetails struct {
	PiiCategories []v1alpha1.PiiCategory `json:"piiCategories"`
}

// CreatePiiMasking creates a new PiiMasking action in Kubernetes
func CreatePiiMasking(ctx context.Context, action model.ActionInput) (model.Action, error) {
	ns := env.GetCurrentNamespace()

	var details PiiMaskingDetails
	err := json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for PiiMasking: %v", err)
	}

	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	piiMaskingAction := &v1alpha1.PiiMasking{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "pi-",
		},
		Spec: v1alpha1.PiiMaskingSpec{
			ActionName:    services.DerefString(action.Name),
			Notes:         services.DerefString(action.Notes),
			Disabled:      action.Disable,
			Signals:       signals,
			PiiCategories: details.PiiCategories,
		},
	}

	generatedAction, err := kube.DefaultClient.ActionsClient.PiiMaskings(ns).Create(ctx, piiMaskingAction, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create PiiMasking: %v", err)
	}

	piiCategories := make([]string, len(details.PiiCategories))
	for i, category := range details.PiiCategories {
		piiCategories[i] = string(category)
	}

	response := &model.PiiMaskingAction{
		ID:      generatedAction.Name,
		Type:    ActionTypePiiMasking,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: piiCategories,
	}

	return response, nil
}

// UpdatePiiMasking updates an existing PiiMasking action in Kubernetes
func UpdatePiiMasking(ctx context.Context, id string, action model.ActionInput) (model.Action, error) {
	ns := env.GetCurrentNamespace()

	existingAction, err := kube.DefaultClient.ActionsClient.PiiMaskings(ns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PiiMasking: %v", err)
	}

	var details PiiMaskingDetails
	err = json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for PiiMasking: %v", err)
	}

	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	existingAction.Spec.ActionName = services.DerefString(action.Name)
	existingAction.Spec.Notes = services.DerefString(action.Notes)
	existingAction.Spec.Disabled = action.Disable
	existingAction.Spec.Signals = signals
	existingAction.Spec.PiiCategories = details.PiiCategories

	updatedAction, err := kube.DefaultClient.ActionsClient.PiiMaskings(ns).Update(ctx, existingAction, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update PiiMasking: %v", err)
	}

	piiCategories := make([]string, len(details.PiiCategories))
	for i, category := range details.PiiCategories {
		piiCategories[i] = string(category)
	}

	response := &model.PiiMaskingAction{
		ID:      updatedAction.Name,
		Type:    ActionTypePiiMasking,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: piiCategories,
	}

	return response, nil
}

// DeletePiiMasking deletes an existing PiiMasking action from Kubernetes
func DeletePiiMasking(ctx context.Context, id string) error {
	ns := env.GetCurrentNamespace()

	err := kube.DefaultClient.ActionsClient.PiiMaskings(ns).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("PiiMasking action with ID %s not found", id)
		}
		return fmt.Errorf("failed to delete PiiMasking action: %v", err)
	}

	return nil
}
