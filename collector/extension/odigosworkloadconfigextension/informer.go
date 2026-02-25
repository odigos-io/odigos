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

	factory.Start(ctx.Done())

	synced := factory.WaitForCacheSync(ctx.Done())
	if !synced[instrumentationConfigGVR] {
		o.logger.Warn("instrumentationconfig informer cache sync did not complete")
	}

	return nil
}

func (o *OdigosWorkloadConfig) handleInstrumentationConfig(obj interface{}) {
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

func instrumentationConfigToWorkloadSampling(u *unstructured.Unstructured) (key WorkloadKey, ok bool, cfg *WorkloadSamplingConfig) {
	key, ok = workloadKeyFromObject(u)
	if !ok {
		return key, false, nil
	}
	specMap, ok, _ := unstructured.NestedMap(u.Object, "spec")
	if !ok || len(specMap) == 0 {
		return key, true, &WorkloadSamplingConfig{}
	}
	workloadCollectorConfigSlice, ok, _ := unstructured.NestedSlice(specMap, "workloadCollectorConfig")
	if !ok || len(workloadCollectorConfigSlice) == 0 {
		return key, true, &WorkloadSamplingConfig{}
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
	cfg = &WorkloadSamplingConfig{
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
