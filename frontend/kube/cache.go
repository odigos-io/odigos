package kube

import (
	"context"
	"fmt"
	"log"

	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var CacheClient client.Client

func podsTransformFunc(obj interface{}) (interface{}, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("expected a Pod, got %T", obj)
	}

	// Strip unnecessary fields to reduce memory usage.
	// Keep only fields needed for computing CachedPod in loader.go and status calculations.
	minimalContainers := make([]corev1.Container, len(pod.Spec.Containers))
	for i, c := range pod.Spec.Containers {
		relevantEnvVars := make([]corev1.EnvVar, 0, 1)
		for _, env := range c.Env {
			if env.Name == k8sconsts.OdigosEnvVarDistroName {
				relevantEnvVars = append(relevantEnvVars, env)
				break
			}
		}
		minimalContainers[i] = corev1.Container{
			Name:      c.Name,
			Env:       relevantEnvVars,
			Resources: c.Resources,
		}
	}

	// Only keep the specific label needed for agent injection status calculation
	var minimalLabels map[string]string
	if agentHashValue, exists := pod.Labels[k8sconsts.OdigosAgentsMetaHashLabel]; exists {
		minimalLabels = map[string]string{k8sconsts.OdigosAgentsMetaHashLabel: agentHashValue}
	}

	minimalPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         pod.Namespace,
			Name:              pod.Name,
			CreationTimestamp: pod.CreationTimestamp,
			Labels:            minimalLabels,
			OwnerReferences:   pod.OwnerReferences,
		},
		Spec: corev1.PodSpec{
			NodeName:   pod.Spec.NodeName,
			Containers: minimalContainers,
		},
		Status: corev1.PodStatus{
			ContainerStatuses:     pod.Status.ContainerStatuses,
			InitContainerStatuses: pod.Status.InitContainerStatuses,
		},
	}

	return minimalPod, nil
}

func deploymentsTransformFunc(obj interface{}) (interface{}, error) {
	dep, ok := obj.(*appsv1.Deployment)
	if !ok {
		return nil, fmt.Errorf("expected a Deployment, got %T", obj)
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dep.Name,
			Namespace: dep.Namespace,
			UID:       dep.UID,
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     dep.Status.ReadyReplicas,
			AvailableReplicas: dep.Status.AvailableReplicas,
		},
	}, nil
}

func statefulSetsTransformFunc(obj interface{}) (interface{}, error) {
	ss, ok := obj.(*appsv1.StatefulSet)
	if !ok {
		return nil, fmt.Errorf("expected a StatefulSet, got %T", obj)
	}
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ss.Name,
			Namespace: ss.Namespace,
			UID:       ss.UID,
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas: ss.Status.ReadyReplicas,
		},
	}, nil
}

func daemonSetsTransformFunc(obj interface{}) (interface{}, error) {
	ds, ok := obj.(*appsv1.DaemonSet)
	if !ok {
		return nil, fmt.Errorf("expected a DaemonSet, got %T", obj)
	}
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ds.Name,
			Namespace: ds.Namespace,
			UID:       ds.UID,
		},
		Status: appsv1.DaemonSetStatus{
			NumberReady: ds.Status.NumberReady,
		},
	}, nil
}

func cronJobsTransformFunc(obj interface{}) (interface{}, error) {
	cj, ok := obj.(*batchv1.CronJob)
	if !ok {
		return nil, fmt.Errorf("expected a CronJob, got %T", obj)
	}
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cj.Name,
			Namespace: cj.Namespace,
			UID:       cj.UID,
		},
		Status: batchv1.CronJobStatus{
			Active: cj.Status.Active,
		},
	}, nil
}

// SetupK8sCache initializes and starts the controller runtime cache for Source resources
// Returns the cache client for direct usage
func SetupK8sCache(ctx context.Context, kubeConfig string, kubeContext string, odigosNs string) (client.Client, error) {
	// Get the Kubernetes config
	cfg, err := config.GetConfigWithContext(kubeContext)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	// Override config if kubeConfig path is provided
	if kubeConfig != "" {
		cfg, err = config.GetConfigWithContext(kubeContext)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubernetes config with custom path: %w", err)
		}
	}

	// Create a new scheme and register all required types
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(odigosv1.AddToScheme(scheme))
	utilruntime.Must(actionsv1.AddToScheme(scheme))

	nsSelector := client.InNamespace(odigosNs).AsSelector()
	// Create cache options
	cacheOptions := cache.Options{
		Scheme:                      scheme,
		ReaderFailOnMissingInformer: true,
		ByObject: map[client.Object]cache.ByObject{
			&corev1.ConfigMap{}: {
				Field: nsSelector, // odigos effective config, collector configs, odigos deployment etc
			},
			&corev1.Pod{}: {
				Transform: podsTransformFunc,
			},
			&odigosv1.Source{}:                  {},
			&odigosv1.InstrumentationConfig{}:   {},
			&odigosv1.InstrumentationInstance{}: {},
			&appsv1.Deployment{}:                {Transform: deploymentsTransformFunc},
			&appsv1.StatefulSet{}:               {Transform: statefulSetsTransformFunc},
			&appsv1.DaemonSet{}:                 {Transform: daemonSetsTransformFunc},
			&batchv1.CronJob{}:                  {Transform: cronJobsTransformFunc},
		},
	}

	// Create the cache
	k8sCache, err := cache.New(cfg, cacheOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	// Create a client that uses the cache
	k8sCacheClient, err := client.New(cfg, client.Options{
		Scheme: scheme,
		Cache: &client.CacheOptions{
			Reader: k8sCache,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cache client: %w", err)
	}

	// Explicitly initialize informers for all configured resource types with selectors.
	// Controller-runtime cache uses lazy initialization - informers are created on-demand.
	// With ReaderFailOnMissingInformer: true, we must ensure informers exist before
	// WaitForCacheSync, otherwise cache operations will fail with "is not cached" errors.
	// This ensures all configured informers are ready before the cache sync completes.
	for obj, _ := range cacheOptions.ByObject {
		_, err = k8sCache.GetInformer(ctx, obj) // just need to call it to initialize the informer
		if err != nil {
			return nil, fmt.Errorf("failed to get informer for %s: %w", obj.GetObjectKind().GroupVersionKind().Kind, err)
		}
	}

	// Start the cache in a goroutine
	go func() {
		if err := k8sCache.Start(ctx); err != nil {
			log.Printf("Error starting source cache: %v", err)
		}
	}()

	// Wait for cache to be synced
	if !k8sCache.WaitForCacheSync(ctx) {
		return nil, fmt.Errorf("failed to sync source cache")
	}

	CacheClient = k8sCacheClient

	log.Println("K8s cache initialized and synced successfully")
	return k8sCacheClient, nil
}
