package sampling

import (
	"context"
	"errors"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ErrorSamplerHandler struct{}

type ErrorConfig struct {
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

func (h *ErrorSamplerHandler) ConvertLegacyToAction(legacyAction metav1.Object) metav1.Object {
	// If the action is already an odigos action, return it
	if _, ok := legacyAction.(*odigosv1.Action); ok {
		return legacyAction
	}
	legacyErrorSampler := legacyAction.(*actionv1.ErrorSampler)
	action := &odigosv1.Action{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Action",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      legacyErrorSampler.Name,
			Namespace: legacyErrorSampler.Namespace,
		},
		Spec: odigosv1.ActionSpec{
			ActionName: legacyErrorSampler.Spec.ActionName,
			Notes:      legacyErrorSampler.Spec.Notes,
			Disabled:   legacyErrorSampler.Spec.Disabled,
			Signals:    legacyErrorSampler.Spec.Signals,
			Samplers: &actionv1.SamplersConfig{
				ErrorSampler: &actionv1.ErrorSamplerConfig{
					FallbackSamplingRatio: legacyErrorSampler.Spec.FallbackSamplingRatio,
				},
			},
		},
	}
	return action
}

func (h *ErrorSamplerHandler) List(ctx context.Context, c client.Client, namespace string) ([]metav1.Object, error) {
	var list odigosv1.ActionList
	if err := c.List(ctx, &list, client.InNamespace(namespace)); err != nil && client.IgnoreNotFound(err) != nil {
		return nil, err
	}
	items := make([]metav1.Object, 0)
	for i, item := range list.Items {
		if item.Spec.Samplers != nil && item.Spec.Samplers.ErrorSampler != nil {
			items = append(items, &list.Items[i])
		}
	}
	return items, nil
}

func (h *ErrorSamplerHandler) IsActionDisabled(action metav1.Object) bool {
	// Handle migration from legacy errorsampler to odigos action
	if a, ok := action.(*actionv1.ErrorSampler); ok {
		return a.Spec.Disabled
	}
	return action.(*odigosv1.Action).Spec.Disabled
}

func (h *ErrorSamplerHandler) ValidateRuleConfig(config []Rule) error {
	for _, rule := range config {
		if err := rule.Details.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (h *ErrorSamplerHandler) GetRuleConfig(action metav1.Object) []Rule {
	var errorSampler *odigosv1.Action
	// Handle migration from legacy errorsampler to odigos action
	if a, ok := action.(*actionv1.ErrorSampler); ok {
		errorSampler = h.ConvertLegacyToAction(a).(*odigosv1.Action)
	}
	if a, ok := action.(*odigosv1.Action); ok {
		errorSampler = a
	}

	if errorSampler.Spec.Samplers == nil || errorSampler.Spec.Samplers.ErrorSampler == nil {
		return []Rule{}
	}

	errorDetails := &ErrorConfig{
		FallbackSamplingRatio: errorSampler.Spec.Samplers.ErrorSampler.FallbackSamplingRatio,
	}

	return []Rule{
		{
			Name:     "error-rule",
			RuleType: ErrorRule,
			Details:  errorDetails,
		},
	}
}

func (h *ErrorSamplerHandler) GetActionReference(action metav1.Object) metav1.OwnerReference {
	// Handle migration from legacy errorsampler to odigos action
	if a, ok := action.(*actionv1.ErrorSampler); ok {
		return metav1.OwnerReference{APIVersion: a.APIVersion, Kind: a.Kind, Name: a.Name, UID: a.UID}
	}
	if a, ok := action.(*odigosv1.Action); ok {
		return metav1.OwnerReference{APIVersion: a.APIVersion, Kind: a.Kind, Name: a.Name, UID: a.UID}
	}
	return metav1.OwnerReference{}
}

func (h *ErrorSamplerHandler) GetActionScope(action metav1.Object) string {
	return "global"
}

func (ec *ErrorConfig) Validate() error {
	if ec.FallbackSamplingRatio < 0 || ec.FallbackSamplingRatio > 100 {
		return errors.New("fallback_sampling_ratio must be between 0 and 100")
	}
	return nil
}
