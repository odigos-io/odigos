package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type TopologySpreadConstraint struct {
	MaxSkew           int                   `json:"maxSkew,omitempty"`
	TopologyKey       string                `json:"topologyKey,omitempty"`
	WhenUnsatisfiable string                `json:"whenUnsatisfiable,omitempty"`
	LabelSelector     *metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// +kubebuilder:object:generate=true
type TopologySpread struct {
	Enabled     bool                       `json:"enabled,omitempty"`
	Constraints []TopologySpreadConstraint `json:"constraints,omitempty"`
}
