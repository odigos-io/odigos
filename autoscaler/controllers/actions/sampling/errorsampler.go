package sampling

import (
	"context"
	"errors"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ErrorSamplerHandler struct{}

type ErrorConfig struct {
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

func (h *ErrorSamplerHandler) List(ctx context.Context, c client.Client, namespace string) ([]metav1.Object, error) {
	var list actionv1.ErrorSamplerList
	if err := c.List(ctx, &list, client.InNamespace(namespace)); err != nil && client.IgnoreNotFound(err) != nil {
		return nil, err
	}
	items := make([]metav1.Object, len(list.Items))
	for i, item := range list.Items {

		items[i] = &item
	}
	return items, nil
}

func (h *ErrorSamplerHandler) IsActionDisabled(action metav1.Object) bool {
	return action.(*actionv1.ErrorSampler).Spec.Disabled
}

func (h *ErrorSamplerHandler) ValidateRuleConfig(config []Rule) error {
	return config[0].Details.Validate()
}

func (h *ErrorSamplerHandler) GetRuleConfig(action metav1.Object) []Rule {
	a := action.(*actionv1.ErrorSampler)
	errorDetails := &ErrorConfig{
		FallbackSamplingRatio: a.Spec.FallbackSamplingRatio,
	}

	return []Rule{
		{
			Name:     "error-rule",
			RuleType: "error",
			Details:  errorDetails,
		},
	}
}

func (h *ErrorSamplerHandler) GetActionReference(action metav1.Object) metav1.OwnerReference {
	a := action.(*actionv1.ErrorSampler)
	return metav1.OwnerReference{APIVersion: a.APIVersion, Kind: a.Kind, Name: a.Name, UID: a.UID}
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
