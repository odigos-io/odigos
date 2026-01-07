package kube

import (
	"sync"
	"testing"
	"time"

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
		ServiceName: "test-deployment",
		Name:        "test-pod",
		Namespace:   "test-ns",
		OwnerName:   "test-deployment-abc123",
		OwnerKind:   "ReplicaSet",
	}
	s.Equal("test-deployment", pod.ServiceName)
	s.Equal("test-pod", pod.Name)
	s.Equal("test-ns", pod.Namespace)
	s.Equal("test-deployment-abc123", pod.OwnerName)
	s.Equal("ReplicaSet", pod.OwnerKind)
}

func (s *ClientTestSuite) TestGetPodMetadata_NonExistent() {
	_, found := s.client.GetPodMetadata(types.UID("non-existent"))
	s.False(found)
}

func (s *ClientTestSuite) TestGetPodMetadata_Existing() {
	podUID := types.UID("pod-uid-123")
	s.client.pods[podUID] = &PartialPodMetadata{
		ServiceName: "my-service",
	}

	pod, found := s.client.GetPodMetadata(podUID)

	s.Require().True(found)
	s.Equal("my-service", pod.ServiceName)
}

func (s *ClientTestSuite) TestHandlePodAdd() {
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
	s.Equal("deployment", pod.ServiceName) // "deployment-abc123" -> "deployment"
	s.Equal("new-pod", pod.Name)
	s.Equal("test-ns", pod.Namespace)
	s.Equal("deployment-abc123", pod.OwnerName)
	s.Equal("ReplicaSet", pod.OwnerKind)
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
	s.Equal("new-owner", pod.ServiceName) // "new-owner-xyz789" -> "new-owner"
	s.Equal("pod-to-update-renamed", pod.Name)
	s.Equal("updated-ns", pod.Namespace)
	s.Equal("new-owner-xyz789", pod.OwnerName)
	s.Equal("ReplicaSet", pod.OwnerKind)
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
	s.Equal("my-service", pod.ServiceName)
	s.Equal("pod-to-delete", pod.Name)
	s.Equal("default", pod.Namespace)
	s.Equal("my-service-abc123", pod.OwnerName)
	s.Equal("ReplicaSet", pod.OwnerKind)
}

// ExtractServiceNameTestSuite tests the extractServiceName function
type ExtractServiceNameTestSuite struct {
	suite.Suite
}

func TestExtractServiceNameTestSuite(t *testing.T) {
	suite.Run(t, new(ExtractServiceNameTestSuite))
}

func (s *ExtractServiceNameTestSuite) TestSingleOwnerWithSuffix() {
	ownerRefs := []metav1.OwnerReference{
		{Name: "my-app-5d4b7c8f9", Kind: "ReplicaSet"},
	}
	s.Equal("my-app", extractServiceName(ownerRefs))
}

func (s *ExtractServiceNameTestSuite) TestMultipleHyphens() {
	ownerRefs := []metav1.OwnerReference{
		{Name: "frontend-api-v2-abc123", Kind: "ReplicaSet"},
	}
	s.Equal("frontend-api-v2", extractServiceName(ownerRefs))
}

func (s *ExtractServiceNameTestSuite) TestEmptyOwnerRefs() {
	s.Equal("", extractServiceName([]metav1.OwnerReference{}))
}

func (s *ExtractServiceNameTestSuite) TestNilOwnerRefs() {
	s.Equal("", extractServiceName(nil))
}

func (s *ExtractServiceNameTestSuite) TestMultipleOwnerRefs() {
	ownerRefs := []metav1.OwnerReference{
		{Name: "owner1-abc"},
		{Name: "owner2-def"},
	}
	s.Equal("", extractServiceName(ownerRefs))
}

func (s *ExtractServiceNameTestSuite) TestNoHyphen() {
	ownerRefs := []metav1.OwnerReference{
		{Name: "nohyphen", Kind: "ReplicaSet"},
	}
	s.Equal("", extractServiceName(ownerRefs))
}

func (s *ExtractServiceNameTestSuite) TestDaemonSetStyle() {
	ownerRefs := []metav1.OwnerReference{
		{Name: "odiglet", Kind: "DaemonSet"},
	}
	s.Equal("", extractServiceName(ownerRefs))
}

// ExtractOwnerInfoTestSuite tests the extractOwnerInfo function
type ExtractOwnerInfoTestSuite struct {
	suite.Suite
}

func TestExtractOwnerInfoTestSuite(t *testing.T) {
	suite.Run(t, new(ExtractOwnerInfoTestSuite))
}

func (s *ExtractOwnerInfoTestSuite) TestReplicaSet() {
	ownerRefs := []metav1.OwnerReference{
		{Name: "my-app-5d4b7c8f9", Kind: "ReplicaSet"},
	}
	name, kind := extractOwnerInfo(ownerRefs)
	s.Equal("my-app-5d4b7c8f9", name)
	s.Equal("ReplicaSet", kind)
}

func (s *ExtractOwnerInfoTestSuite) TestDaemonSet() {
	ownerRefs := []metav1.OwnerReference{
		{Name: "odiglet", Kind: "DaemonSet"},
	}
	name, kind := extractOwnerInfo(ownerRefs)
	s.Equal("odiglet", name)
	s.Equal("DaemonSet", kind)
}

func (s *ExtractOwnerInfoTestSuite) TestStatefulSet() {
	ownerRefs := []metav1.OwnerReference{
		{Name: "postgres", Kind: "StatefulSet"},
	}
	name, kind := extractOwnerInfo(ownerRefs)
	s.Equal("postgres", name)
	s.Equal("StatefulSet", kind)
}

func (s *ExtractOwnerInfoTestSuite) TestJob() {
	ownerRefs := []metav1.OwnerReference{
		{Name: "batch-job-abc123", Kind: "Job"},
	}
	name, kind := extractOwnerInfo(ownerRefs)
	s.Equal("batch-job-abc123", name)
	s.Equal("Job", kind)
}

func (s *ExtractOwnerInfoTestSuite) TestEmptyOwnerRefs() {
	name, kind := extractOwnerInfo([]metav1.OwnerReference{})
	s.Equal("", name)
	s.Equal("", kind)
}

func (s *ExtractOwnerInfoTestSuite) TestNilOwnerRefs() {
	name, kind := extractOwnerInfo(nil)
	s.Equal("", name)
	s.Equal("", kind)
}

func (s *ExtractOwnerInfoTestSuite) TestMultipleOwnerRefs() {
	ownerRefs := []metav1.OwnerReference{
		{Name: "owner1", Kind: "ReplicaSet"},
		{Name: "owner2", Kind: "DaemonSet"},
	}
	name, kind := extractOwnerInfo(ownerRefs)
	s.Equal("", name)
	s.Equal("", kind)
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

// StopTestSuite tests the Stop functionality
type StopTestSuite struct {
	suite.Suite
}

func TestStopTestSuite(t *testing.T) {
	suite.Run(t, new(StopTestSuite))
}

func (s *StopTestSuite) TestStopWithActiveChannel() {
	client := &PodMetadataClient{
		pods:   make(map[types.UID]*PartialPodMetadata),
		stopCh: make(chan struct{}),
	}

	s.NotPanics(func() {
		client.Stop()
	})

	// Channel should be closed
	select {
	case <-client.stopCh:
		// Channel is closed, expected
	default:
		s.Fail("stopCh should be closed after Stop()")
	}
}

func (s *StopTestSuite) TestStopWithNilChannel() {
	client := &PodMetadataClient{
		pods:   make(map[types.UID]*PartialPodMetadata),
		stopCh: nil,
	}

	s.NotPanics(func() {
		client.Stop()
	})
}

func (s *StopTestSuite) TestStopIdempotent() {
	client := &PodMetadataClient{
		pods:   make(map[types.UID]*PartialPodMetadata),
		stopCh: make(chan struct{}),
	}

	// First stop
	client.Stop()

	// Second stop should not panic (idempotent)
	s.NotPanics(func() {
		client.Stop()
	})
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
