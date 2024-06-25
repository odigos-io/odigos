package sampling

import (
	"context"
	"reflect"

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
	List(ctx context.Context, client client.Client, namespace string) ([]metav1.Object, error)
	// ValidatePolicyConfig validates the policy configuration for the action
	ValidatePolicyConfig(Policy) error
	// GetActionReference returns the owner reference for the action
	GetActionReference(action metav1.Object) metav1.OwnerReference
	// IsActionDisabled returns whether the action is disabled [action.Spec.Disabled]
	IsActionDisabled(action metav1.Object) bool
	// SelectSampler returns the selected action in case of conflict.
	SelectSampler(actions []metav1.Object) (metav1.Object, error)
}
