package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ErrorSamplerDetails struct {
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

// CreateErrorSampler creates a new ErrorSampler action in Kubernetes
func CreateErrorSampler(ctx context.Context, action model.ActionInput) (model.Action, error) {
	odigosns := consts.DefaultOdigosNamespace

	var details ErrorSamplerDetails
	err := json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for ErrorSampler: %v", err)
	}

	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	errorSamplerAction := &v1alpha1.ErrorSampler{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "es-",
		},
		Spec: v1alpha1.ErrorSamplerSpec{
			ActionName:            services.DerefString(action.Name),
			Notes:                 services.DerefString(action.Notes),
			Disabled:              action.Disable,
			Signals:               signals,
			FallbackSamplingRatio: details.FallbackSamplingRatio,
		},
	}

	generatedAction, err := kube.DefaultClient.ActionsClient.ErrorSamplers(odigosns).Create(ctx, errorSamplerAction, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create ErrorSampler: %v", err)
	}
	detailsString := strconv.FormatFloat(details.FallbackSamplingRatio, 'f', -1, 64)
	response := &model.ErrorSamplerAction{
		ID:      generatedAction.Name,
		Type:    ActionTypeErrorSampler,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: detailsString,
	}

	return response, nil
}

// UpdateErrorSampler updates an existing ErrorSampler action in Kubernetes
func UpdateErrorSampler(ctx context.Context, id string, action model.ActionInput) (model.Action, error) {
	odigosns := consts.DefaultOdigosNamespace

	existingAction, err := kube.DefaultClient.ActionsClient.ErrorSamplers(odigosns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ErrorSampler: %v", err)
	}

	var details ErrorSamplerDetails
	err = json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for ErrorSampler: %v", err)
	}

	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	existingAction.Spec.ActionName = services.DerefString(action.Name)
	existingAction.Spec.Notes = services.DerefString(action.Notes)
	existingAction.Spec.Disabled = action.Disable
	existingAction.Spec.Signals = signals
	existingAction.Spec.FallbackSamplingRatio = details.FallbackSamplingRatio

	updatedAction, err := kube.DefaultClient.ActionsClient.ErrorSamplers(odigosns).Update(ctx, existingAction, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update ErrorSampler: %v", err)
	}
	detailsString := strconv.FormatFloat(details.FallbackSamplingRatio, 'f', -1, 64)
	response := &model.ErrorSamplerAction{
		ID:      updatedAction.Name,
		Type:    ActionTypeErrorSampler,
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Details: detailsString,
	}

	return response, nil
}

// DeleteErrorSampler deletes an existing ErrorSampler action from Kubernetes
func DeleteErrorSampler(ctx context.Context, id string) error {
	odigosns := consts.DefaultOdigosNamespace

	err := kube.DefaultClient.ActionsClient.ErrorSamplers(odigosns).Delete(ctx, id, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("ErrorSampler action with ID %s not found", id)
		}
		return fmt.Errorf("failed to delete ErrorSampler action: %v", err)
	}

	return nil
}
