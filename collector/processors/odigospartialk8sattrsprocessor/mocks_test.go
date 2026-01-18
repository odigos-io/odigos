package odigospartialk8sattrsprocessor

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	"github.com/odigos-io/odigos/collector/processor/odigospartialk8sattrsprocessor/internal/kube"
)

func init() {
	// Override newKubeClient so that all processors created during tests
	// use a mock client instead of trying to connect to a real Kubernetes cluster.
	newKubeClient = func() (kube.Client, error) {
		return newMockKubeClient(), nil
	}
}

// mockKubeClient is a mock implementation of kube.Client for testing
type mockKubeClient struct {
	pods    map[types.UID]*kube.PartialPodMetadata
	started bool
}

func newMockKubeClient() *mockKubeClient {
	return &mockKubeClient{
		pods: make(map[types.UID]*kube.PartialPodMetadata),
	}
}

func (m *mockKubeClient) GetPodMetadata(uid types.UID) (*kube.PartialPodMetadata, bool) {
	pod, ok := m.pods[uid]
	return pod, ok
}

func (m *mockKubeClient) Start(ctx context.Context) error {
	m.started = true
	return nil
}

func (m *mockKubeClient) AddPod(uid types.UID, workloadName string, workloadKind kube.WorkloadKind, name, namespace string) {
	m.pods[uid] = &kube.PartialPodMetadata{
		Name:         name,
		Namespace:    namespace,
		WorkloadName: workloadName,
		WorkloadKind: workloadKind,
	}
}
