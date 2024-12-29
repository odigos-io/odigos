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

func StartInstrumentationConfigWatcher(ctx context.Context, namespace string) error {
	addedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			Event:   sse.MessageEventAdded,
			CRDType: "InstrumentationConfig",
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
			CRDType: "InstrumentationConfig",
			SuccessBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("successfully deleted %d sources", count)
			},
			FailureBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("failed to delete %d sources", count)
			},
		},
	)

	watcher, err := kube.DefaultClient.OdigosClient.InstrumentationConfigs(namespace).Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error creating watcher: %v", err)
	}

	go handleInstrumentationConfigWatchEvents(ctx, watcher)
	return nil
}

func handleInstrumentationConfigWatchEvents(ctx context.Context, watcher watch.Interface) {
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
				handleAddedEvent(event.Object.(*v1alpha1.InstrumentationConfig))
			case watch.Deleted:
				handleDeletedEvent(event.Object.(*v1alpha1.InstrumentationConfig))
			}
		}
	}
}

func handleAddedEvent(instruConfig *v1alpha1.InstrumentationConfig) {
	namespace := instruConfig.Namespace
	name, kind, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(instruConfig.Name)
	if err != nil {
		genericErrorMessage(sse.MessageEventAdded, "InstrumentationConfig", err.Error())
		return
	}

	target := fmt.Sprintf("namespace=%s&name=%s&kind=%s", namespace, name, kind)
	data := fmt.Sprintf("InstrumentationConfig %s created successfully", name)
	addedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
}

func handleDeletedEvent(instruConfig *v1alpha1.InstrumentationConfig) {
	namespace := instruConfig.Namespace
	name, kind, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(instruConfig.Name)
	if err != nil {
		genericErrorMessage(sse.MessageEventDeleted, "InstrumentationConfig", err.Error())
		return
	}

	target := fmt.Sprintf("namespace=%s&name=%s&kind=%s", namespace, name, kind)
	data := fmt.Sprintf("InstrumentationConfig %s deleted successfully", name)
	deletedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
}
