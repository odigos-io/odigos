package odigosconfigk8sextension

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
		o.logger.Info("informer received non-unstructured object", zap.String("type", fmt.Sprintf("%T", obj)))
		return
	}

	workloadKey, ok := workloadKeyFromObject(u)
	if !ok {
		o.logger.Info("failed to get workload key from instrumentation config", zap.String("namespace", workloadKey.Namespace), zap.String("kind", workloadKey.Kind), zap.String("name", workloadKey.Name))
		return
	}

	specMap, ok, _ := unstructured.NestedMap(u.Object, "spec")
	if !ok || len(specMap) == 0 {
		o.logger.Info("failed to get instrumentation config spec; clearing workload state", zap.String("namespace", workloadKey.Namespace), zap.String("kind", workloadKey.Kind), zap.String("name", workloadKey.Name))
		o.syncWorkloadToDesiredState(workloadKey, nil)
		return
	}

	workloadCollectorConfigSlice, ok, _ := unstructured.NestedSlice(specMap, "workloadCollectorConfig")
	if !ok || len(workloadCollectorConfigSlice) == 0 {
		o.syncWorkloadToDesiredState(workloadKey, nil)
		return
	}
	desired := o.parseWorkloadCollectorConfig(workloadKey, workloadCollectorConfigSlice)
	o.syncWorkloadToDesiredState(workloadKey, desired)
}

// handleInstrumentationConfigDelete is called when an IC is removed. Desired state for this workload is empty.
func (o *OdigosWorkloadConfig) handleInstrumentationConfigDelete(obj interface{}) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		if deleted, ok := obj.(k8scache.DeletedFinalStateUnknown); ok {
			o.handleInstrumentationConfigDelete(deleted.Obj)
		}
		return
	}
	workloadKey, ok := workloadKeyFromObject(u)
	if !ok {
		return
	}
	o.syncWorkloadToDesiredState(workloadKey, nil)
}

// containerEntry is a single container's cache key and config for the desired state.
type containerEntry struct {
	key string
	cfg *commonapi.ContainerCollectorConfig
}

// parseWorkloadCollectorConfig turns the IC's workloadCollectorConfig slice into a list of containerEntry.
// Invalid or empty-container entries are skipped.
func (o *OdigosWorkloadConfig) parseWorkloadCollectorConfig(workloadKey workloadKey, slice []interface{}) []containerEntry {
	var out []containerEntry
	for _, item := range slice {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			o.logger.Info("failed to get container collector config from workload collector config", zap.String("namespace", workloadKey.Namespace), zap.String("kind", workloadKey.Kind), zap.String("name", workloadKey.Name))
			continue
		}
		var c commonapi.ContainerCollectorConfig
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(itemMap, &c); err != nil {
			continue
		}
		if c.ContainerName == "" {
			// Lookup always uses container-specific key; workload-level "default" is not supported.
			o.logger.Debug("skipping container collector config with empty containerName", zap.String("namespace", workloadKey.Namespace), zap.String("kind", workloadKey.Kind), zap.String("name", workloadKey.Name))
			continue
		}
		key := k8sSourceKey(workloadKey.Namespace, workloadKey.Kind, workloadKey.Name, c.ContainerName)
		cCopy := c
		out = append(out, containerEntry{key: key, cfg: &cCopy})
	}
	return out
}

// syncWorkloadToDesiredState makes the extension cache match the desired state. The cache
// notifies the callback internally on each Set/Delete. desired == nil or empty means
// "no containers for this workload" (e.g. IC deleted or spec has no workloadCollectorConfig).
//
// Order: (1) apply new/updated entries — cache.Set (cache invokes OnSet); (2) remove stale
// entries — cache.Delete (cache invokes OnDeleteKey). No span sees a gap.
func (o *OdigosWorkloadConfig) syncWorkloadToDesiredState(workloadKey workloadKey, desired []containerEntry) {
	workloadKeyStr := WorkloadKeyString(workloadKey.Namespace, workloadKey.Kind, workloadKey.Name)
	keyPrefix := workloadKeyStr + "/"

	oldKeys := o.getKeysForPrefix(keyPrefix)
	newKeys := make(map[string]struct{}, len(desired))

	// 1) Apply new/updated entries first (cache.Set triggers OnSet inside cache).
	for _, e := range desired {
		o.cache.Set(e.key, e.cfg)
		o.addKeyToIndex(e.key)
		newKeys[e.key] = struct{}{}
	}

	// 2) Remove stale entries (cache.Delete triggers OnDeleteKey inside cache).
	var numRemoved int
	for _, k := range oldKeys {
		if _, inNew := newKeys[k]; !inNew {
			o.removeKeyFromIndex(k)
			o.cache.Delete(k)
			numRemoved++
		}
	}

	o.logger.Debug("synced workload to desired state", zap.String("workload", workloadKeyStr), zap.Int("desired", len(desired)), zap.Int("removed", numRemoved))
}

// workloadKeyFromObject returns a WorkloadKey from the InstrumentationConfig's metadata.
// The object name format is <workload-kind>-<workload-name> (e.g. deployment-myapp).
// kindFromInstrumentationConfigName is a local copy of the parsing logic from
// k8sutils/pkg/workload/runtimeobjects.ExtractWorkloadInfoFromRuntimeObjectName and
// workloadkinds.WorkloadKindFromLowerCase. It is duplicated here temporarily to avoid
// coupling the collector extension to k8sutils and the odigos api package.
func workloadKeyFromObject(u *unstructured.Unstructured) (workloadKey, bool) {
	namespace, _, _ := unstructured.NestedString(u.Object, "metadata", "namespace")
	runtimeObjectName, _, _ := unstructured.NestedString(u.Object, "metadata", "name")
	if namespace == "" || runtimeObjectName == "" {
		return workloadKey{}, false
	}
	parts := strings.SplitN(runtimeObjectName, "-", 2)
	if len(parts) != 2 {
		return workloadKey{}, false
	}
	kind := kindFromInstrumentationConfigName(parts[0])
	if kind == "" {
		return workloadKey{}, false
	}
	return workloadKey{Namespace: namespace, Kind: kind, Name: parts[1]}, true
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
