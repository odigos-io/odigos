package kube

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/metadata/metadatainformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// PartialPodMetadata contains only the metadata fields we need from a pod
type PartialPodMetadata struct {
	ServiceName string
	Name        string
	Namespace   string
	OwnerName   string // The owner reference name (e.g., "my-app-abc123" for ReplicaSet)
	OwnerKind   string // The owner reference kind (e.g., "ReplicaSet", "DaemonSet")
}

type Client interface {
	GetPodMetadata(uid types.UID) (*PartialPodMetadata, bool)
	Start(stopCh <-chan struct{}) error
	Stop()
}

type deleteRequest struct {
	podUID    types.UID
	timestamp time.Time
}

type PodMetadataClient struct {
	m           sync.RWMutex
	deleteMut   sync.Mutex
	deleteQueue []deleteRequest
	// NOTE: we add pods to the cache even if we can't determine a service name
	pods     map[types.UID]*PartialPodMetadata
	informer cache.SharedIndexInformer
	stopCh   chan struct{}
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

	factory := metadatainformer.NewSharedInformerFactory(metadataClient, 0)
	c.informer = factory.ForResource(podGVR).Informer()

	_, err = c.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// Cache functions require "any"
		AddFunc: func(obj any) {
			if podMeta := extractPartialMetadata(obj); podMeta != nil {
				c.handlePodAdd(podMeta)
			}
		},
		UpdateFunc: func(oldObj, newObj any) {
			if podMeta := extractPartialMetadata(newObj); podMeta != nil {
				c.handlePodUpdate(podMeta)
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

func extractServiceName(ownerRefs []metav1.OwnerReference) string {
	if len(ownerRefs) != 1 {
		return ""
	}
	ownerName := ownerRefs[0].Name
	// Strip the suffix (e.g., "my-app-5d4b7c8f9" -> "my-app")
	hyphenIndex := strings.LastIndex(ownerName, "-")
	if hyphenIndex == -1 {
		return ""
	}
	return ownerName[:hyphenIndex]
}

// extractOwnerInfo extracts the owner name and kind from owner references.
// Returns empty strings if owner info cannot be determined.
func extractOwnerInfo(ownerRefs []metav1.OwnerReference) (name, kind string) {
	if len(ownerRefs) != 1 {
		return "", ""
	}
	return ownerRefs[0].Name, ownerRefs[0].Kind
}

func (c *PodMetadataClient) handlePodAdd(podMeta *metav1.PartialObjectMetadata) {
	c.m.Lock()
	defer c.m.Unlock()
	serviceName := extractServiceName(podMeta.OwnerReferences)
	ownerName, ownerKind := extractOwnerInfo(podMeta.OwnerReferences)
	c.pods[podMeta.UID] = &PartialPodMetadata{
		ServiceName: serviceName,
		Name:        podMeta.Name,
		Namespace:   podMeta.Namespace,
		OwnerName:   ownerName,
		OwnerKind:   ownerKind,
	}
}

func (c *PodMetadataClient) handlePodUpdate(podMeta *metav1.PartialObjectMetadata) {
	c.m.Lock()
	defer c.m.Unlock()
	serviceName := extractServiceName(podMeta.OwnerReferences)
	ownerName, ownerKind := extractOwnerInfo(podMeta.OwnerReferences)
	c.pods[podMeta.UID] = &PartialPodMetadata{
		ServiceName: serviceName,
		Name:        podMeta.Name,
		Namespace:   podMeta.Namespace,
		OwnerName:   ownerName,
		OwnerKind:   ownerKind,
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

func (c *PodMetadataClient) Start(stopCh <-chan struct{}) error {
	c.stopCh = make(chan struct{})
	go c.informer.Run(c.stopCh)

	if stopCh != nil {
		go func() {
			<-stopCh
			c.Stop()
		}()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		return fmt.Errorf("timed out waiting for pod metadata cache to sync")
	}

	return nil
}

func (c *PodMetadataClient) Stop() {
	if c.stopCh != nil {
		select {
		case <-c.stopCh:
			// Already closed
		default:
			close(c.stopCh)
		}
	}
}
