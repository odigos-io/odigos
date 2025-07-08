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

type K8sAttributesDetails struct {
	CollectContainerAttributes  bool                              `json:"collectContainerAttributes"`
	CollectReplicaSetAttributes bool                              `json:"collectReplicaSetAttributes"`
	CollectWorkloadUID          bool                              `json:"collectWorkloadId"`
	CollectClusterUID           bool                              `json:"collectClusterId"`
	LabelsAttributes            []v1alpha1.K8sLabelAttribute      `json:"labelsAttributes,omitempty"`
	AnnotationsAttributes       []v1alpha1.K8sAnnotationAttribute `json:"annotationsAttributes,omitempty"`
}

func CreateK8sAttributes(ctx context.Context, action model.ActionInput) (model.Action, error) {
	var details K8sAttributesDetails
	err := json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for K8sAttributes: %v", err)
	}

	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	k8sAttributesAction := &v1alpha1.K8sAttributesResolver{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "ka-",
		},
		Spec: v1alpha1.K8sAttributesSpec{
			ActionName:                  services.DerefString(action.Name),
			Notes:                       services.DerefString(action.Notes),
			Disabled:                    action.Disable,
			Signals:                     signals,
			CollectContainerAttributes:  details.CollectContainerAttributes,
			CollectReplicaSetAttributes: details.CollectReplicaSetAttributes,
			CollectWorkloadUID:          details.CollectWorkloadUID,
			CollectClusterUID:           details.CollectClusterUID,
			LabelsAttributes:            details.LabelsAttributes,
			AnnotationsAttributes:       details.AnnotationsAttributes,
		},
	}

	ns := env.GetCurrentNamespace()

	generatedAction, err := kube.DefaultClient.ActionsClient.K8sAttributesResolvers(ns).Create(ctx, k8sAttributesAction, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create K8sAttributes: %v", err)
	}

	labelAttrs := make([]*model.K8sLabelAttribute, len(details.LabelsAttributes))
	for i, attr := range details.LabelsAttributes {
		labelAttrs[i] = &model.K8sLabelAttribute{
			LabelKey:     attr.LabelKey,
			AttributeKey: attr.AttributeKey,
		}
	}

	annotAttrs := make([]*model.K8sAnnotationAttribute, len(details.AnnotationsAttributes))
	for i, attr := range details.AnnotationsAttributes {
		annotAttrs[i] = &model.K8sAnnotationAttribute{
			AnnotationKey: attr.AnnotationKey,
			AttributeKey:  attr.AttributeKey,
		}
	}

	response := &model.K8sAttributesAction{
		ID:      generatedAction.Name,
		Type:    ActionTypeK8sAttributes,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: &model.K8sAttributes{
			CollectContainerAttributes: details.CollectContainerAttributes,
			CollectWorkloadID:          details.CollectWorkloadUID,
			CollectClusterID:           details.CollectClusterUID,
			LabelsAttributes:           labelAttrs,
			AnnotationsAttributes:      annotAttrs,
		},
	}

	return response, nil
}

func UpdateK8sAttributes(ctx context.Context, id string, action model.ActionInput) (model.Action, error) {
	ns := env.GetCurrentNamespace()

	// Fetch the existing action
	existingAction, err := kube.DefaultClient.ActionsClient.K8sAttributesResolvers(ns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch K8sAttributes: %v", err)
	}

	// Parse the details from action.Details
	var details K8sAttributesDetails
	err = json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for K8sAttributes: %v", err)
	}

	// Convert signals from action input
	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	// Update the existing action with new values
	existingAction.Spec.ActionName = services.DerefString(action.Name)
	existingAction.Spec.Notes = services.DerefString(action.Notes)
	existingAction.Spec.Disabled = action.Disable
	existingAction.Spec.Signals = signals
	existingAction.Spec.CollectContainerAttributes = details.CollectContainerAttributes
	existingAction.Spec.CollectReplicaSetAttributes = details.CollectReplicaSetAttributes
	existingAction.Spec.CollectWorkloadUID = details.CollectWorkloadUID
	existingAction.Spec.CollectClusterUID = details.CollectClusterUID
	existingAction.Spec.LabelsAttributes = details.LabelsAttributes
	existingAction.Spec.AnnotationsAttributes = details.AnnotationsAttributes

	// Update the action in Kubernetes
	updatedAction, err := kube.DefaultClient.ActionsClient.K8sAttributesResolvers(ns).Update(ctx, existingAction, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update K8sAttributes: %v", err)
	}

	// Prepare the response model
	labelAttrs := make([]*model.K8sLabelAttribute, len(details.LabelsAttributes))
	for i, attr := range details.LabelsAttributes {
		labelAttrs[i] = &model.K8sLabelAttribute{
			LabelKey:     attr.LabelKey,
			AttributeKey: attr.AttributeKey,
		}
	}

	annotAttrs := make([]*model.K8sAnnotationAttribute, len(details.AnnotationsAttributes))
	for i, attr := range details.AnnotationsAttributes {
		annotAttrs[i] = &model.K8sAnnotationAttribute{
			AnnotationKey: attr.AnnotationKey,
			AttributeKey:  attr.AttributeKey,
		}
	}

	response := &model.K8sAttributesAction{
		ID:      updatedAction.Name,
		Type:    ActionTypeK8sAttributes,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: &model.K8sAttributes{
			CollectContainerAttributes: details.CollectContainerAttributes,
			CollectWorkloadID:          details.CollectWorkloadUID,
			CollectClusterID:           details.CollectClusterUID,
			LabelsAttributes:           labelAttrs,
			AnnotationsAttributes:      annotAttrs,
		},
	}

	return response, nil
}

func DeleteK8sAttributes(ctx context.Context, id string) error {
	ns := env.GetCurrentNamespace()

	// Delete the action by its ID from Kubernetes
	err := kube.DefaultClient.ActionsClient.K8sAttributesResolvers(ns).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("K8sAttributes action with ID %s not found", id)
		}
		return fmt.Errorf("failed to delete K8sAttributes action: %v", err)
	}

	return nil
}
