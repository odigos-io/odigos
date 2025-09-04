package sampling

import (
	"context"
	"errors"
	"fmt"
	"strings"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LatencySamplerHandler struct{}

type LatencyConfig struct {
	ThresholdMs           int     `json:"threshold"`
	HttpRoute             string  `json:"http_route"`
	ServiceName           string  `json:"service_name"`
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

func (h *LatencySamplerHandler) ConvertLegacyToAction(legacyAction metav1.Object) metav1.Object {
	// If the action is already an odigos action, return it
	if _, ok := legacyAction.(*odigosv1.Action); ok {
		return legacyAction
	}
	legacyLatencySampler := legacyAction.(*actionv1.LatencySampler)
	action := &odigosv1.Action{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Action",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      legacyLatencySampler.Name,
			Namespace: legacyLatencySampler.Namespace,
		},
		Spec: odigosv1.ActionSpec{
			ActionName: legacyLatencySampler.Spec.ActionName,
			Notes:      legacyLatencySampler.Spec.Notes,
			Disabled:   legacyLatencySampler.Spec.Disabled,
			Signals:    legacyLatencySampler.Spec.Signals,
			Samplers: &actionv1.SamplersConfig{
				LatencySampler: &actionv1.LatencySamplerConfig{
					EndpointsFilters: legacyLatencySampler.Spec.EndpointsFilters,
				},
			},
		},
	}
	return action
}

func (h *LatencySamplerHandler) List(ctx context.Context, c client.Client, namespace string) ([]metav1.Object, error) {
	var legacyList actionv1.LatencySamplerList
	if err := c.List(ctx, &legacyList, client.InNamespace(namespace)); err != nil && client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	// Handle the migration from legacy latencysampler to odigos action, convert legacy latencysampler to odigos action
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
		if item.Spec.Samplers.LatencySampler != nil {
			items = append(items, &list.Items[i])
		}
	}
	items = append(items, legacyItems...)
	return items, nil
}

func (h *LatencySamplerHandler) IsActionDisabled(action metav1.Object) bool {
	// Handle migration from legacy latencysampler to odigos action
	if a, ok := action.(*actionv1.LatencySampler); ok {
		return a.Spec.Disabled
	}
	return action.(*odigosv1.Action).Spec.Disabled
}

func (h *LatencySamplerHandler) ValidateRuleConfig(config []Rule) error {
	for _, rule := range config {
		if err := rule.Details.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (h *LatencySamplerHandler) GetRuleConfig(action metav1.Object) []Rule {
	var latencysampler *odigosv1.Action
	// Handle migration from legacy latencysampler to odigos action
	if a, ok := action.(*actionv1.LatencySampler); ok {
		latencysampler = h.ConvertLegacyToAction(a).(*odigosv1.Action)
	}
	if a, ok := action.(*odigosv1.Action); ok {
		latencysampler = a
	}

	actionRules := []Rule{}

	for _, config := range latencysampler.Spec.Samplers.LatencySampler.EndpointsFilters {
		latencyDetails := &LatencyConfig{
			ThresholdMs:           config.MinimumLatencyThreshold,
			HttpRoute:             config.HttpRoute,
			ServiceName:           config.ServiceName,
			FallbackSamplingRatio: config.FallbackSamplingRatio,
		}

		actionRules = append(actionRules, Rule{
			Name:     fmt.Sprintf("latency-%s-%s", latencyDetails.ServiceName, latencyDetails.HttpRoute),
			RuleType: LatencyRule,
			Details:  latencyDetails,
		})
	}

	return actionRules

}

func (h *LatencySamplerHandler) GetActionReference(action metav1.Object) metav1.OwnerReference {
	// Handle migration from legacy latencysampler to odigos action
	if a, ok := action.(*actionv1.LatencySampler); ok {
		return metav1.OwnerReference{APIVersion: a.APIVersion, Kind: a.Kind, Name: a.Name, UID: a.UID}
	}
	if a, ok := action.(*odigosv1.Action); ok {
		return metav1.OwnerReference{APIVersion: a.APIVersion, Kind: a.Kind, Name: a.Name, UID: a.UID}
	}
	return metav1.OwnerReference{}
}

func (h *LatencySamplerHandler) GetActionScope(action metav1.Object) string {
	return "endpoint"
}

func (lc *LatencyConfig) Validate() error {
	if lc.ThresholdMs <= 0 {
		return errors.New("minimum latency threshold must be positive")
	}
	if lc.FallbackSamplingRatio < 0 || lc.FallbackSamplingRatio > 100 {
		return errors.New("fallback_sampling_ratio must be between 0 and 100")
	}
	if lc.HttpRoute == "" {
		return errors.New("http_route cannot be empty")
	}
	if !strings.HasPrefix(lc.HttpRoute, "/") {
		return errors.New("http_route must start with /")
	}
	if lc.ServiceName == "" {
		return errors.New("service_name cannot be empty")
	}
	return nil
}
