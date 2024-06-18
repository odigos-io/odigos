package sampling

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var TailSamplingSupportedActions = map[reflect.Type]ActionHandler{
	reflect.TypeOf(&actionv1.LatencySampler{}):       &LatencySamplerHandler{},
	reflect.TypeOf(&actionv1.ProbabilisticSampler{}): &ProbabilisticSamplerHandler{},
	// Add more action types here
}

// ActionHandler defines methods for handling tail sampling actions using metav1.Objects for generic handling
type ActionHandler interface {
	// GetPolicyConfig Returns policy configuration for a given action
	GetPolicyConfig(action metav1.Object) Policy
	// // Lists all actions of a specific type in a given namespace
	List(client client.Client, namespace string) ([]metav1.Object, error)
	// ValidatePolicyConfig validates the policy configuration for the action
	ValidatePolicyConfig(Policy) error
	// GetActionReference returns the owner reference for the action
	GetActionReference(action metav1.Object) metav1.OwnerReference
	// IsActionDisabled returns whether the action is disabled [action.Spec.Disabled]
	IsActionDisabled(action metav1.Object) bool
	// SelectSampler returns the selected action in case of conflict.
	SelectSampler(actions []metav1.Object) (metav1.Object, error)
}

type LatencySamplerHandler struct{}

func (h *LatencySamplerHandler) List(c client.Client, namespace string) ([]metav1.Object, error) {
	var list actionv1.LatencySamplerList
	if err := c.List(context.TODO(), &list, client.InNamespace(namespace)); err != nil && client.IgnoreNotFound(err) != nil {
		return nil, err
	}
	items := make([]metav1.Object, len(list.Items))
	for i, item := range list.Items {
		items[i] = item.DeepCopyObject().(metav1.Object)
	}
	return items, nil
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
		MinimumLatencyThreshold: a.Spec.MinimumLatencyThreshold,
	}

	if a.Spec.MaximumLatencyThreshold != nil {
		latencyDetails.MaximumLatencyThreshold = a.Spec.MaximumLatencyThreshold
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

type ProbabilisticSamplerHandler struct{}

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

func (h *ProbabilisticSamplerHandler) List(c client.Client, namespace string) ([]metav1.Object, error) {
	var list actionv1.ProbabilisticSamplerList
	if err := c.List(context.TODO(), &list, client.InNamespace(namespace)); err != nil && client.IgnoreNotFound(err) != nil {
		return nil, err
	}
	items := make([]metav1.Object, len(list.Items))
	for i, item := range list.Items {
		items[i] = item.DeepCopyObject().(metav1.Object)
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
