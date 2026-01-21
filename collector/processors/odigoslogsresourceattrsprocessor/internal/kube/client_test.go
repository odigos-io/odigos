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
		Name:         "test-pod",
		Namespace:    "test-ns",
		WorkloadName: "test-deployment",
		WorkloadKind: WorkloadKindDeployment,
	}
	s.Equal("test-pod", pod.Name)
	s.Equal("test-ns", pod.Namespace)
	s.Equal("test-deployment", pod.WorkloadName)
	s.Equal(WorkloadKindDeployment, pod.WorkloadKind)
}

func (s *ClientTestSuite) TestGetPodMetadataNonExistent() {
	_, found := s.client.GetPodMetadata(types.UID("non-existent"))
	s.False(found)
}

func (s *ClientTestSuite) TestGetPodMetadataExisting() {
	podUID := types.UID("pod-uid-123")
	s.client.pods[podUID] = &PartialPodMetadata{
		WorkloadName: "my-service",
		WorkloadKind: WorkloadKindDeployment,
	}

	pod, found := s.client.GetPodMetadata(podUID)

	s.Require().True(found)
	s.Equal("my-service", pod.WorkloadName)
	s.Equal(WorkloadKindDeployment, pod.WorkloadKind)
}

func (s *ClientTestSuite) TestHandlePodAddDeployment() {
	// Pod owned by a ReplicaSet which is owned by a Deployment
	// The workload info is pre-extracted before calling handlePodAdd
	podMeta := PartialPodMetadata{
		UID:          types.UID("new-pod-uid"),
		Name:         "new-pod",
		Namespace:    "test-ns",
		WorkloadName: "deployment",
		WorkloadKind: WorkloadKindDeployment,
	}

	s.client.handlePodAdd(podMeta)

	pod, found := s.client.GetPodMetadata(types.UID("new-pod-uid"))
	s.Require().True(found)
	s.Equal("new-pod", pod.Name)
	s.Equal("test-ns", pod.Namespace)
	s.Equal("deployment", pod.WorkloadName)
	s.Equal(WorkloadKindDeployment, pod.WorkloadKind)
}

func (s *ClientTestSuite) TestHandlePodAddDaemonSet() {
	podMeta := PartialPodMetadata{
		UID:          types.UID("daemon-pod-uid"),
		Name:         "daemon-pod",
		Namespace:    "kube-system",
		WorkloadName: "my-daemonset",
		WorkloadKind: WorkloadKindDaemonSet,
	}

	s.client.handlePodAdd(podMeta)

	pod, found := s.client.GetPodMetadata(types.UID("daemon-pod-uid"))
	s.Require().True(found)
	s.Equal("daemon-pod", pod.Name)
	s.Equal("kube-system", pod.Namespace)
	s.Equal("my-daemonset", pod.WorkloadName)
	s.Equal(WorkloadKindDaemonSet, pod.WorkloadKind)
}

func (s *ClientTestSuite) TestHandlePodAddStatefulSet() {
	podMeta := PartialPodMetadata{
		UID:          types.UID("postgres-pod-uid"),
		Name:         "postgres-0",
		Namespace:    "database",
		WorkloadName: "postgres",
		WorkloadKind: WorkloadKindStatefulSet,
	}

	s.client.handlePodAdd(podMeta)

	pod, found := s.client.GetPodMetadata(types.UID("postgres-pod-uid"))
	s.Require().True(found)
	s.Equal("postgres-0", pod.Name)
	s.Equal("database", pod.Namespace)
	s.Equal("postgres", pod.WorkloadName)
	s.Equal(WorkloadKindStatefulSet, pod.WorkloadKind)
}

func (s *ClientTestSuite) TestHandlePodAddNoOwner() {
	// Pods without owners (empty workload name) should not be added to cache
	podMeta := PartialPodMetadata{
		UID:          types.UID("standalone-pod-uid"),
		Name:         "standalone-pod",
		Namespace:    "default",
		WorkloadName: "",
		WorkloadKind: "",
	}

	s.client.handlePodAdd(podMeta)

	_, found := s.client.GetPodMetadata(types.UID("standalone-pod-uid"))
	s.False(found)
}

func (s *ClientTestSuite) TestHandlePodAddUpdate() {
	// Add initial pod
	initialPodMeta := PartialPodMetadata{
		UID:          types.UID("update-uid"),
		Name:         "pod-to-update",
		Namespace:    "default",
		WorkloadName: "old-owner",
		WorkloadKind: WorkloadKindDeployment,
	}
	s.client.handlePodAdd(initialPodMeta)

	// Update pod with new owner reference
	updatedPodMeta := PartialPodMetadata{
		UID:          types.UID("update-uid"),
		Name:         "pod-to-update-renamed",
		Namespace:    "updated-ns",
		WorkloadName: "new-owner",
		WorkloadKind: WorkloadKindDeployment,
	}
	s.client.handlePodAdd(updatedPodMeta)

	pod, found := s.client.GetPodMetadata(types.UID("update-uid"))
	s.Require().True(found)
	s.Equal("pod-to-update-renamed", pod.Name)
	s.Equal("updated-ns", pod.Namespace)
	s.Equal("new-owner", pod.WorkloadName)
	s.Equal(WorkloadKindDeployment, pod.WorkloadKind)
}

func (s *ClientTestSuite) TestHandlePodDelete() {
	// Add a pod first
	podMeta := PartialPodMetadata{
		UID:          types.UID("delete-uid"),
		Name:         "pod-to-delete",
		Namespace:    "default",
		WorkloadName: "my-service",
		WorkloadKind: WorkloadKindDeployment,
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
	s.Equal(WorkloadKindDeployment, pod.WorkloadKind)
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

// ProcessDeleteQueueTestSuite tests the processDeleteQueue function
type ProcessDeleteQueueTestSuite struct {
	suite.Suite
	client *PodMetadataClient
}

func TestProcessDeleteQueueTestSuite(t *testing.T) {
	suite.Run(t, new(ProcessDeleteQueueTestSuite))
}

func (s *ProcessDeleteQueueTestSuite) SetupTest() {
	s.client = &PodMetadataClient{
		pods:        make(map[types.UID]*PartialPodMetadata),
		deleteQueue: []deleteRequest{},
	}
}

func (s *ProcessDeleteQueueTestSuite) TestProcessDeleteQueueRemovesOldItems() {
	// Add a pod to the cache
	podUID := types.UID("old-pod-uid")
	s.client.pods[podUID] = &PartialPodMetadata{
		Name:         "old-pod",
		Namespace:    "default",
		WorkloadName: "my-service",
		WorkloadKind: WorkloadKindDeployment,
	}

	// Add a delete request older than the deletion delay
	s.client.deleteQueue = []deleteRequest{
		{podUID: podUID, timestamp: time.Now().Add(-podDeletionDelay - time.Second)},
	}

	s.client.processDeleteQueue()

	// Pod should be removed from cache
	_, found := s.client.GetPodMetadata(podUID)
	s.False(found)

	// Queue should be empty
	s.Empty(s.client.deleteQueue)
}

func (s *ProcessDeleteQueueTestSuite) TestProcessDeleteQueueKeepsRecentItems() {
	// Add a pod to the cache
	podUID := types.UID("recent-pod-uid")
	s.client.pods[podUID] = &PartialPodMetadata{
		Name:         "recent-pod",
		Namespace:    "default",
		WorkloadName: "my-service",
		WorkloadKind: WorkloadKindDeployment,
	}

	// Add a delete request newer than the deletion delay
	s.client.deleteQueue = []deleteRequest{
		{podUID: podUID, timestamp: time.Now()},
	}

	s.client.processDeleteQueue()

	// Pod should still be in cache
	pod, found := s.client.GetPodMetadata(podUID)
	s.True(found)
	s.Equal("recent-pod", pod.Name)

	// Queue should still have the item
	s.Len(s.client.deleteQueue, 1)
}

func (s *ProcessDeleteQueueTestSuite) TestProcessDeleteQueueMixedItems() {
	// Add pods to the cache
	oldPodUID := types.UID("old-pod-uid")
	recentPodUID := types.UID("recent-pod-uid")

	s.client.pods[oldPodUID] = &PartialPodMetadata{
		Name:         "old-pod",
		Namespace:    "default",
		WorkloadName: "old-service",
		WorkloadKind: WorkloadKindDeployment,
	}
	s.client.pods[recentPodUID] = &PartialPodMetadata{
		Name:         "recent-pod",
		Namespace:    "default",
		WorkloadName: "recent-service",
		WorkloadKind: WorkloadKindDeployment,
	}

	// Add delete requests with different timestamps
	s.client.deleteQueue = []deleteRequest{
		{podUID: oldPodUID, timestamp: time.Now().Add(-podDeletionDelay - time.Second)},
		{podUID: recentPodUID, timestamp: time.Now()},
	}

	s.client.processDeleteQueue()

	// Old pod should be removed
	_, foundOld := s.client.GetPodMetadata(oldPodUID)
	s.False(foundOld)

	// Recent pod should still be in cache
	pod, foundRecent := s.client.GetPodMetadata(recentPodUID)
	s.True(foundRecent)
	s.Equal("recent-pod", pod.Name)

	// Queue should only have the recent item
	s.Len(s.client.deleteQueue, 1)
	s.Equal(recentPodUID, s.client.deleteQueue[0].podUID)
}

func (s *ProcessDeleteQueueTestSuite) TestProcessDeleteQueueEmptyQueue() {
	// Add a pod to the cache
	podUID := types.UID("pod-uid")
	s.client.pods[podUID] = &PartialPodMetadata{
		Name:         "pod",
		Namespace:    "default",
		WorkloadName: "my-service",
		WorkloadKind: WorkloadKindDeployment,
	}

	// Empty queue
	s.client.deleteQueue = []deleteRequest{}

	s.client.processDeleteQueue()

	// Pod should still be in cache
	pod, found := s.client.GetPodMetadata(podUID)
	s.True(found)
	s.Equal("pod", pod.Name)

	// Queue should still be empty
	s.Empty(s.client.deleteQueue)
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
			podMeta := PartialPodMetadata{
				UID:          types.UID("concurrent-uid"),
				Name:         "concurrent-pod",
				Namespace:    "default",
				WorkloadName: "concurrent-deployment",
				WorkloadKind: WorkloadKindDeployment,
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
			podMeta := PartialPodMetadata{
				UID:          types.UID("concurrent-uid"),
				Name:         "concurrent-pod-updated",
				Namespace:    "default",
				WorkloadName: "concurrent-deployment",
				WorkloadKind: WorkloadKindDeployment,
			}
			client.handlePodAdd(podMeta)
		}
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// If we got here without deadlock or race condition panic, the test passes
}

func (s *ConcurrencyTestSuite) TestConcurrentDeleteQueueAccess() {
	client := &PodMetadataClient{
		pods:        make(map[types.UID]*PartialPodMetadata),
		deleteQueue: []deleteRequest{},
	}

	var wg sync.WaitGroup
	wg.Add(3)

	// Adder goroutine - adds pods
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			podMeta := PartialPodMetadata{
				UID:          types.UID("concurrent-delete-uid"),
				Name:         "concurrent-delete-pod",
				Namespace:    "default",
				WorkloadName: "concurrent-deployment",
				WorkloadKind: WorkloadKindDeployment,
			}
			client.handlePodAdd(podMeta)
		}
	}()

	// Delete goroutine - queues deletes
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			podMeta := PartialPodMetadata{
				UID: types.UID("concurrent-delete-uid"),
			}
			client.handlePodDelete(podMeta)
		}
	}()

	// Process goroutine - processes delete queue
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			client.processDeleteQueue()
		}
	}()

	wg.Wait()

	// If we got here without deadlock or race condition panic, the test passes
}
