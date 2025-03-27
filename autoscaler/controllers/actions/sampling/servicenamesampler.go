package sampling

import (
	"context"
	"errors"
	"fmt"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceNameSamplerHandler struct{}

type ServiceNameConfig struct {
	ServiceName           string  `json:"service_name"`
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

func (h *ServiceNameSamplerHandler) List(ctx context.Context, c client.Client, namespace string) ([]metav1.Object, error) {
	var list actionv1.ServiceNameSamplerList
	if err := c.List(ctx, &list, client.InNamespace(namespace)); err != nil && client.IgnoreNotFound(err) != nil {
		return nil, err
	}
	items := make([]metav1.Object, len(list.Items))
	for i := range list.Items {
		items[i] = &list.Items[i]
	}
	return items, nil
}

func (h *ServiceNameSamplerHandler) IsActionDisabled(action metav1.Object) bool {
	return action.(*actionv1.ServiceNameSampler).Spec.Disabled
}

func (h *ServiceNameSamplerHandler) ValidateRuleConfig(config []Rule) error {
	for _, rule := range config {
		if err := rule.Details.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (h *ServiceNameSamplerHandler) GetRuleConfig(action metav1.Object) []Rule {
	svcAction := action.(*actionv1.ServiceNameSampler)
	rules := make([]Rule, 0, len(svcAction.Spec.Services))

	for _, service := range svcAction.Spec.Services {
		rules = append(rules, Rule{
			Name:     fmt.Sprintf("service-%s", service.ServiceName),
			RuleType: ServiceNameRule,
			Details: &ServiceNameConfig{
				ServiceName:           service.ServiceName,
				FallbackSamplingRatio: service.FallbackSamplingRatio,
			},
		})
	}
	return rules
}

func (h *ServiceNameSamplerHandler) GetActionReference(action metav1.Object) metav1.OwnerReference {
	a := action.(*actionv1.ServiceNameSampler)
	return metav1.OwnerReference{APIVersion: a.APIVersion, Kind: a.Kind, Name: a.Name, UID: a.UID}
}

func (h *ServiceNameSamplerHandler) GetActionScope(action metav1.Object) string {
	return "global" // or "service" if you want it more scoped
}

func (cfg *ServiceNameConfig) Validate() error {
	if cfg.ServiceName == "" {
		return errors.New("service_name cannot be empty")
	}
	if cfg.FallbackSamplingRatio < 0 || cfg.FallbackSamplingRatio > 100 {
		return errors.New("fallback_sampling_ratio must be between 0 and 100")
	}
	return nil
}
