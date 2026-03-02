package odigosworkloadconfigextension

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	commonapi "github.com/odigos-io/odigos/common/api"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	k8scache "k8s.io/client-go/tools/cache"
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
// It runs until ctx is cancelled. The cache is keyed by WorkloadKey (namespace, kind, name).
// If not running in a cluster (e.g. InClusterConfig fails), the informer is not started
// and the cache remains empty; the extension still starts successfully.
func (o *OdigosWorkloadConfig) startInformer(ctx context.Context) error {
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

	o.informerFactory = factory
	factory.Start(ctx.Done())
	// Do not call WaitForCacheSync here; Start() returns immediately so the collector
	// does not block. Dependent components call WaitForCacheSync themselves.
	return nil
}

// WaitForCacheSync blocks until the InstrumentationConfig informer cache has synced
// or ctx is done. It returns true if the cache synced successfully, false if the
// context was cancelled or the extension is not running in-cluster (in which case
// the cache is empty and callers may treat true as "ready"). Start() does not block
// on sync; components that depend on the cache should call WaitForCacheSync before
// relying on GetWorkloadSamplingConfig (e.g. in a goroutine so the collector stays
// non-blocking).
func (o *OdigosWorkloadConfig) WaitForCacheSync(ctx context.Context) bool {
	if o.informerFactory == nil {
		return true // not in-cluster; cache is empty, consider "ready"
	}
	synced := o.informerFactory.WaitForCacheSync(ctx.Done())
	if !synced[instrumentationConfigGVR] {
		o.logger.Warn("instrumentationconfig informer cache sync did not complete")
		return false
	}
	return true
}

func (o *OdigosWorkloadConfig) handleInstrumentationConfig(obj interface{}) {
	// We're currently using unstructured to avoid a dependency on the odigos api package.
	// The api package can bring in transitive dependencies that conflict with OTel upstream dependencies.
	// This is a temporary solution until we have a better way to handle the instrumentation config (ie, using our api directly ideally)
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		o.logger.Debug("informer received non-unstructured object", zap.String("type", fmt.Sprintf("%T", obj)))
		return
	}
	key, ok, cfg := instrumentationConfigToWorkloadSampling(u)
	if !ok {
		return
	}
	o.cache.Set(key, cfg)
	o.logger.Debug("updated workload sampling cache", zap.String("namespace", key.Namespace), zap.String("kind", key.Kind), zap.String("name", key.Name))
}

func (o *OdigosWorkloadConfig) handleInstrumentationConfigDelete(obj interface{}) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		if deleted, ok := obj.(k8scache.DeletedFinalStateUnknown); ok {
			o.handleInstrumentationConfigDelete(deleted.Obj)
		}
		return
	}
	key, ok := workloadKeyFromObject(u)
	if ok {
		o.cache.Delete(key)
		o.logger.Debug("removed workload from sampling cache", zap.String("namespace", key.Namespace), zap.String("kind", key.Kind), zap.String("name", key.Name))
	}
}

func instrumentationConfigToWorkloadSampling(u *unstructured.Unstructured) (key WorkloadKey, ok bool, cfg *WorkloadConfig) {
	key, ok = workloadKeyFromObject(u)
	if !ok {
		return key, false, nil
	}
	specMap, ok, _ := unstructured.NestedMap(u.Object, "spec")
	if !ok || len(specMap) == 0 {
		return key, true, &WorkloadConfig{}
	}
	workloadCollectorConfigSlice, ok, _ := unstructured.NestedSlice(specMap, "workloadCollectorConfig")
	if !ok || len(workloadCollectorConfigSlice) == 0 {
		return key, true, &WorkloadConfig{}
	}
	var workloadCollectorConfig []commonapi.ContainerCollectorConfig
	for _, item := range workloadCollectorConfigSlice {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		var c commonapi.ContainerCollectorConfig
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(itemMap, &c); err != nil {
			continue
		}
		workloadCollectorConfig = append(workloadCollectorConfig, c)
	}
	cfg = &WorkloadConfig{
		WorkloadCollectorConfig: workloadCollectorConfig,
	}
	return key, true, cfg
}

// workloadKeyFromObject returns a WorkloadKey from the InstrumentationConfig's metadata.
// The object name format is <workload-kind>-<workload-name> (e.g. deployment-myapp).
// kindFromInstrumentationConfigName is a local copy of the parsing logic from
// k8sutils/pkg/workload/runtimeobjects.ExtractWorkloadInfoFromRuntimeObjectName and
// workloadkinds.WorkloadKindFromLowerCase. It is duplicated here temporarily to avoid
// coupling the collector extension to k8sutils and the odigos api package.
func workloadKeyFromObject(u *unstructured.Unstructured) (WorkloadKey, bool) {
	namespace, _, _ := unstructured.NestedString(u.Object, "metadata", "namespace")
	runtimeObjectName, _, _ := unstructured.NestedString(u.Object, "metadata", "name")
	if namespace == "" || runtimeObjectName == "" {
		return WorkloadKey{}, false
	}
	parts := strings.SplitN(runtimeObjectName, "-", 2)
	if len(parts) != 2 {
		return WorkloadKey{}, false
	}
	kind := kindFromInstrumentationConfigName(parts[0])
	if kind == "" {
		return WorkloadKey{}, false
	}
	return WorkloadKey{Namespace: namespace, Kind: kind, Name: parts[1]}, true
}

// kindFromInstrumentationConfigName maps lowercase workload kind (from InstrumentationConfig
// name prefix) to PascalCase Kubernetes Kind. Mirrors k8sutils/pkg/workload/workloadkinds
// and api/k8sconsts.WorkloadKindLowerCase/WorkloadKind. Returns "" for unsupported kinds.
func kindFromInstrumentationConfigName(lowercase string) string {
	switch strings.ToLower(lowercase) {
	case "deployment":
		return "Deployment"
	case "daemonset":
		return "DaemonSet"
	case "statefulset":
		return "StatefulSet"
	case "namespace":
		return "Namespace"
	case "staticpod":
		return "StaticPod"
	case "cronjob":
		return "CronJob"
	case "job":
		return "Job"
	case "deploymentconfig":
		return "DeploymentConfig"
	case "rollout":
		return "Rollout"
	default:
		return ""
	}
}
