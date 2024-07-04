package sampling

import (
	"context"
	"reflect"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var SamplingSupportedActions = map[reflect.Type]ActionHandler{
	reflect.TypeOf(&actionv1.LatencySampler{}): &LatencySamplerHandler{},
	reflect.TypeOf(&actionv1.ErrorSampler{}):   &ErrorSamplerHandler{},
	// Add more action types here
}

// ActionHandler defines methods for handling sampling actions using metav1.Objects for generic handling
type ActionHandler interface {
	// GetRuleConfig Returns rule configuration for a given action
	GetRuleConfig(action metav1.Object) []Rule
	// // Lists all actions of a specific type in a given namespace
	List(ctx context.Context, client client.Client, namespace string) ([]metav1.Object, error)
	// ValidateRuleConfig validates the rule configuration for the action
	ValidateRuleConfig([]Rule) error
	// GetActionReference returns the owner reference for the action
	GetActionReference(action metav1.Object) metav1.OwnerReference
	// IsActionDisabled returns whether the action is disabled [action.Spec.Disabled]
	IsActionDisabled(action metav1.Object) bool
	// GetActionScope returns the scope of the action [global/service/endpoint]
	GetActionScope(action metav1.Object) string
}
