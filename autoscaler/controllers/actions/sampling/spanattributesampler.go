package sampling

import (
	"context"
	"errors"
	"fmt"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SpanAttributeSamplerHandler struct{}

type SpanAttributeConfig struct {
	AttributeKey          string  `json:"attribute_key"`
	Condition             string  `json:"condition"`
	ExpectedValue         string  `json:"expected_value,omitempty"`
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

func (h *SpanAttributeSamplerHandler) List(ctx context.Context, c client.Client, namespace string) ([]metav1.Object, error) {
	var list actionv1.SpanAttributeSamplerList
	if err := c.List(ctx, &list, client.InNamespace(namespace)); err != nil && client.IgnoreNotFound(err) != nil {
		return nil, err
	}
	items := make([]metav1.Object, len(list.Items))
	for i := range list.Items {
		items[i] = &list.Items[i]
	}
	return items, nil
}

func (h *SpanAttributeSamplerHandler) IsActionDisabled(action metav1.Object) bool {
	return action.(*actionv1.SpanAttributeSampler).Spec.Disabled
}

func (h *SpanAttributeSamplerHandler) ValidateRuleConfig(config []Rule) error {
	for _, rule := range config {
		if err := rule.Details.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (h *SpanAttributeSamplerHandler) GetRuleConfig(action metav1.Object) []Rule {
	attrAction := action.(*actionv1.SpanAttributeSampler)
	rules := make([]Rule, 0, len(attrAction.Spec.AttributesFilters))

	for _, cfg := range attrAction.Spec.AttributesFilters {
		rules = append(rules, Rule{
			Name:     fmt.Sprintf("attribute-%s", cfg.AttributeKey),
			RuleType: SpanAttributeRule,
			Details: &SpanAttributeConfig{
				AttributeKey:          cfg.AttributeKey,
				Condition:             cfg.Condition,
				ExpectedValue:         cfg.ExpectedValue,
				FallbackSamplingRatio: cfg.FallbackSamplingRatio,
			},
		})
	}
	return rules
}

func (h *SpanAttributeSamplerHandler) GetActionReference(action metav1.Object) metav1.OwnerReference {
	a := action.(*actionv1.SpanAttributeSampler)
	return metav1.OwnerReference{APIVersion: a.APIVersion, Kind: a.Kind, Name: a.Name, UID: a.UID}
}

func (h *SpanAttributeSamplerHandler) GetActionScope(action metav1.Object) string {
	return "global" // or "service"/"endpoint" if applicable
}

func (cfg *SpanAttributeConfig) Validate() error {
	if cfg.AttributeKey == "" {
		return errors.New("attribute_key cannot be empty")
	}
	if cfg.Condition != "exists" && cfg.Condition != "equals" && cfg.Condition != "not_equals" {
		return errors.New("condition must be one of: exists, equals, not_equals")
	}
	if (cfg.Condition == "equals" || cfg.Condition == "not_equals") && cfg.ExpectedValue == "" {
		return errors.New("expected_value is required for 'equals' and 'not_equals'")
	}
	if cfg.FallbackSamplingRatio < 0 || cfg.FallbackSamplingRatio > 100 {
		return errors.New("fallback_sampling_ratio must be between 0 and 100")
	}
	return nil
}
