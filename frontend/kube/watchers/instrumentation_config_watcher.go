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
	commonutils "github.com/odigos-io/odigos/k8sutils/pkg/workload"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var instrumentationConfigAddedEventBatcher *EventBatcher
var instrumentationConfigModifiedEventBatcher *EventBatcher
var instrumentationConfigDeletedEventBatcher *EventBatcher

// RunInstrumentationConfigWatcher runs the instrumentation config watcher in a reconnection loop.
// It should be launched as a goroutine. It only returns when ctx is cancelled.
func RunInstrumentationConfigWatcher(ctx context.Context, namespace string) {
	instrumentationConfigAddedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			MinBatchSize: 1,
			Duration:     3 * time.Second, // 2s less than frontend EVENT_DEBOUNCE_MS
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

	instrumentationConfigModifiedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			MinBatchSize: 1,
			Duration:     3 * time.Second, // 2s less than frontend EVENT_DEBOUNCE_MS
			Event:        sse.MessageEventModified,
			CRDType:      consts.InstrumentationConfig,
			Debounce:     true, // Reset timer on each event, send only after `Duration` seconds of silence
			SuccessBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Successfully updated %d sources", count)
			},
			FailureBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Failed to update %d sources", count)
			},
		},
	)

	instrumentationConfigDeletedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			MinBatchSize: 1,
			Duration:     3 * time.Second, // 2s less than frontend EVENT_DEBOUNCE_MS
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

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		watcher, err := StartRetryWatcher(ctx, WatcherConfig[*v1alpha1.InstrumentationConfigList]{
			ListFunc: func(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.InstrumentationConfigList, error) {
				return kube.DefaultClient.OdigosClient.InstrumentationConfigs(namespace).List(ctx, opts)
			},
			WatchFunc: func(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
				return kube.DefaultClient.OdigosClient.InstrumentationConfigs(namespace).Watch(ctx, opts)
			},
			GetResourceVersion: func(list *v1alpha1.InstrumentationConfigList) string {
				return list.ResourceVersion
			},
			ResourceName: "instrumentation configs",
		})
		if err != nil {
			log.Printf("Failed to start instrumentation config watcher: %v, retrying in 5s", err)
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return
			}
			continue
		}

		processInstrumentationConfigWatchEvents(ctx, watcher)
		log.Println("InstrumentationConfig watcher disconnected, reconnecting...")
	}
}

func processInstrumentationConfigWatchEvents(ctx context.Context, watcher watch.Interface) {
	ch := watcher.ResultChan()
	defer instrumentationConfigAddedEventBatcher.Cancel()
	defer instrumentationConfigModifiedEventBatcher.Cancel()
	defer instrumentationConfigDeletedEventBatcher.Cancel()
	for {
		select {
		case <-ctx.Done():
			watcher.Stop()
			return
		case event, ok := <-ch:
			if !ok {
				log.Println("InstrumentationConfig watcher channel closed")
				return
			}
			switch event.Type {
			case watch.Added:
				handleAddedInstrumentationConfig(event.Object.(*v1alpha1.InstrumentationConfig))
			case watch.Modified:
				handleModifiedInstrumentationConfig(event.Object.(*v1alpha1.InstrumentationConfig))
			case watch.Deleted:
				handleDeletedInstrumentationConfig(event.Object.(*v1alpha1.InstrumentationConfig))
			}
		}
	}
}

func handleAddedInstrumentationConfig(instruConfig *v1alpha1.InstrumentationConfig) {
	pw, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(instruConfig.Name, instruConfig.Namespace)
	if err != nil {
		genericErrorMessage(sse.MessageEventAdded, consts.InstrumentationConfig, err.Error())
		return
	}

	target := fmt.Sprintf("namespace=%s&name=%s&kind=%s", pw.Namespace, pw.Name, pw.Kind)
	data := fmt.Sprintf(`Successfully created "%s" source`, pw.Name)
	instrumentationConfigAddedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
}

func handleModifiedInstrumentationConfig(instruConfig *v1alpha1.InstrumentationConfig) {
	pw, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(instruConfig.Name, instruConfig.Namespace)
	if err != nil {
		genericErrorMessage(sse.MessageEventModified, consts.InstrumentationConfig, err.Error())
		return
	}

	// Use source identifier as target for deduplication (same source = counted once)
	target := fmt.Sprintf("%s/%s/%s", pw.Namespace, pw.Kind, pw.Name)
	data := fmt.Sprintf(`Successfully updated "%s" source`, pw.Name)
	instrumentationConfigModifiedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
}

func handleDeletedInstrumentationConfig(instruConfig *v1alpha1.InstrumentationConfig) {
	pw, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(instruConfig.Name, instruConfig.Namespace)
	if err != nil {
		genericErrorMessage(sse.MessageEventDeleted, consts.InstrumentationConfig, err.Error())
		return
	}

	target := fmt.Sprintf("namespace=%s&name=%s&kind=%s", pw.Namespace, pw.Name, pw.Kind)
	data := fmt.Sprintf(`Successfully deleted "%s" source`, pw.Name)
	instrumentationConfigDeletedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
}
