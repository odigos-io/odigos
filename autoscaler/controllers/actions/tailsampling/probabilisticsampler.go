package sampling

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ProbabilisticSamplerHandler struct{}

type ProbabilisticConfig struct {
	Value float64 `json:"sampling_percentage"`
}

func (h *ProbabilisticSamplerHandler) IsActionDisabled(action metav1.Object) bool {
	return action.(*actionv1.ProbabilisticSampler).Spec.Disabled
}

func (h *ProbabilisticSamplerHandler) ValidatePolicyConfig(config Policy) error {
	return config.Details.Validate()
}

func (h *ProbabilisticSamplerHandler) GetPolicyConfig(action metav1.Object) Policy {
	a := action.(*actionv1.ProbabilisticSampler)
	samplingPercentage, err := strconv.ParseFloat(a.Spec.SamplingPercentage, 32)
	if err != nil {
		return Policy{}
	}

	probabilisticDetails := &ProbabilisticConfig{
		Value: samplingPercentage,
	}

	return Policy{
		Name:       "probabilistic_policy",
		PolicyType: "probabilistic",
		Details:    probabilisticDetails,
	}
}

func (h *ProbabilisticSamplerHandler) GetActionReference(action metav1.Object) metav1.OwnerReference {
	a := action.(*actionv1.ProbabilisticSampler)
	return metav1.OwnerReference{APIVersion: a.APIVersion, Kind: a.Kind, Name: a.Name, UID: a.UID}
}

func (h *ProbabilisticSamplerHandler) List(ctx context.Context, c client.Client, namespace string) ([]metav1.Object, error) {
	var list actionv1.ProbabilisticSamplerList
	if err := c.List(ctx, &list, client.InNamespace(namespace)); err != nil && client.IgnoreNotFound(err) != nil {
		return nil, err
	}
	items := make([]metav1.Object, len(list.Items))
	for i, item := range list.Items {
		items[i] = &item
	}
	return items, nil
}

func (h *ProbabilisticSamplerHandler) SelectSampler(actions []metav1.Object) (metav1.Object, error) {
	// Specific selection logic for ProbabilisticSampler in case of conflict
	// Choosing the one with the lowest sampling percentage
	if len(actions) == 0 {
		return nil, fmt.Errorf("no actions provided")
	}

	selected := actions[0].(*actionv1.ProbabilisticSampler)
	for _, action := range actions {
		probabilisticSampler := action.(*actionv1.ProbabilisticSampler)
		currentPercentage, err := strconv.ParseFloat(probabilisticSampler.Spec.SamplingPercentage, 64)
		if err != nil {
			return nil, err
		}
		selectedPercentage, err := strconv.ParseFloat(selected.Spec.SamplingPercentage, 64)
		if err != nil {
			return nil, err
		}
		if currentPercentage < selectedPercentage {
			selected = probabilisticSampler
		}
	}
	return selected, nil
}

func (pc *ProbabilisticConfig) Validate() error {
	if pc.Value < 0 {
		return errors.New("sampling_percentage cannot be negative")
	}
	return nil
}
