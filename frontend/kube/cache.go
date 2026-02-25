package kube

import (
	"context"
	"fmt"
	"log"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var CacheClient client.Client

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
	utilruntime.Must(argorolloutsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(openshiftappsv1.AddToScheme(scheme))

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
			&appsv1.Deployment{}: {
				Transform: deploymentsTransformFunc,
			},
			&appsv1.DaemonSet{}: {
				Transform: daemonsetsTransformFunc,
			},
			&appsv1.StatefulSet{}: {
				Transform: statefulsetsTransformFunc,
			},
			&batchv1.CronJob{}: {
				Transform: cronjobsTransformFunc,
			},
			&odigosv1.Source{}:                  {},
			&odigosv1.InstrumentationConfig{}:   {},
			&odigosv1.InstrumentationInstance{}: {},
		},
	}

	// if argo rollout is available, add it to the cache as well
	if IsArgoRolloutAvailable {
		cacheOptions.ByObject[&argorolloutsv1alpha1.Rollout{}] = cache.ByObject{
			Transform: argoRolloutsTransformFunc,
		}
	}

	// if open shift deployment config is available, add it to the cache as well
	if IsOpenShiftDeploymentConfigAvailable {
		cacheOptions.ByObject[&openshiftappsv1.DeploymentConfig{}] = cache.ByObject{
			Transform: deploymentConfigsTransformFunc,
		}
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
			log.Printf("Error starting kubernetes cache: %v", err)
		}
	}()

	// Wait for cache to be synced
	if !k8sCache.WaitForCacheSync(ctx) {
		return nil, fmt.Errorf("failed to sync kubernetes cache")
	}

	CacheClient = k8sCacheClient

	log.Println("K8s cache initialized and synced successfully")
	return k8sCacheClient, nil
}
