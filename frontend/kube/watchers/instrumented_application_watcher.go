package watchers

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/endpoints/sse"
	"github.com/odigos-io/odigos/frontend/kube"
	commonutils "github.com/odigos-io/odigos/k8sutils/pkg/workload"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var addedEventBatcher *EventBatcher
var deletedEventBatcher *EventBatcher

func StartInstrumentedApplicationWatcher(ctx context.Context, namespace string) error {
	addedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			Event:   sse.MessageEventAdded,
			CRDType: "InstrumentedApplication",
			SuccessBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("successfully added %d sources", count)
			},
			FailureBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("failed to add %d sources", count)
			},
		},
	)

	deletedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			Event:   sse.MessageEventDeleted,
			CRDType: "InstrumentedApplication",
			SuccessBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("successfully deleted %d sources", count)
			},
			FailureBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("failed to delete %d sources", count)
			},
		},
	)

	watcher, err := kube.DefaultClient.OdigosClient.InstrumentedApplications(namespace).Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error creating watcher: %v", err)
	}

	go handleInstrumentedApplicationWatchEvents(ctx, watcher)
	return nil
}

func handleInstrumentedApplicationWatchEvents(ctx context.Context, watcher watch.Interface) {
	ch := watcher.ResultChan()
	defer addedEventBatcher.Cancel()
	defer deletedEventBatcher.Cancel()
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
				handleAddedEvent(event.Object.(*v1alpha1.InstrumentedApplication))
			case watch.Deleted:
				handleDeletedEvent(event.Object.(*v1alpha1.InstrumentedApplication))
			}
		}
	}
}

func handleAddedEvent(app *v1alpha1.InstrumentedApplication) {
	name, kind, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(app.Name)
	if err != nil {
		genericErrorMessage(sse.MessageEventAdded, "InstrumentedApplication", "error getting workload info")
		return
	}
	namespace := app.Namespace
	target := fmt.Sprintf("name=%s&kind=%s&namespace=%s", name, kind, namespace)
	data := fmt.Sprintf("InstrumentedApplication %s created", name)
	addedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
}

func handleDeletedEvent(app *v1alpha1.InstrumentedApplication) {
	name, _, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(app.Name)
	if err != nil {
		genericErrorMessage(sse.MessageEventDeleted, "InstrumentedApplication", "error getting workload info")
		return
	}
	data := fmt.Sprintf("Source %s deleted successfully", name)
	deletedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, "")
}
