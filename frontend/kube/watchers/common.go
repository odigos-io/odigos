package watchers

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/frontend/services/sse"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	toolsWatch "k8s.io/client-go/tools/watch"
)

func genericErrorMessage(event sse.MessageEvent, crd string, data string) {
	sse.SendMessageToClient(sse.SSEMessage{
		Type:    sse.MessageTypeError,
		Event:   event,
		Data:    "Something went wrong: " + data,
		CRDType: crd,
		Target:  "",
	})
}

// WatcherConfig holds the configuration for creating a retry watcher.
// L is the list type returned by the ListFunc (e.g., *v1alpha1.DestinationList).
type WatcherConfig[L any] struct {
	// ListFunc retrieves the current list of resources to get the initial resource version.
	ListFunc func(ctx context.Context, opts metav1.ListOptions) (L, error)
	// WatchFunc creates a watch on the resources.
	WatchFunc func(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	// GetResourceVersion extracts the resource version from the list result.
	GetResourceVersion func(L) string
	// LabelSelector is an optional label selector to filter resources.
	LabelSelector string
	// ResourceName is used for error messages (e.g., "destinations", "node collector pods").
	ResourceName string
}

// StartRetryWatcher creates a retry watcher that starts from the current resource version.
// This prevents old events from re-surfacing when the watcher is created.
func StartRetryWatcher[L any](ctx context.Context, cfg WatcherConfig[L]) (watch.Interface, error) {
	listOpts := metav1.ListOptions{}
	if cfg.LabelSelector != "" {
		listOpts.LabelSelector = cfg.LabelSelector
	}

	// List first to get the current resource version
	list, err := cfg.ListFunc(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", cfg.ResourceName, err)
	}

	resourceVersion := cfg.GetResourceVersion(list)

	// Create watcher with current resource version (prevents old events from re-surfacing)
	watcher, err := toolsWatch.NewRetryWatcherWithContext(ctx, resourceVersion, &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			if cfg.LabelSelector != "" {
				options.LabelSelector = cfg.LabelSelector
			}
			return cfg.WatchFunc(ctx, options)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create %s watcher: %w", cfg.ResourceName, err)
	}

	return watcher, nil
}
