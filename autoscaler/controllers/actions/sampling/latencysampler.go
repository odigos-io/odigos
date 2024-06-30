package sampling

import (
	"context"
	"errors"
	"fmt"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LatencySamplerHandler struct{}

type LatencyConfig struct {
	ThresholdMs int    `json:"threshold"`
	Endpoint    string `json:"endpoint"`
	Service     string `json:"service"`
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

func (h *LatencySamplerHandler) ValidateRuleConfig(config Rule) error {
	return config.Details.Validate()
}

func (h *LatencySamplerHandler) GetRuleConfig(action metav1.Object) Rule {
	a := action.(*actionv1.LatencySampler)
	latencyDetails := &LatencyConfig{
		ThresholdMs: a.Spec.MinimumLatencyThreshold,
		Endpoint:    a.Spec.Endpoint,
		Service:     a.Spec.Service,
	}

	return Rule{
		Name:     fmt.Sprintf("latency-%s-%s", latencyDetails.Service, latencyDetails.Endpoint),
		RuleType: "http_latency",
		Details:  latencyDetails,
	}
}

func (h *LatencySamplerHandler) GetActionReference(action metav1.Object) metav1.OwnerReference {
	a := action.(*actionv1.LatencySampler)
	return metav1.OwnerReference{APIVersion: a.APIVersion, Kind: a.Kind, Name: a.Name, UID: a.UID}
}

func (lc *LatencyConfig) Validate() error {
	if lc.ThresholdMs < 0 {
		return errors.New("minimum latency threshold must be positive")
	}

	return nil
}
