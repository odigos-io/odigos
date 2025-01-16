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

type AddClusterInfoDetails struct {
	ClusterAttributes []model.ClusterInfo `json:"clusterAttributes"`
}

func CreateAddClusterInfo(ctx context.Context, action model.ActionInput) (model.Action, error) {
	var details AddClusterInfoDetails
	err := json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for AddClusterInfo: %v", err)
	}

	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	clusterAttributes := make([]v1alpha1.OtelAttributeWithValue, len(details.ClusterAttributes))
	for i, attr := range details.ClusterAttributes {
		clusterAttributes[i] = v1alpha1.OtelAttributeWithValue{
			AttributeName:        attr.AttributeName,
			AttributeStringValue: attr.AttributeStringValue,
		}
	}

	addClusterInfoAction := &v1alpha1.AddClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "aci-",
		},
		Spec: v1alpha1.AddClusterInfoSpec{
			ActionName:        services.DerefString(action.Name),
			Notes:             services.DerefString(action.Notes),
			Disabled:          action.Disable,
			Signals:           signals,
			ClusterAttributes: clusterAttributes,
		},
	}

	ns := env.GetCurrentNamespace()

	generatedAction, err := kube.DefaultClient.ActionsClient.AddClusterInfos(ns).Create(ctx, addClusterInfoAction, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create AddClusterInfo: %v", err)
	}

	resDetails := make([]*model.ClusterInfo, len(details.ClusterAttributes))
	for i, attr := range details.ClusterAttributes {
		resDetails[i] = &model.ClusterInfo{
			AttributeName:        attr.AttributeName,
			AttributeStringValue: attr.AttributeStringValue,
		}
	}

	response := &model.AddClusterInfoAction{
		ID:      generatedAction.Name,
		Type:    ActionTypeAddClusterInfo,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: resDetails,
	}

	return response, nil
}

func UpdateAddClusterInfo(ctx context.Context, id string, action model.ActionInput) (model.Action, error) {
	ns := env.GetCurrentNamespace()

	// Fetch the existing action
	existingAction, err := kube.DefaultClient.ActionsClient.AddClusterInfos(ns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch AddClusterInfo: %v", err)
	}

	// Parse the details from action.Details
	var details AddClusterInfoDetails
	err = json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for AddClusterInfo: %v", err)
	}

	// Convert signals from action input
	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	// Convert ClusterAttributes from model to v1alpha1
	clusterAttributes := make([]v1alpha1.OtelAttributeWithValue, len(details.ClusterAttributes))
	for i, attr := range details.ClusterAttributes {
		clusterAttributes[i] = v1alpha1.OtelAttributeWithValue{
			AttributeName:        attr.AttributeName,
			AttributeStringValue: attr.AttributeStringValue,
		}
	}

	// Update the existing action with new values
	existingAction.Spec.ActionName = services.DerefString(action.Name)
	existingAction.Spec.Notes = services.DerefString(action.Notes)
	existingAction.Spec.Disabled = action.Disable
	existingAction.Spec.Signals = signals
	existingAction.Spec.ClusterAttributes = clusterAttributes

	// Update the action in Kubernetes
	updatedAction, err := kube.DefaultClient.ActionsClient.AddClusterInfos(ns).Update(ctx, existingAction, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update AddClusterInfo: %v", err)
	}

	// Prepare the response model
	resDetails := make([]*model.ClusterInfo, len(details.ClusterAttributes))
	for i, attr := range details.ClusterAttributes {
		resDetails[i] = &model.ClusterInfo{
			AttributeName:        attr.AttributeName,
			AttributeStringValue: attr.AttributeStringValue,
		}
	}

	// Return the updated model as the response
	response := &model.AddClusterInfoAction{
		ID:      updatedAction.Name,
		Type:    ActionTypeAddClusterInfo,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: resDetails,
	}

	return response, nil
}

func DeleteAddClusterInfo(ctx context.Context, id string) error {
	ns := env.GetCurrentNamespace()

	// Delete the action by its ID from Kubernetes
	err := kube.DefaultClient.ActionsClient.AddClusterInfos(ns).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("AddClusterInfo action with ID %s not found", id)
		}
		return fmt.Errorf("failed to delete AddClusterInfo action: %v", err)
	}

	return nil
}
