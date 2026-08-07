package kube

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/metadata/metadatainformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	nodeNameEnvVar                = "NODE_NAME"
	podDeletionDelay              = 30 * time.Second
	podDeleteQueueProcessInterval = 10 * time.Second
)

// PartialPodMetadata contains only the metadata fields we need from a pod
type PartialPodMetadata struct {
	UID          types.UID
	Name         string
	Namespace    string
	WorkloadName string       // The resolved workload name (e.g., "my-app" for a Deployment)
	WorkloadKind WorkloadKind // The resolved workload kind (e.g., "Deployment", "DaemonSet")
}

type Client interface {
	GetPodMetadata(uid types.UID) (*PartialPodMetadata, bool)
	Start(ctx context.Context) error
}

type deleteRequest struct {
	podUID    types.UID
	timestamp time.Time
}

type PodMetadataClient struct {
	podsMapMutex     sync.RWMutex
	deleteQueueMutex sync.Mutex
	deleteQueue      []deleteRequest
	pods             map[types.UID]*PartialPodMetadata
	podInformer      cache.SharedIndexInformer
}

var podGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "pods",
}

func NewMetadataClient(config *rest.Config) (Client, error) {
	metadataClient, err := metadata.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata client: %w", err)
	}

	c := &PodMetadataClient{
		pods: map[types.UID]*PartialPodMetadata{},
	}

	nodeName := os.Getenv(nodeNameEnvVar)
	if nodeName == "" {
		return nil, fmt.Errorf("%s environment variable not set", nodeNameEnvVar)
	}

	// Create filtered informer factory to only watch pods on this node
	tweakListOptions := func(options *metav1.ListOptions) {
		options.FieldSelector = fields.OneTermEqualSelector("spec.nodeName", nodeName).String()
	}
	factory := metadatainformer.NewFilteredSharedInformerFactory(metadataClient, 0, metav1.NamespaceAll, tweakListOptions)
	c.podInformer = factory.ForResource(podGVR).Informer()

	// Strip managedFields from cached PartialObjectMetadata to reduce memory usage.
	// The informer cache retains full ObjectMeta including managedFields which are
	// unused by this processor and consume ~20MB at scale (~40KB per pod).
	if err := c.podInformer.SetTransform(stripManagedFields); err != nil {
		return nil, fmt.Errorf("failed to set informer transform: %w", err)
	}

	_, err = c.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			podMeta := extractPartialMetadata(obj)
			if podMeta == nil {
				return
			}
			c.handlePodAdd(*podMeta)
		},
		DeleteFunc: func(obj any) {
			podMeta := extractPartialMetadata(obj)
			if podMeta == nil {
				return
			}
			c.handlePodDelete(*podMeta)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add event handler: %w", err)
	}

	return c, nil
}

// stripManagedFields is a cache.TransformFunc that nils out
// ObjectMeta.ManagedFields on cached objects. ManagedFields hold
// server-side-apply field ownership data which Odigos does not consume but
// which costs ~40KB per pod at scale. Mirrors
// sigs.k8s.io/controller-runtime/pkg/cache.TransformStripManagedFields (used
// in odiglet, frontend, scheduler, instrumentor and autoscaler) — copied here
// to keep the collector module free of the controller-runtime dependency. The
// nil-check guards against kubernetes/kubernetes#124337.
func stripManagedFields(obj any) (any, error) {
	if accessor, err := meta.Accessor(obj); err == nil && accessor.GetManagedFields() != nil {
		accessor.SetManagedFields(nil)
	}
	return obj, nil
}

func extractPartialMetadata(obj any) *PartialPodMetadata {
	if podMeta, ok := obj.(*metav1.PartialObjectMetadata); ok {
		workloadName, workloadKind := extractWorkloadInfo(podMeta)
		return &PartialPodMetadata{
			UID:          podMeta.UID,
			Name:         podMeta.Name,
			Namespace:    podMeta.Namespace,
			WorkloadName: workloadName,
			WorkloadKind: workloadKind,
		}
	}
	if deleted, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		if podMeta, ok := deleted.Obj.(*metav1.PartialObjectMetadata); ok {
			workloadName, workloadKind := extractWorkloadInfo(podMeta)
			return &PartialPodMetadata{
				UID:          podMeta.UID,
				Name:         podMeta.Name,
				Namespace:    podMeta.Namespace,
				WorkloadName: workloadName,
				WorkloadKind: workloadKind,
			}
		}
	}
	return nil
}

// extractWorkloadInfo resolves the workload name and kind from owner references.
// Handles ReplicaSet → Deployment/ArgoRollout resolution.
func extractWorkloadInfo(podMeta *metav1.PartialObjectMetadata) (name string, kind WorkloadKind) {
	for _, ownerRef := range podMeta.OwnerReferences {
		// Create a minimal Pod with labels for Argo Rollout detection
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: podMeta.Labels,
			},
		}

		workloadName, workloadKind, err := getWorkloadNameAndKind(ownerRef.Name, ownerRef.Kind, pod)
		if err == nil {
			return workloadName, workloadKind
		}
	}

	return "", ""
}

func (c *PodMetadataClient) handlePodAdd(partialPodMetaData PartialPodMetadata) {
	// Skip pods without workload info (e.g., standalone pods without owner references)
	if partialPodMetaData.WorkloadName == "" {
		return
	}
	c.podsMapMutex.Lock()
	defer c.podsMapMutex.Unlock()
	c.pods[partialPodMetaData.UID] = &partialPodMetaData
}

func (c *PodMetadataClient) handlePodDelete(partialPodMetaData PartialPodMetadata) {
	c.deleteQueueMutex.Lock()
	defer c.deleteQueueMutex.Unlock()
	c.deleteQueue = append(c.deleteQueue, deleteRequest{podUID: partialPodMetaData.UID, timestamp: time.Now()})
}

func (c *PodMetadataClient) GetPodMetadata(uid types.UID) (*PartialPodMetadata, bool) {
	c.podsMapMutex.RLock()
	defer c.podsMapMutex.RUnlock()
	pod, ok := c.pods[uid]
	return pod, ok
}

func (c *PodMetadataClient) Start(ctx context.Context) error {
	go c.podInformer.Run(ctx.Done())

	syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Wait for pod cache to sync
	if !cache.WaitForCacheSync(syncCtx.Done(), c.podInformer.HasSynced) {
		return fmt.Errorf("timed out waiting for pod metadata cache to sync")
	}

	go c.runDeleteQueueProcessor(ctx)

	return nil
}

// Goroutine to process the delete queue and delete pods from the cache after the deletion delay.
func (c *PodMetadataClient) runDeleteQueueProcessor(ctx context.Context) {
	ticker := time.NewTicker(podDeleteQueueProcessInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.processDeleteQueue()
		}
	}
}

func (c *PodMetadataClient) processDeleteQueue() {
	c.deleteQueueMutex.Lock()
	defer c.deleteQueueMutex.Unlock()

	now := time.Now()
	var toDelete []types.UID
	remaining := c.deleteQueue[:0]

	for _, request := range c.deleteQueue {
		if now.Sub(request.timestamp) >= podDeletionDelay {
			toDelete = append(toDelete, request.podUID)
		} else {
			remaining = append(remaining, request)
		}
	}

	c.deleteQueue = remaining

	if len(toDelete) > 0 {
		c.podsMapMutex.Lock()
		for _, uid := range toDelete {
			delete(c.pods, uid)
		}
		c.podsMapMutex.Unlock()
	}
}
