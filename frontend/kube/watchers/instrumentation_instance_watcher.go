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

var modifiedBatcher *EventBatcher

func StartInstrumentationInstanceWatcher(ctx context.Context, namespace string) error {
	modifiedBatcher = NewEventBatcher(
		EventBatcherConfig{
			Event:       sse.MessageEventModified,
			MessageType: sse.MessageTypeError,
			Duration:    10 * time.Second,
			CRDType:     "InstrumentationInstance",
			FailureBatchMessageFunc: func(batchSize int, crd string) string {
				return fmt.Sprintf("Failed to instrument %d instances", batchSize)
			},
		},
	)
	watcher, err := kube.DefaultClient.OdigosClient.InstrumentationInstances(namespace).Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error creating watcher: %v", err)
	}

	go handleInstrumentationInstanceWatchEvents(ctx, watcher)
	return nil
}

func handleInstrumentationInstanceWatchEvents(ctx context.Context, watcher watch.Interface) {
	ch := watcher.ResultChan()
	defer modifiedBatcher.Cancel()
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
			case watch.Modified:
				handleModifiedInstrumentationInstance(event)
			}
		}
	}
}

func handleModifiedInstrumentationInstance(event watch.Event) {
	instrumentedInstance, ok := event.Object.(*v1alpha1.InstrumentationInstance)
	if !ok {
		genericErrorMessage(sse.MessageEventModified, "InstrumentationInstance", "error type assertion")
	}
	healthy := instrumentedInstance.Status.Healthy

	if healthy == nil {
		return
	}

	if *healthy {
		// send notification to frontend only if the instance is not healthy
		return
	}

	labels := instrumentedInstance.GetLabels()
	if labels == nil {
		genericErrorMessage(sse.MessageEventModified, "InstrumentationInstance", "error getting labels")
	}

	instrumentedAppName, ok := labels[consts.InstrumentedAppNameLabel]
	if !ok {
		genericErrorMessage(sse.MessageEventModified, "InstrumentationInstance", "error getting instrumented app name from labels")
	}

	name, kind, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(instrumentedAppName)
	if err != nil {
		genericErrorMessage(sse.MessageEventModified, "InstrumentationInstance", "error getting workload info")
	}

	namespace := instrumentedInstance.Namespace

	target := fmt.Sprintf("name=%s&kind=%s&namespace=%s", name, kind, namespace)
	data := fmt.Sprintf("%s %s", instrumentedInstance.Status.Reason, instrumentedInstance.Status.Message)

	fmt.Printf("InstrumentationInstance %s modified\n", name)
	modifiedBatcher.AddEvent(sse.MessageTypeError, data, target)
}
