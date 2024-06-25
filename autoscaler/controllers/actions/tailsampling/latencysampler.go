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
	ThresholdMs      int  `json:"threshold_ms"`
	UpperThresholdMs *int `json:"upper_threshold_ms,omitempty"`
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
func (h *LatencySamplerHandler) SelectSampler(actions []metav1.Object) (metav1.Object, error) {
	if len(actions) == 0 {
		return nil, fmt.Errorf("no actions provided")
	}

	selected := actions[0].(*actionv1.LatencySampler)
	for _, action := range actions {
		latencySampler := action.(*actionv1.LatencySampler)
		if latencySampler.Spec.MinimumLatencyThreshold < selected.Spec.MinimumLatencyThreshold {
			selected = latencySampler
		}
	}
	return selected, nil
}

func (h *LatencySamplerHandler) IsActionDisabled(action metav1.Object) bool {
	return action.(*actionv1.LatencySampler).Spec.Disabled
}

func (h *LatencySamplerHandler) ValidatePolicyConfig(config Policy) error {
	return config.Details.Validate()
}

func (h *LatencySamplerHandler) GetPolicyConfig(action metav1.Object) Policy {
	a := action.(*actionv1.LatencySampler)
	latencyDetails := &LatencyConfig{
		ThresholdMs: a.Spec.MinimumLatencyThreshold,
	}

	if a.Spec.MaximumLatencyThreshold != nil {
		latencyDetails.UpperThresholdMs = a.Spec.MaximumLatencyThreshold
	}

	return Policy{
		Name:       "latency_policy",
		PolicyType: "latency",
		Details:    latencyDetails,
	}
}

func (h *LatencySamplerHandler) GetActionReference(action metav1.Object) metav1.OwnerReference {
	a := action.(*actionv1.LatencySampler)
	return metav1.OwnerReference{APIVersion: a.APIVersion, Kind: a.Kind, Name: a.Name, UID: a.UID}
}

func (lc *LatencyConfig) Validate() error {
	if lc.UpperThresholdMs != nil {
		if *lc.UpperThresholdMs < 0 {
			return errors.New("upper latency threshold must be positive")
		}
		if *lc.UpperThresholdMs <= lc.ThresholdMs {
			return errors.New("upper latency threshold must be greater than minimum latency threshold")
		}

	}

	if lc.ThresholdMs < 0 {
		return errors.New("minimum latency threshold must be positive")
	}

	return nil
}
