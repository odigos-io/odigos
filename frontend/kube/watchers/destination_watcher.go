package watchers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services/sse"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	toolsWatch "k8s.io/client-go/tools/watch"
)

var destinationModifiedEventBatcher *EventBatcher

func StartDestinationWatcher(ctx context.Context, namespace string) error {
	destinationModifiedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			MinBatchSize: 2,
			Duration:     3 * time.Second,
			Event:        sse.MessageEventModified,
			CRDType:      consts.Destination,
			SuccessBatchMessageFunc: func(batchSize int, crd string) string {
				return fmt.Sprintf("Successfully updated %d destinations", batchSize)
			},
			FailureBatchMessageFunc: func(batchSize int, crd string) string {
				return fmt.Sprintf("Failed to update %d destinations", batchSize)
			},
		},
	)

	watcher, err := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: func(_ metav1.ListOptions) (watch.Interface, error) {
		return kube.DefaultClient.OdigosClient.Destinations(namespace).Watch(ctx, metav1.ListOptions{})
	}})
	if err != nil {
		return fmt.Errorf("error creating destinations watcher: %v", err)
	}

	go handleDestinationWatchEvents(ctx, watcher)
	return nil
}

func handleDestinationWatchEvents(ctx context.Context, watcher watch.Interface) {
	ch := watcher.ResultChan()
	defer destinationModifiedEventBatcher.Cancel()
	for {
		select {
		case <-ctx.Done():
			watcher.Stop()
			return
		case event, ok := <-ch:
			if !ok {
				log.Println("Destination watcher closed")
				return
			}
			switch event.Type {
			case watch.Modified:
				handleModifiedDestination(event.Object.(*v1alpha1.Destination))
			}
		}
	}
}

func handleModifiedDestination(destination *v1alpha1.Destination) {
	length := len(destination.Status.Conditions)
	if length == 0 {
		return
	}

	target := destination.Name
	data := fmt.Sprintf(`Successfully updated "%s" destination`, destination.Spec.Type)

	destinationModifiedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
}
