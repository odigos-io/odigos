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
	ServiceName           string                   `json:"service_name"`
	AttributeKey          string                   `json:"attribute_key"`
	FallbackSamplingRatio float64                  `json:"fallback_sampling_ratio"`
	Condition             AttributeConditionConfig `json:"condition"`
}

type AttributeConditionConfig struct {
	Type          string `json:"type"`
	Operation     string `json:"operation"`
	ExpectedValue string `json:"expected_value,omitempty"`
	JsonPath      string `json:"json_path,omitempty"`
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
	rules := make([]Rule, 0, len(attrAction.Spec.AttributeFilters))

	for _, filter := range attrAction.Spec.AttributeFilters {
		cond := AttributeConditionConfig{}

		switch {
		case filter.Condition.StringCondition != nil:
			cond.Type = "string"
			cond.Operation = filter.Condition.StringCondition.Operation
			cond.ExpectedValue = filter.Condition.StringCondition.ExpectedValue

		case filter.Condition.NumberCondition != nil:
			cond.Type = "number"
			cond.Operation = filter.Condition.NumberCondition.Operation
			cond.ExpectedValue = fmt.Sprintf("%v", filter.Condition.NumberCondition.ExpectedValue)

		case filter.Condition.BooleanCondition != nil:
			cond.Type = "boolean"
			cond.Operation = filter.Condition.BooleanCondition.Operation
			cond.ExpectedValue = fmt.Sprintf("%v", filter.Condition.BooleanCondition.ExpectedValue)

		case filter.Condition.JsonCondition != nil:
			cond.Type = "json"
			cond.Operation = filter.Condition.JsonCondition.Operation
			cond.ExpectedValue = filter.Condition.JsonCondition.ExpectedValue
			cond.JsonPath = filter.Condition.JsonCondition.JsonPath
		}

		ruleName := fmt.Sprintf("%s-%s-%s", filter.ServiceName, cond.Type, filter.AttributeKey)

		rules = append(rules, Rule{
			Name:     ruleName,
			RuleType: SpanAttributeRule,
			Details: &SpanAttributeConfig{
				ServiceName:           filter.ServiceName,
				AttributeKey:          filter.AttributeKey,
				FallbackSamplingRatio: filter.FallbackSamplingRatio,
				Condition:             cond,
			},
		})
	}
	return rules
}

func (h *SpanAttributeSamplerHandler) GetActionReference(action metav1.Object) metav1.OwnerReference {
	a := action.(*actionv1.SpanAttributeSampler)
	return metav1.OwnerReference{
		APIVersion: a.APIVersion,
		Kind:       a.Kind,
		Name:       a.Name,
		UID:        a.UID,
	}
}

func (h *SpanAttributeSamplerHandler) GetActionScope(action metav1.Object) string {
	return "service"
}

func (cfg *SpanAttributeConfig) Validate() error {
	if cfg.ServiceName == "" {
		return errors.New("service_name cannot be empty")
	}

	if cfg.AttributeKey == "" {
		return errors.New("attribute_key cannot be empty")
	}

	if cfg.FallbackSamplingRatio < 0 || cfg.FallbackSamplingRatio > 100 {
		return errors.New("fallback_sampling_ratio must be between 0 and 100")
	}

	switch cfg.Condition.Type {
	case "string":
		validOps := map[string]bool{
			"exists": true, "equals": true, "not_equals": true,
			"contains": true, "not_contains": true, "regex": true,
		}
		if !validOps[cfg.Condition.Operation] {
			return fmt.Errorf("invalid operation '%s' for string condition", cfg.Condition.Operation)
		}
		if cfg.Condition.Operation != "exists" && cfg.Condition.ExpectedValue == "" {
			return errors.New("expected_value required for string condition except 'exists'")
		}

	case "number":
		validOps := map[string]bool{
			"exists": true, "equals": true, "not_equals": true,
			"greater_than": true, "less_than": true,
			"greater_than_or_equal": true, "less_than_or_equal": true,
		}
		if !validOps[cfg.Condition.Operation] {
			return fmt.Errorf("invalid operation '%s' for number condition", cfg.Condition.Operation)
		}
		if cfg.Condition.Operation != "exists" && cfg.Condition.ExpectedValue == "" {
			return errors.New("expected_value required for number condition except 'exists'")
		}

	case "boolean":
		validOps := map[string]bool{"exists": true, "equals": true}
		if !validOps[cfg.Condition.Operation] {
			return fmt.Errorf("invalid operation '%s' for boolean condition", cfg.Condition.Operation)
		}
		if cfg.Condition.Operation == "equals" && cfg.Condition.ExpectedValue == "" {
			return errors.New("expected_value required for boolean equals operation")
		}

	case "json":
		validOps := map[string]bool{
			"exists": true, "is_valid_json": true, "is_invalid_json": true,
			"equals": true, "not_equals": true,
			"contains_key": true, "not_contains_key": true,
			"jsonpath_exists": true,
		}
		if !validOps[cfg.Condition.Operation] {
			return fmt.Errorf("invalid operation '%s' for json condition", cfg.Condition.Operation)
		}

		switch cfg.Condition.Operation {
		case "equals", "not_equals", "contains_key", "not_contains_key":
			if cfg.Condition.ExpectedValue == "" {
				return fmt.Errorf("expected_value required for '%s' operation", cfg.Condition.Operation)
			}
		case "jsonpath_exists":
			if cfg.Condition.JsonPath == "" {
				return errors.New("json_path required for 'jsonpath_exists' operation")
			}
		}

	default:
		return fmt.Errorf("unsupported attribute condition type: %s", cfg.Condition.Type)
	}

	return nil
}
