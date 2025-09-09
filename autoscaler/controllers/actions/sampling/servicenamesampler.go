package sampling

import (
	"context"
	"errors"
	"fmt"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceNameSamplerHandler struct{}

type ServiceNameConfig struct {
	ServiceName           string  `json:"service_name"`
	SamplingRatio         float64 `json:"sampling_ratio"`
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

func (h *ServiceNameSamplerHandler) ConvertLegacyToAction(legacyAction metav1.Object) metav1.Object {
	// If the action is already an odigos action, return it
	if _, ok := legacyAction.(*odigosv1.Action); ok {
		return legacyAction
	}
	legacyServiceNameSampler := legacyAction.(*actionv1.ServiceNameSampler)
	action := &odigosv1.Action{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Action",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      legacyServiceNameSampler.Name,
			Namespace: legacyServiceNameSampler.Namespace,
		},
		Spec: odigosv1.ActionSpec{
			ActionName: legacyServiceNameSampler.Spec.ActionName,
			Notes:      legacyServiceNameSampler.Spec.Notes,
			Disabled:   legacyServiceNameSampler.Spec.Disabled,
			Signals:    legacyServiceNameSampler.Spec.Signals,
			Samplers: &actionv1.SamplersConfig{
				ServiceNameSampler: &actionv1.ServiceNameSamplerConfig{
					ServicesNameFilters: legacyServiceNameSampler.Spec.ServicesNameFilters,
				},
			},
		},
	}
	return action
}

func (h *ServiceNameSamplerHandler) List(ctx context.Context, c client.Client, namespace string) ([]metav1.Object, error) {
	var legacyList actionv1.ServiceNameSamplerList
	if err := c.List(ctx, &legacyList, client.InNamespace(namespace)); err != nil && client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	// Handle the migration from legacy servicenamesampler to odigos action, convert legacy servicenamesampler to odigos action
	// and add the new odigos action to the list
	legacyItems := make([]metav1.Object, len(legacyList.Items))
	for i, item := range legacyList.Items {
		legacyItems[i] = &item
	}

	var list odigosv1.ActionList
	if err := c.List(ctx, &list, client.InNamespace(namespace)); err != nil && client.IgnoreNotFound(err) != nil {
		return nil, err
	}
	items := make([]metav1.Object, 0)
	for i, item := range list.Items {
		if item.Spec.Samplers.ServiceNameSampler != nil {
			items = append(items, &list.Items[i])
		}
	}
	items = append(items, legacyItems...)
	return items, nil
}

func (h *ServiceNameSamplerHandler) IsActionDisabled(action metav1.Object) bool {
	// Handle migration from legacy servicenamesampler to odigos action
	if a, ok := action.(*actionv1.ServiceNameSampler); ok {
		return a.Spec.Disabled
	}
	return action.(*odigosv1.Action).Spec.Disabled
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
	var serviceNameSampler *odigosv1.Action
	// Handle migration from legacy servicenamesampler to odigos action
	if a, ok := action.(*actionv1.ServiceNameSampler); ok {
		serviceNameSampler = h.ConvertLegacyToAction(a).(*odigosv1.Action)
	}
	if a, ok := action.(*odigosv1.Action); ok {
		serviceNameSampler = a
	}

	rules := make([]Rule, 0, len(serviceNameSampler.Spec.Samplers.ServiceNameSampler.ServicesNameFilters))

	for _, service := range serviceNameSampler.Spec.Samplers.ServiceNameSampler.ServicesNameFilters {
		rules = append(rules, Rule{
			Name:     fmt.Sprintf("service-%s", service.ServiceName),
			RuleType: ServiceNameRule,
			Details: &ServiceNameConfig{
				ServiceName:           service.ServiceName,
				SamplingRatio:         service.SamplingRatio,
				FallbackSamplingRatio: service.FallbackSamplingRatio,
			},
		})
	}
	return rules
}

func (h *ServiceNameSamplerHandler) GetActionReference(action metav1.Object) metav1.OwnerReference {
	// Handle migration from legacy servicenamesampler to odigos action
	if a, ok := action.(*actionv1.ServiceNameSampler); ok {
		return metav1.OwnerReference{
			APIVersion: a.APIVersion,
			Kind:       a.Kind,
			Name:       a.Name,
			UID:        a.UID,
		}
	}
	if a, ok := action.(*odigosv1.Action); ok {
		return metav1.OwnerReference{
			APIVersion: a.APIVersion,
			Kind:       a.Kind,
			Name:       a.Name,
			UID:        a.UID,
		}
	}
	return metav1.OwnerReference{}
}

func (h *ServiceNameSamplerHandler) GetActionScope(action metav1.Object) string {
	return "service"
}

func (cfg *ServiceNameConfig) Validate() error {
	if cfg.ServiceName == "" {
		return errors.New("service_name cannot be empty")
	}
	if cfg.SamplingRatio < 0 || cfg.SamplingRatio > 100 {
		return errors.New("sampling_ratio must be between 0 and 100")
	}
	if cfg.FallbackSamplingRatio < 0 || cfg.FallbackSamplingRatio > 100 {
		return errors.New("fallback_sampling_ratio must be between 0 and 100")
	}
	return nil
}
