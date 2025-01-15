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

type RenameAttributeDetails struct {
	Renames map[string]string `json:"renames"`
}

// CreateRenameAttribute creates a new RenameAttribute action in Kubernetes
func CreateRenameAttribute(ctx context.Context, action model.ActionInput) (model.Action, error) {
	var details RenameAttributeDetails
	err := json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for RenameAttribute: %v", err)
	}

	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	renameAttributeAction := &v1alpha1.RenameAttribute{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "ra-",
		},
		Spec: v1alpha1.RenameAttributeSpec{
			ActionName: services.DerefString(action.Name),
			Notes:      services.DerefString(action.Notes),
			Disabled:   action.Disable,
			Signals:    signals,
			Renames:    details.Renames,
		},
	}

	ns := env.GetCurrentNamespace()

	generatedAction, err := kube.DefaultClient.ActionsClient.RenameAttributes(ns).Create(ctx, renameAttributeAction, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create RenameAttribute: %v", err)
	}

	// Convert Renames to JSON string for response details
	detailsJSON, err := json.Marshal(details.Renames)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal renames: %v", err)
	}
	responseDetails := string(detailsJSON)

	response := &model.RenameAttributeAction{
		ID:      generatedAction.Name,
		Type:    ActionTypeRenameAttribute,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: responseDetails,
	}

	return response, nil
}

// UpdateRenameAttribute updates an existing RenameAttribute action in Kubernetes
func UpdateRenameAttribute(ctx context.Context, id string, action model.ActionInput) (model.Action, error) {
	ns := env.GetCurrentNamespace()

	existingAction, err := kube.DefaultClient.ActionsClient.RenameAttributes(ns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RenameAttribute: %v", err)
	}

	var details RenameAttributeDetails
	err = json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for RenameAttribute: %v", err)
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
	existingAction.Spec.Renames = details.Renames

	updatedAction, err := kube.DefaultClient.ActionsClient.RenameAttributes(ns).Update(ctx, existingAction, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update RenameAttribute: %v", err)
	}

	// Convert Renames to JSON string for response details
	detailsJSON, err := json.Marshal(details.Renames)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal renames: %v", err)
	}
	responseDetails := string(detailsJSON)

	response := &model.RenameAttributeAction{
		ID:      updatedAction.Name,
		Type:    ActionTypeRenameAttribute,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: responseDetails,
	}

	return response, nil
}

func DeleteRenameAttribute(ctx context.Context, id string) error {
	ns := env.GetCurrentNamespace()

	err := kube.DefaultClient.ActionsClient.RenameAttributes(ns).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("RenameAttribute action with ID %s not found", id)
		}
		return fmt.Errorf("failed to delete RenameAttribute action: %v", err)
	}

	return nil
}
