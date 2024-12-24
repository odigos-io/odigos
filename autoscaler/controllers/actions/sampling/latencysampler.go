package sampling

import (
	"context"
	"errors"
	"fmt"
	"strings"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
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

func (h *LatencySamplerHandler) List(ctx context.Context, c client.Client, namespace string) ([]metav1.Object, error) {
	var list actionv1.LatencySamplerList
	if err := c.List(ctx, &list, client.InNamespace(namespace)); err != nil && client.IgnoreNotFound(err) != nil {
		return nil, err
	}
	items := make([]metav1.Object, len(list.Items))
	for i, item := range list.Items {

		items[i] = &item
	}
	return items, nil
}

func (h *LatencySamplerHandler) IsActionDisabled(action metav1.Object) bool {
	return action.(*actionv1.LatencySampler).Spec.Disabled
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
	latencysampler := action.(*actionv1.LatencySampler)
	actionRules := []Rule{}

	for _, config := range latencysampler.Spec.EndpointsFilters {
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
	a := action.(*actionv1.LatencySampler)
	return metav1.OwnerReference{APIVersion: a.APIVersion, Kind: a.Kind, Name: a.Name, UID: a.UID}
}

func (h *LatencySamplerHandler) GetActionScope(action metav1.Object) string {
	return "endpoint"
}

func (lc *LatencyConfig) Validate() error {
	if lc.ThresholdMs < 0 {
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
