package kube

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/metadata/metadatainformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const nodeNameEnvVar = "NODE_NAME"

// PartialPodMetadata contains only the metadata fields we need from a pod
type PartialPodMetadata struct {
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
	m           sync.RWMutex
	deleteMut   sync.Mutex
	deleteQueue []deleteRequest
	pods        map[types.UID]*PartialPodMetadata
	podInformer cache.SharedIndexInformer
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

	_, err = c.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// Cache functions require "any"
		AddFunc: func(obj any) {
			if podMeta := extractPartialMetadata(obj); podMeta != nil {
				c.handlePodAdd(podMeta)
			}
		},
		DeleteFunc: func(obj any) {
			if podMeta := extractPartialMetadata(obj); podMeta != nil {
				c.handlePodDelete(podMeta)
			}
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add event handler: %w", err)
	}

	return c, nil
}

func extractPartialMetadata(obj any) *metav1.PartialObjectMetadata {
	if podMeta, ok := obj.(*metav1.PartialObjectMetadata); ok {
		return podMeta
	}
	if deleted, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		if podMeta, ok := deleted.Obj.(*metav1.PartialObjectMetadata); ok {
			return podMeta
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

func (c *PodMetadataClient) handlePodAdd(podMeta *metav1.PartialObjectMetadata) {
	workloadName, workloadKind := extractWorkloadInfo(podMeta)
	if workloadName == "" || workloadKind == "" {
		return
	}

	c.m.Lock()
	defer c.m.Unlock()
	c.pods[podMeta.UID] = &PartialPodMetadata{
		Name:         podMeta.Name,
		Namespace:    podMeta.Namespace,
		WorkloadName: workloadName,
		WorkloadKind: workloadKind,
	}
}

func (c *PodMetadataClient) handlePodUpdate(podMeta *metav1.PartialObjectMetadata) {
	workloadName, workloadKind := extractWorkloadInfo(podMeta)
	if workloadName == "" || workloadKind == "" {
		// Remove from cache if workload info can't be determined
		c.m.Lock()
		delete(c.pods, podMeta.UID)
		c.m.Unlock()
		return
	}

	c.m.Lock()
	defer c.m.Unlock()
	c.pods[podMeta.UID] = &PartialPodMetadata{
		Name:         podMeta.Name,
		Namespace:    podMeta.Namespace,
		WorkloadName: workloadName,
		WorkloadKind: workloadKind,
	}
}

func (c *PodMetadataClient) handlePodDelete(podMeta *metav1.PartialObjectMetadata) {
	c.m.Lock()
	defer c.m.Unlock()
	c.deleteQueue = append(c.deleteQueue, deleteRequest{podUID: podMeta.UID, timestamp: time.Now()})
}

func (c *PodMetadataClient) GetPodMetadata(uid types.UID) (*PartialPodMetadata, bool) {
	c.m.RLock()
	defer c.m.RUnlock()
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

	return nil
}
