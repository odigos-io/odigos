package kube

import (
	"context"
	"fmt"
	"log"

	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
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

	nsSelector := client.InNamespace(odigosNs).AsSelector()
	// Create cache options
	cacheOptions := cache.Options{
		Scheme: scheme,
		ByObject: map[client.Object]cache.ByObject{
			&corev1.ConfigMap{}: {
				Field: nsSelector, // odigos effective config, collector configs, odigos deployment etc
			},
			&odigosv1.Source{}:                {},
			&odigosv1.InstrumentationConfig{}: {},
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
