package watchers

import (
	"context"
	"fmt"
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
			Duration:     5 * time.Second,
			Event:        sse.MessageEventAdded,
			CRDType:      consts.Destination,
			SuccessBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Successfully created %d destinations", count)
			},
			FailureBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Failed to create %d destinations", count)
			},
		},
	)

	destinationModifiedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			MinBatchSize: 1,
			Duration:     5 * time.Second,
			Event:        sse.MessageEventModified,
			CRDType:      consts.Destination,
			SuccessBatchMessageFunc: func(batchSize int, crd string) string {
				return fmt.Sprintf("Successfully transformed %d destinations to otelcol configuration", batchSize)
			},
			FailureBatchMessageFunc: func(batchSize int, crd string) string {
				return fmt.Sprintf("Failed to transform %d destinations to otelcol configuration", batchSize)
			},
		},
	)

	destinationDeletedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			MinBatchSize: 1,
			Duration:     5 * time.Second,
			Event:        sse.MessageEventDeleted,
			CRDType:      consts.Destination,
			SuccessBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Successfully deleted %d destinations", count)
			},
			FailureBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Failed to delete %d destinations", count)
			},
		},
	)

	watcher, err := kube.DefaultClient.OdigosClient.Destinations(namespace).Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error creating watcher: %v", err)
	}

	go handleDestinationWatchEvents(ctx, watcher)
	return nil
}

func handleDestinationWatchEvents(ctx context.Context, watcher watch.Interface) {
	ch := watcher.ResultChan()
	defer destinationAddedEventBatcher.Cancel()
	defer destinationModifiedEventBatcher.Cancel()
	defer destinationDeletedEventBatcher.Cancel()
	for {
		select {
		case <-ctx.Done():
			watcher.Stop()
			return
		case event, ok := <-ch:
			if !ok {
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
	name := destination.Spec.DestinationName
	if name == "" {
		name = string(destination.Spec.Type)
	}

	target := destination.Name
	data := fmt.Sprintf(`%s "%s" created`, consts.Destination, name)
	destinationAddedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
}

func handleModifiedDestination(destination *v1alpha1.Destination) {
	length := len(destination.Status.Conditions)
	if length == 0 {
		return
	}

	target := destination.Name
	lastCondition := destination.Status.Conditions[length-1]
	data := lastCondition.Message

	conditionType := sse.MessageTypeInfo
	if lastCondition.Status == metav1.ConditionTrue {
		conditionType = sse.MessageTypeSuccess
	} else if lastCondition.Status == metav1.ConditionFalse {
		conditionType = sse.MessageTypeError
	}

	destinationModifiedEventBatcher.AddEvent(conditionType, data, target)
}

func handleDeletedDestination(destination *v1alpha1.Destination) {
	name := destination.Spec.DestinationName
	if name == "" {
		name = string(destination.Spec.Type)
	}

	target := destination.Name
	data := fmt.Sprintf(`%s "%s" deleted`, consts.Destination, name)
	destinationDeletedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
}
