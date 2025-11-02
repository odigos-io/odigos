package odigosk8sresourcesexention

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	// copy of the const from api/k8sconsts/source.go
	// in order to avoid pulling in it's dependencies and create conflicts with collector dependencies
	SourceDataStreamLabelPrefix = "odigos.io/data-stream-"
)

type OdigosKsResources struct {
	dynClient  dynamic.Interface
	icInformer cache.SharedInformer
	stopCh     chan struct{}
	logger     *zap.Logger

	m                          sync.RWMutex
	workloadToDatastreamsCache map[WorkloadKey][]DatastreamName
}

var _ extension.Extension = (*OdigosKsResources)(nil)

func NewOdigosKsResources(
	set component.TelemetrySettings,
) (*OdigosKsResources, error) {
	logger := set.Logger
	k8sClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}
	// the `HTTPS_PROXY` we allow to set is used for exporting to https destinations.
	// we don't want to use it for accessing k8s api.
	k8sClusterConfig.Proxy = func(req *http.Request) (*url.URL, error) {
		return nil, nil
	}

	// Create dynamic client instead of typed client
	dynClient, err := dynamic.NewForConfig(k8sClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic kubernetes client: %w", err)
	}

	// Define the GVR (GroupVersionResource) for InstrumentationConfig
	// API Version: odigos.io/v1alpha1
	// Kind: InstrumentationConfig (resource name is usually plural and lowercase)
	gvr := schema.GroupVersionResource{
		Group:    "odigos.io",
		Version:  "v1alpha1",
		Resource: "instrumentationconfigs", // plural, lowercase
	}

	// Create informer for the generic resource
	informer := cache.NewSharedInformer(
		&cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				// Watch across all namespaces (empty string means all)
				return dynClient.Resource(gvr).Namespace("").List(context.Background(), opts)
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				return dynClient.Resource(gvr).Namespace("").Watch(context.Background(), opts)
			},
		},
		&unstructured.Unstructured{}, // Generic unstructured object
		5*time.Minute,
	)

	return &OdigosKsResources{
		dynClient:                  dynClient,
		icInformer:                 informer,
		stopCh:                     make(chan struct{}),
		logger:                     logger,
		workloadToDatastreamsCache: make(map[WorkloadKey][]DatastreamName),
	}, nil
}

func (c *OdigosKsResources) Start(_ context.Context, _ component.Host) error {
	// Add event handlers to process the resources
	c.icInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.HandleIcAddOrUpdate(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.HandleIcAddOrUpdate(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			c.HandleIcDelete(obj)
		},
	})

	go c.icInformer.Run(c.stopCh)
	return nil
}

func (c *OdigosKsResources) Shutdown(ctx context.Context) error {
	close(c.stopCh)
	return nil
}

func (c *OdigosKsResources) HandleIcAddOrUpdate(obj interface{}) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		c.logger.Error("unexpected object type", zap.Any("object", obj))
		return
	}

	ns := unstructuredObj.GetNamespace()
	name := unstructuredObj.GetName()
	workloadKey, err := InstrumentationConfigToWorkloadKey(ns, name)
	if err != nil {
		c.logger.Error("failed to calculate workload key from added or updated instrumentation config", zap.Error(err))
		return
	}

	labels := unstructuredObj.GetLabels()
	if labels == nil {
		return
	}

	// get datastream names from labels
	datastreamNames := make([]DatastreamName, 0)
	for labelKey, labelValue := range labels {
		// ignore datastream which are not marked as active
		if labelValue != "true" {
			continue
		}

		// only consider datastream labels
		if !strings.HasPrefix(labelKey, SourceDataStreamLabelPrefix) {
			continue
		}

		// collect the datastream name and save it to store in cache later on
		dataStreamName := strings.TrimPrefix(labelKey, SourceDataStreamLabelPrefix)
		datastreamNames = append(datastreamNames, DatastreamName(dataStreamName))
	}

	c.m.Lock()
	defer c.m.Unlock()
	c.workloadToDatastreamsCache[workloadKey] = datastreamNames
}

func (c *OdigosKsResources) HandleIcDelete(obj interface{}) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		c.logger.Error("unexpected object type", zap.Any("object", obj))
		return
	}

	ns := unstructuredObj.GetNamespace()
	name := unstructuredObj.GetName()
	workloadKey, err := InstrumentationConfigToWorkloadKey(ns, name)
	if err != nil {
		c.logger.Error("failed to calculate workload key from deleted instrumentation config", zap.Error(err))
		return
	}

	c.m.Lock()
	defer c.m.Unlock()
	delete(c.workloadToDatastreamsCache, workloadKey)
}

func (c *OdigosKsResources) GetDatastreamsForWorkload(workloadKey WorkloadKey) ([]DatastreamName, bool) {
	c.m.RLock()
	defer c.m.RUnlock()
	datastreams, ok := c.workloadToDatastreamsCache[workloadKey]
	return datastreams, ok
}

// Wait for cache to sync
func (c *OdigosKsResources) WaitForCacheSync(ctx context.Context) bool {
	return cache.WaitForCacheSync(ctx.Done(), c.icInformer.HasSynced)
}
