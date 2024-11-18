package version

import (
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

// GetKubernetesVersion returns the Kubernetes version of the cluster
// This util function is intended to be called once during the initialization.
// Do not call this from reconcile or hot path.
func GetKubernetesVersion() (*version.Version, error) {
	// Create a Kubernetes REST config directly
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// Create a discovery client using the config
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	// Retrieve the server version
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return nil, err
	}

	// Parse and return the version
	return version.Parse(serverVersion.String())
}
