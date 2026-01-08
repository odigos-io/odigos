package kube

import (
	"sync"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

// ClientTestSuite tests the PodMetadataClient
type ClientTestSuite struct {
	suite.Suite
	client *PodMetadataClient
}

func (s *ClientTestSuite) SetupTest() {
	s.client = &PodMetadataClient{
		pods:        make(map[types.UID]*PartialPodMetadata),
		deleteQueue: []deleteRequest{},
	}
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) TestPartialPodMetadataStruct() {
	pod := &PartialPodMetadata{
		Name:         "test-pod",
		Namespace:    "test-ns",
		WorkloadName: "test-deployment",
		WorkloadKind: k8sconsts.WorkloadKindDeployment,
	}
	s.Equal("test-pod", pod.Name)
	s.Equal("test-ns", pod.Namespace)
	s.Equal("test-deployment", pod.WorkloadName)
	s.Equal(k8sconsts.WorkloadKindDeployment, pod.WorkloadKind)
}

func (s *ClientTestSuite) TestGetPodMetadata_NonExistent() {
	_, found := s.client.GetPodMetadata(types.UID("non-existent"))
	s.False(found)
}

func (s *ClientTestSuite) TestGetPodMetadata_Existing() {
	podUID := types.UID("pod-uid-123")
	s.client.pods[podUID] = &PartialPodMetadata{
		WorkloadName: "my-service",
		WorkloadKind: k8sconsts.WorkloadKindDeployment,
	}

	pod, found := s.client.GetPodMetadata(podUID)

	s.Require().True(found)
	s.Equal("my-service", pod.WorkloadName)
	s.Equal(k8sconsts.WorkloadKindDeployment, pod.WorkloadKind)
}

func (s *ClientTestSuite) TestHandlePodAdd_Deployment() {
	// Pod owned by a ReplicaSet which is owned by a Deployment
	// The extractWorkloadInfo should resolve this to Deployment
	podMeta := &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "new-pod",
			Namespace: "test-ns",
			UID:       types.UID("new-pod-uid"),
			OwnerReferences: []metav1.OwnerReference{
				{Name: "deployment-abc123", Kind: "ReplicaSet"},
			},
		},
	}

	s.client.handlePodAdd(podMeta)

	pod, found := s.client.GetPodMetadata(types.UID("new-pod-uid"))
	s.Require().True(found)
	s.Equal("new-pod", pod.Name)
	s.Equal("test-ns", pod.Namespace)
	// Note: extractWorkloadInfo uses the workload package which resolves ReplicaSet to Deployment
	// The workload name is derived from stripping the suffix from the ReplicaSet name
	s.Equal("deployment", pod.WorkloadName)
	s.Equal(k8sconsts.WorkloadKindDeployment, pod.WorkloadKind)
}

func (s *ClientTestSuite) TestHandlePodAdd_DaemonSet() {
	podMeta := &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "daemon-pod",
			Namespace: "kube-system",
			UID:       types.UID("daemon-pod-uid"),
			OwnerReferences: []metav1.OwnerReference{
				{Name: "my-daemonset", Kind: "DaemonSet"},
			},
		},
	}

	s.client.handlePodAdd(podMeta)

	pod, found := s.client.GetPodMetadata(types.UID("daemon-pod-uid"))
	s.Require().True(found)
	s.Equal("daemon-pod", pod.Name)
	s.Equal("kube-system", pod.Namespace)
	s.Equal("my-daemonset", pod.WorkloadName)
	s.Equal(k8sconsts.WorkloadKindDaemonSet, pod.WorkloadKind)
}

func (s *ClientTestSuite) TestHandlePodAdd_StatefulSet() {
	podMeta := &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgres-0",
			Namespace: "database",
			UID:       types.UID("postgres-pod-uid"),
			OwnerReferences: []metav1.OwnerReference{
				{Name: "postgres", Kind: "StatefulSet"},
			},
		},
	}

	s.client.handlePodAdd(podMeta)

	pod, found := s.client.GetPodMetadata(types.UID("postgres-pod-uid"))
	s.Require().True(found)
	s.Equal("postgres-0", pod.Name)
	s.Equal("database", pod.Namespace)
	s.Equal("postgres", pod.WorkloadName)
	s.Equal(k8sconsts.WorkloadKindStatefulSet, pod.WorkloadKind)
}

func (s *ClientTestSuite) TestHandlePodAdd_NoOwner() {
	podMeta := &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "standalone-pod",
			Namespace:       "default",
			UID:             types.UID("standalone-pod-uid"),
			OwnerReferences: []metav1.OwnerReference{},
		},
	}

	s.client.handlePodAdd(podMeta)

	pod, found := s.client.GetPodMetadata(types.UID("standalone-pod-uid"))
	s.Require().True(found)
	s.Equal("standalone-pod", pod.Name)
	s.Equal("default", pod.Namespace)
	s.Equal("", pod.WorkloadName)
	s.Equal(k8sconsts.WorkloadKind(""), pod.WorkloadKind)
}

func (s *ClientTestSuite) TestHandlePodUpdate() {
	// Add initial pod
	initialPodMeta := &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-to-update",
			Namespace: "default",
			UID:       types.UID("update-uid"),
			OwnerReferences: []metav1.OwnerReference{
				{Name: "old-owner-abc123", Kind: "ReplicaSet"},
			},
		},
	}
	s.client.handlePodAdd(initialPodMeta)

	// Update pod with new owner reference
	updatedPodMeta := &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-to-update-renamed",
			Namespace: "updated-ns",
			UID:       types.UID("update-uid"),
			OwnerReferences: []metav1.OwnerReference{
				{Name: "new-owner-xyz789", Kind: "ReplicaSet"},
			},
		},
	}
	s.client.handlePodUpdate(updatedPodMeta)

	pod, found := s.client.GetPodMetadata(types.UID("update-uid"))
	s.Require().True(found)
	s.Equal("pod-to-update-renamed", pod.Name)
	s.Equal("updated-ns", pod.Namespace)
	s.Equal("new-owner", pod.WorkloadName)
	s.Equal(k8sconsts.WorkloadKindDeployment, pod.WorkloadKind)
}

func (s *ClientTestSuite) TestHandlePodDelete() {
	// Add a pod first
	podMeta := &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-to-delete",
			Namespace: "default",
			UID:       types.UID("delete-uid"),
			OwnerReferences: []metav1.OwnerReference{
				{Name: "my-service-abc123", Kind: "ReplicaSet"},
			},
		},
	}
	s.client.handlePodAdd(podMeta)

	// Delete the pod
	s.client.handlePodDelete(podMeta)

	// Verify pod is queued for deletion
	s.Len(s.client.deleteQueue, 1)
	s.Equal(types.UID("delete-uid"), s.client.deleteQueue[0].podUID)

	// Pod should still be in cache until cleanup runs
	pod, found := s.client.GetPodMetadata(types.UID("delete-uid"))
	s.True(found)
	s.Equal("pod-to-delete", pod.Name)
	s.Equal("default", pod.Namespace)
	s.Equal("my-service", pod.WorkloadName)
	s.Equal(k8sconsts.WorkloadKindDeployment, pod.WorkloadKind)
}

// ExtractPartialMetadataTestSuite tests the extractPartialMetadata function
type ExtractPartialMetadataTestSuite struct {
	suite.Suite
}

func TestExtractPartialMetadataTestSuite(t *testing.T) {
	suite.Run(t, new(ExtractPartialMetadataTestSuite))
}

func (s *ExtractPartialMetadataTestSuite) TestDirectPartialObjectMetadata() {
	input := &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "direct-pod",
			Namespace: "default",
			UID:       types.UID("direct-uid"),
		},
	}

	result := extractPartialMetadata(input)

	s.Require().NotNil(result)
	s.Equal("direct-pod", result.Name)
	s.Equal("default", result.Namespace)
	s.Equal(types.UID("direct-uid"), result.UID)
}

func (s *ExtractPartialMetadataTestSuite) TestDeletedFinalStateUnknownWrapper() {
	input := cache.DeletedFinalStateUnknown{
		Key: "default/deleted-pod",
		Obj: &metav1.PartialObjectMetadata{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deleted-pod",
				Namespace: "default",
				UID:       types.UID("deleted-uid"),
			},
		},
	}

	result := extractPartialMetadata(input)

	s.Require().NotNil(result)
	s.Equal("deleted-pod", result.Name)
	s.Equal("default", result.Namespace)
	s.Equal(types.UID("deleted-uid"), result.UID)
}

func (s *ExtractPartialMetadataTestSuite) TestNilInput() {
	result := extractPartialMetadata(nil)
	s.Nil(result)
}

func (s *ExtractPartialMetadataTestSuite) TestWrongType() {
	result := extractPartialMetadata("some string")
	s.Nil(result)
}

func (s *ExtractPartialMetadataTestSuite) TestDeletedFinalStateUnknownWrongInnerType() {
	input := cache.DeletedFinalStateUnknown{
		Key: "some-key",
		Obj: "not a pod metadata",
	}
	result := extractPartialMetadata(input)
	s.Nil(result)
}

// MiscTestSuite tests miscellaneous functionality
type MiscTestSuite struct {
	suite.Suite
}

func TestMiscTestSuite(t *testing.T) {
	suite.Run(t, new(MiscTestSuite))
}

func (s *MiscTestSuite) TestDeleteRequest() {
	now := time.Now()
	req := deleteRequest{
		podUID:    types.UID("test-uid"),
		timestamp: now,
	}

	s.Equal(types.UID("test-uid"), req.podUID)
	s.Equal(now, req.timestamp)
}

func (s *MiscTestSuite) TestClientInterface() {
	// Verify PodMetadataClient implements Client interface
	var _ Client = (*PodMetadataClient)(nil)
}

// ConcurrencyTestSuite tests concurrent access
type ConcurrencyTestSuite struct {
	suite.Suite
}

func TestConcurrencyTestSuite(t *testing.T) {
	suite.Run(t, new(ConcurrencyTestSuite))
}

func (s *ConcurrencyTestSuite) TestConcurrentAccess() {
	client := &PodMetadataClient{
		pods:        make(map[types.UID]*PartialPodMetadata),
		deleteQueue: []deleteRequest{},
	}

	var wg sync.WaitGroup
	wg.Add(3)

	// Writer goroutine - adds pods
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			podMeta := &metav1.PartialObjectMetadata{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "concurrent-pod",
					Namespace: "default",
					UID:       types.UID("concurrent-uid"),
				},
			}
			client.handlePodAdd(podMeta)
		}
	}()

	// Reader goroutine - reads pods
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			client.GetPodMetadata(types.UID("concurrent-uid"))
		}
	}()

	// Updater goroutine - updates pods
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			podMeta := &metav1.PartialObjectMetadata{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "concurrent-pod-updated",
					Namespace: "default",
					UID:       types.UID("concurrent-uid"),
				},
			}
			client.handlePodUpdate(podMeta)
		}
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// If we got here without deadlock or race condition panic, the test passes
}
