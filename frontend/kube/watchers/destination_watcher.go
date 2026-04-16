package watchers

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
	"github.com/odigos-io/odigos/frontend/services/sse"
	toolscache "k8s.io/client-go/tools/cache"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
)

var destinationAddedEventBatcher *EventBatcher
var destinationModifiedEventBatcher *EventBatcher
var destinationDeletedEventBatcher *EventBatcher

func StartDestinationWatcher(ctx context.Context, k8sCache ctrlcache.Cache, metricsConsumer *collectormetrics.OdigosMetricsConsumer) error {
	destinationAddedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			MinBatchSize: 1,
			Duration:     1 * time.Second,
			Event:        sse.MessageEventAdded,
			CRDType:      consts.Destination,
			MaxBatchSize: 100,
			MaxDelay:     10 * time.Second,
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
			MaxBatchSize: 100,
			MaxDelay:     10 * time.Second,
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
			MaxBatchSize: 100,
			MaxDelay:     10 * time.Second,
			SuccessBatchMessageFunc: func(batchSize int, crd string) string {
				return fmt.Sprintf("Successfully deleted %d destinations", batchSize)
			},
			FailureBatchMessageFunc: func(batchSize int, crd string) string {
				return fmt.Sprintf("Failed to delete %d destinations", batchSize)
			},
		},
	)

	informer, err := k8sCache.GetInformer(ctx, &v1alpha1.Destination{})
	if err != nil {
		return fmt.Errorf("failed to get Destination informer: %w", err)
	}

	_, err = informer.AddEventHandler(toolscache.ResourceEventHandlerDetailedFuncs{
		AddFunc: func(obj interface{}, isInInitialList bool) {
			if isInInitialList {
				return
			}
			dest, ok := obj.(*v1alpha1.Destination)
			if !ok {
				return
			}
			handleAddedDestination(dest)
		},
		UpdateFunc: func(_, newObj interface{}) {
			dest, ok := newObj.(*v1alpha1.Destination)
			if !ok {
				return
			}
			handleModifiedDestination(dest)
		},
		DeleteFunc: func(obj interface{}) {
			dest, ok := obj.(*v1alpha1.Destination)
			if !ok {
				tombstone, ok := obj.(toolscache.DeletedFinalStateUnknown)
				if !ok {
					return
				}
				dest, ok = tombstone.Obj.(*v1alpha1.Destination)
				if !ok {
					return
				}
			}

			metricsConsumer.NotifyDestinationDeleted(dest.Name)
			handleDeletedDestination(dest)
		},
	})

	return err
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
