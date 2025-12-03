package nodedetails

import (
	"context"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

// Feature is an interface for checking node features.
// Enterprise implementations can provide additional features that will be checked
// alongside the OSS features to gather extra node information.
type Feature interface {
	// Check examines the node and updates the NodeDetailsSpec with feature information.
	// It receives the Kubernetes Node object and the current spec, and should modify the spec
	// with the detected feature state.
	Check(ctx context.Context, node *v1.Node, spec *v1alpha1.NodeDetailsSpec) error

	// Name returns a human-readable name for this feature, used for logging.
	Name() string
}

var (
	// features holds all registered features (OSS + enterprise extensions)
	features []Feature
)

// RegisterFeature adds a feature to the registry.
// This allows enterprise code to register additional features that will be checked
// during the node initialization phase.
func RegisterFeature(feature Feature) {
	features = append(features, feature)
}

// GetFeatures returns all registered features.
func GetFeatures() []Feature {
	return features
}

// ClearFeatures removes all registered features.
// This is primarily useful for testing.
func ClearFeatures() {
	features = nil
}

func init() {
	// Register OSS default features
	RegisterFeature(&KernelVersionFeature{})
	RegisterFeature(&CPUCapacityFeature{})
	RegisterFeature(&MemoryCapacityFeature{})
}
