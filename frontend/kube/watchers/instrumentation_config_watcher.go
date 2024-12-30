package watchers

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
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
			MinBatchSize: 1,
			Duration:     10 * time.Second,
			Event:        sse.MessageEventAdded,
			CRDType:      consts.InstrumentationConfig,
			SuccessBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Successfully created %d sources", count)
			},
			FailureBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Failed to create %d sources", count)
			},
		},
	)

	deletedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			MinBatchSize: 1,
			Duration:     10 * time.Second,
			Event:        sse.MessageEventDeleted,
			CRDType:      consts.InstrumentationConfig,
			SuccessBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Successfully deleted %d sources", count)
			},
			FailureBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Failed to delete %d sources", count)
			},
		},
	)

	watcher, err := kube.DefaultClient.OdigosClient.InstrumentationConfigs(namespace).Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error creating watcher: %w", err)
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
		genericErrorMessage(sse.MessageEventAdded, consts.InstrumentationConfig, err.Error())
		return
	}

	target := fmt.Sprintf("namespace=%s&name=%s&kind=%s", namespace, name, kind)
	data := fmt.Sprintf(`Source "%s" created`, name)
	addedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
}

func handleDeletedEvent(instruConfig *v1alpha1.InstrumentationConfig) {
	namespace := instruConfig.Namespace
	name, kind, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(instruConfig.Name)
	if err != nil {
		genericErrorMessage(sse.MessageEventDeleted, consts.InstrumentationConfig, err.Error())
		return
	}

	target := fmt.Sprintf("namespace=%s&name=%s&kind=%s", namespace, name, kind)
	data := fmt.Sprintf(`Source "%s" deleted`, name)
	deletedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
}
