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

type DeleteAttributeDetails struct {
	AttributeNamesToDelete []string `json:"attributeNamesToDelete"`
}

// CreateDeleteAttribute creates a new DeleteAttribute action in Kubernetes
func CreateDeleteAttribute(ctx context.Context, action model.ActionInput) (model.Action, error) {
	var details DeleteAttributeDetails
	err := json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for DeleteAttribute: %v", err)
	}

	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	deleteAttributeAction := &v1alpha1.DeleteAttribute{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "da-",
		},
		Spec: v1alpha1.DeleteAttributeSpec{
			ActionName:             services.DerefString(action.Name),
			Notes:                  services.DerefString(action.Notes),
			Disabled:               action.Disable,
			Signals:                signals,
			AttributeNamesToDelete: details.AttributeNamesToDelete,
		},
	}

	ns := env.GetCurrentNamespace()

	generatedAction, err := kube.DefaultClient.ActionsClient.DeleteAttributes(ns).Create(ctx, deleteAttributeAction, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create DeleteAttribute: %v", err)
	}

	response := &model.DeleteAttributeAction{
		ID:      generatedAction.Name,
		Type:    ActionTypeDeleteAttribute,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: details.AttributeNamesToDelete,
	}

	return response, nil
}

// UpdateDeleteAttribute updates an existing DeleteAttribute action in Kubernetes
func UpdateDeleteAttribute(ctx context.Context, id string, action model.ActionInput) (model.Action, error) {
	ns := env.GetCurrentNamespace()

	existingAction, err := kube.DefaultClient.ActionsClient.DeleteAttributes(ns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch DeleteAttribute: %v", err)
	}

	var details DeleteAttributeDetails
	err = json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for DeleteAttribute: %v", err)
	}

	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	existingAction.Spec.ActionName = services.DerefString(action.Name)
	existingAction.Spec.Notes = services.DerefString(action.Notes)
	existingAction.Spec.Disabled = action.Disable
	existingAction.Spec.Signals = signals
	existingAction.Spec.AttributeNamesToDelete = details.AttributeNamesToDelete

	updatedAction, err := kube.DefaultClient.ActionsClient.DeleteAttributes(ns).Update(ctx, existingAction, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update DeleteAttribute: %v", err)
	}

	response := &model.DeleteAttributeAction{
		ID:      updatedAction.Name,
		Type:    ActionTypeDeleteAttribute,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: details.AttributeNamesToDelete,
	}

	return response, nil
}

// DeleteDeleteAttribute deletes an existing DeleteAttribute action from Kubernetes
func DeleteDeleteAttribute(ctx context.Context, id string) error {
	ns := env.GetCurrentNamespace()

	err := kube.DefaultClient.ActionsClient.DeleteAttributes(ns).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("DeleteAttribute action with ID %s not found", id)
		}
		return fmt.Errorf("failed to delete DeleteAttribute action: %v", err)
	}

	return nil
}
