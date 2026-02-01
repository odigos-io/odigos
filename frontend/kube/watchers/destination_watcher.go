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
)

var destinationAddedEventBatcher *EventBatcher
var destinationModifiedEventBatcher *EventBatcher
var destinationDeletedEventBatcher *EventBatcher

func StartDestinationWatcher(ctx context.Context, namespace string) error {
	destinationAddedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			MinBatchSize: 1,
			Duration:     1 * time.Second,
			Event:        sse.MessageEventAdded,
			CRDType:      consts.Destination,
			SuccessBatchMessageFunc: func(batchSize int, crd string) string {
				return fmt.Sprintf("Successfully created %d destinations", batchSize)
			},
			FailureBatchMessageFunc: func(batchSize int, crd string) string {
				return fmt.Sprintf("Failed to create %d destinations", batchSize)
			},
		},
	)

	destinationModifiedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			MinBatchSize: 1,
			Duration:     1 * time.Second,
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

	destinationDeletedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			MinBatchSize: 1,
			Duration:     1 * time.Second,
			Event:        sse.MessageEventDeleted,
			CRDType:      consts.Destination,
			SuccessBatchMessageFunc: func(batchSize int, crd string) string {
				return fmt.Sprintf("Successfully deleted %d destinations", batchSize)
			},
			FailureBatchMessageFunc: func(batchSize int, crd string) string {
				return fmt.Sprintf("Failed to delete %d destinations", batchSize)
			},
		},
	)

	watcher, err := StartRetryWatcher(ctx, WatcherConfig[*v1alpha1.DestinationList]{
		ListFunc: func(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.DestinationList, error) {
			return kube.DefaultClient.OdigosClient.Destinations(namespace).List(ctx, opts)
		},
		WatchFunc: func(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
			return kube.DefaultClient.OdigosClient.Destinations(namespace).Watch(ctx, opts)
		},
		GetResourceVersion: func(list *v1alpha1.DestinationList) string {
			return list.ResourceVersion
		},
		ResourceName: "destinations",
	})
	if err != nil {
		return err
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
			case watch.Added:
				handleAddedDestination(event.Object.(*v1alpha1.Destination))
			case watch.Modified:
				handleModifiedDestination(event.Object.(*v1alpha1.Destination))
			case watch.Deleted:
				handleDeletedDestination(event.Object.(*v1alpha1.Destination))
			}
		}
	}
}

func handleAddedDestination(destination *v1alpha1.Destination) {
	target := destination.Name
	data := fmt.Sprintf(`Successfully created "%s" destination`, destination.Spec.Type)

	destinationAddedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
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

func handleDeletedDestination(destination *v1alpha1.Destination) {
	target := destination.Name
	data := fmt.Sprintf(`Successfully deleted "%s" destination`, destination.Spec.Type)

	destinationDeletedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
}
