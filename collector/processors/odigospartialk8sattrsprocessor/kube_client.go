package odigospartialk8sattrsprocessor

import (
	"fmt"

	"k8s.io/client-go/rest"

	"github.com/odigos-io/odigos/collector/processor/odigospartialk8sattrsprocessor/internal/kube"
)

// newKubeClient is a factory function that creates a kube.Client.
//
// This is defined as a variable rather than a regular function to allow
// tests to override it with a mock implementation. This pattern enables
// testing the processor's Start() lifecycle without requiring an actual
// Kubernetes cluster connection.
//
// In production, this always uses the default implementation which connects
// to the in-cluster Kubernetes API. In tests, mocks_test.go overrides this
// to return a mock client.
var newKubeClient = func() (kube.Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	client, err := kube.NewMetadataClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pod metadata client: %w", err)
	}
	return client, nil
}
