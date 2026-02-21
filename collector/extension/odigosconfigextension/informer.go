package odigosconfigextension

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	k8scache "k8s.io/client-go/tools/cache"

	"github.com/odigos-io/odigos/collector/extension/odigosconfigextension/api"
)

const (
	instrumentationConfigGroup    = "odigos.io"
	instrumentationConfigVersion  = "v1alpha1"
	instrumentationConfigResource = "instrumentationconfigs"
	resyncPeriod                  = 10 * time.Minute
)

var instrumentationConfigGVR = schema.GroupVersionResource{
	Group:    instrumentationConfigGroup,
	Version:  instrumentationConfigVersion,
	Resource: instrumentationConfigResource,
}

// startInformer starts a dynamic informer for InstrumentationConfigs and updates the extension's cache.
// It runs until ctx is cancelled. The cache is keyed by workload (namespace/kind/name).
// If not running in a cluster (e.g. InClusterConfig fails), the informer is not started
// and the cache remains empty; the extension still starts successfully.
func (o *OdigosConfig) startInformer(ctx context.Context) error {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		o.logger.Warn("not running in-cluster, instrumentation config cache will be empty", zap.Error(err))
		return nil
	}

	client, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(client, resyncPeriod)
	informer := factory.ForResource(instrumentationConfigGVR).Informer()

	_, err = informer.AddEventHandler(k8scache.ResourceEventHandlerFuncs{
		AddFunc:    o.handleInstrumentationConfig,
		UpdateFunc: func(_, newObj interface{}) { o.handleInstrumentationConfig(newObj) },
		DeleteFunc: o.handleInstrumentationConfigDelete,
	})
	if err != nil {
		return err
	}

	factory.Start(ctx.Done())

	synced := factory.WaitForCacheSync(ctx.Done())
	if !synced[instrumentationConfigGVR] {
		o.logger.Warn("instrumentationconfig informer cache sync did not complete")
	}

	return nil
}

func (o *OdigosConfig) handleInstrumentationConfig(obj interface{}) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		o.logger.Debug("informer received non-unstructured object", zap.String("type", fmt.Sprintf("%T", obj)))
		return
	}
	key, cfg := instrumentationConfigToWorkloadSampling(u)
	if key == "" {
		return
	}
	o.cache.Set(key, cfg)
	o.logger.Debug("updated workload sampling cache", zap.String("workload", key))
}

func (o *OdigosConfig) handleInstrumentationConfigDelete(obj interface{}) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		if deleted, ok := obj.(k8scache.DeletedFinalStateUnknown); ok {
			o.handleInstrumentationConfigDelete(deleted.Obj)
		}
		return
	}
	key := workloadKeyFromOwnerRef(u)
	if key != "" {
		o.cache.Delete(key)
		o.logger.Debug("removed workload from sampling cache", zap.String("workload", key))
	}
}

func instrumentationConfigToWorkloadSampling(u *unstructured.Unstructured) (workloadKey string, cfg *WorkloadSamplingConfig) {
	key := workloadKeyFromOwnerRef(u)
	if key == "" {
		return "", nil
	}
	specMap, ok, _ := unstructured.NestedMap(u.Object, "spec")
	if !ok || len(specMap) == 0 {
		return key, &WorkloadSamplingConfig{}
	}
	var spec api.InstrumentationConfigSpec
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(specMap, &spec); err != nil {
		return key, &WorkloadSamplingConfig{}
	}

	cfg = &WorkloadSamplingConfig{
		ContainersHeadSampling: make(map[string]*api.HeadSamplingConfig),
	}
	for _, c := range spec.Containers {
		if c.ContainerName == "" {
			continue
		}
		if c.Traces != nil && c.Traces.HeadSampling != nil {
			cfg.ContainersHeadSampling[c.ContainerName] = c.Traces.HeadSampling
		}
	}
	cfg.WorkloadCollectorConfig = spec.WorkloadCollectorConfig
	return key, cfg
}

// ownerRefView is used to unmarshal a single entry from metadata.ownerReferences.
type ownerRefView struct {
	Controller *bool  `json:"controller,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
}

func workloadKeyFromOwnerRef(u *unstructured.Unstructured) string {
	namespace, _, _ := unstructured.NestedString(u.Object, "metadata", "namespace")
	ownerRefs, ok, _ := unstructured.NestedSlice(u.Object, "metadata", "ownerReferences")
	if !ok || len(ownerRefs) == 0 {
		return ""
	}
	for _, r := range ownerRefs {
		refMap, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		var ref ownerRefView
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(refMap, &ref); err != nil {
			continue
		}
		if ref.Controller == nil || !*ref.Controller || ref.Kind == "" || ref.Name == "" {
			continue
		}
		return WorkloadCacheKey(namespace, ref.Kind, ref.Name)
	}
	return ""
}
