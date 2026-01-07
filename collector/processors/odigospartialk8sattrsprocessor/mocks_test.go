package odigospartialk8sattrsprocessor

import (
	"k8s.io/apimachinery/pkg/types"

	"github.com/odigos-io/odigos/collector/processor/odigospartialk8sattrsprocessor/internal/kube"
)

// mockKubeClient is a mock implementation of kube.Client for testing
type mockKubeClient struct {
	pods    map[types.UID]*kube.PartialPodMetadata
	started bool
	stopped bool
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

func (m *mockKubeClient) Start(stopCh <-chan struct{}) error {
	m.started = true
	return nil
}

func (m *mockKubeClient) Stop() {
	m.stopped = true
}

func (m *mockKubeClient) AddPod(uid types.UID, serviceName, name, namespace, ownerName, ownerKind string) {
	m.pods[uid] = &kube.PartialPodMetadata{
		ServiceName: serviceName,
		Name:        name,
		Namespace:   namespace,
		OwnerName:   ownerName,
		OwnerKind:   ownerKind,
	}
}
