package watchers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/endpoints/sse"
	"github.com/odigos-io/odigos/frontend/kube"
	commonutils "github.com/odigos-io/odigos/k8sutils/pkg/workload"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	batchDuration = 2 * time.Second // Time to wait before sending a batch of notifications
	minBatchSize  = 4               // Minimum number of notifications to batch
)

type EventBatcher struct {
	mu                 sync.Mutex
	addedEventCount    int
	deletedEventCount  int
	modifiedEventCount int
	addedEvents        []sse.SSEMessage
	deletedEvents      []sse.SSEMessage
	modifiedEvents     []sse.SSEMessage
	timer              *time.Timer
	batchDuration      time.Duration
}

func NewEventBatcher(duration time.Duration) *EventBatcher {
	return &EventBatcher{
		batchDuration: duration,
	}
}

func (eb *EventBatcher) AddEvent(eventType string, message sse.SSEMessage) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	switch eventType {
	case "Added":
		eb.addedEventCount++
		eb.addedEvents = append(eb.addedEvents, message)
	case "Deleted":
		eb.deletedEventCount++
		eb.deletedEvents = append(eb.deletedEvents, message)
	case "Modified":
		eb.modifiedEventCount++
		eb.modifiedEvents = append(eb.modifiedEvents, message)
	}

	if eb.timer == nil {
		eb.timer = time.AfterFunc(eb.batchDuration, eb.sendBatch)
	}
}

func (eb *EventBatcher) sendBatch() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.addedEventCount > 0 {
		if eb.addedEventCount < minBatchSize {
			for _, msg := range eb.addedEvents {
				sse.SendMessageToClient(msg)
			}
		} else {
			addedMessage := sse.SSEMessage{
				Event:   sse.MessageEventAdded,
				Type:    sse.MessageTypeSuccess,
				Target:  "",
				Data:    fmt.Sprintf("%d sources added successfully", eb.addedEventCount),
				CRDType: "InstrumentedApplication",
			}
			sse.SendMessageToClient(addedMessage)
		}
		eb.addedEventCount = 0
		eb.addedEvents = nil
	}

	if eb.deletedEventCount > 0 {
		if eb.deletedEventCount < minBatchSize {
			for _, msg := range eb.deletedEvents {
				sse.SendMessageToClient(msg)
			}
		} else {
			deletedMessage := sse.SSEMessage{
				Event:   sse.MessageEventDeleted,
				Type:    sse.MessageTypeSuccess,
				Target:  "",
				Data:    fmt.Sprintf("%d sources deleted successfully", eb.deletedEventCount),
				CRDType: "InstrumentedApplication",
			}
			sse.SendMessageToClient(deletedMessage)
		}
		eb.deletedEventCount = 0
		eb.deletedEvents = nil
	}

	if eb.modifiedEventCount > 0 {
		if eb.modifiedEventCount < minBatchSize {
			for _, msg := range eb.modifiedEvents {
				sse.SendMessageToClient(msg)
			}
		} else {
			modifiedMessage := sse.SSEMessage{
				Event:   sse.MessageEventModified,
				Type:    sse.MessageTypeSuccess,
				Target:  "",
				Data:    fmt.Sprintf("%d sources modified successfully", eb.modifiedEventCount),
				CRDType: "InstrumentedApplication",
			}
			sse.SendMessageToClient(modifiedMessage)
		}
		eb.modifiedEventCount = 0
		eb.modifiedEvents = nil
	}

	eb.timer = nil
}

var batcher = NewEventBatcher(batchDuration)

func StartInstrumentedApplicationWatcher(namespace string) error {
	watcher, err := kube.DefaultClient.OdigosClient.InstrumentedApplications(namespace).Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error creating watcher: %v", err)
	}

	go handleInstrumentedApplicationWatchEvents(watcher)
	return nil
}

func handleInstrumentedApplicationWatchEvents(watcher watch.Interface) {
	ch := watcher.ResultChan()
	for event := range ch {
		switch event.Type {
		case watch.Added:
			handleAddedEvent(event.Object.(*v1alpha1.InstrumentedApplication))
		// case watch.Modified:
		// 	handleModifiedEvent(event.Object.(*v1alpha1.InstrumentedApplication))
		case watch.Deleted:
			handleDeletedEvent(event.Object.(*v1alpha1.InstrumentedApplication))

		}
	}
}

func handleAddedEvent(app *v1alpha1.InstrumentedApplication) {
	name, kind, err := commonutils.GetWorkloadInfoRuntimeName(app.Name)
	if err != nil {
		genericErrorMessage(sse.MessageEventAdded, "InstrumentedApplication", "error getting workload info")
		return
	}
	namespace := app.Namespace
	target := fmt.Sprintf("name=%s&kind=%s&namespace=%s", name, kind, namespace)
	data := fmt.Sprintf("InstrumentedApplication %s created", name)
	message := sse.SSEMessage{
		Event:   sse.MessageEventAdded,
		Type:    sse.MessageTypeSuccess,
		Target:  target,
		Data:    data,
		CRDType: "InstrumentedApplication",
	}
	batcher.AddEvent("Added", message)
}

func handleModifiedEvent(app *v1alpha1.InstrumentedApplication) {
	conditions := app.Status.Conditions
	if len(conditions) == 0 {
		return
	}
	name, kind, err := commonutils.GetWorkloadInfoRuntimeName(app.Name)
	if err != nil {
		return
	}

	lastCondition := conditions[len(conditions)-1]
	data := lastCondition.Message
	namespace := app.Namespace
	target := fmt.Sprintf("name=%s&kind=%s&namespace=%s", name, kind, namespace)
	conditionType := sse.MessageTypeSuccess
	if lastCondition.Status == "False" {
		conditionType = sse.MessageTypeError
	}

	message := sse.SSEMessage{
		Event:   "Modified",
		Type:    conditionType,
		Target:  target,
		Data:    data,
		CRDType: "InstrumentedApplication",
	}

	batcher.AddEvent("Modified", message)
}

func handleDeletedEvent(app *v1alpha1.InstrumentedApplication) {
	name, _, err := commonutils.GetWorkloadInfoRuntimeName(app.Name)
	if err != nil {
		genericErrorMessage(sse.MessageEventDeleted, "InstrumentedApplication", "error getting workload info")
		return
	}
	data := fmt.Sprintf("Source %s deleted successfully", name)
	message := sse.SSEMessage{
		Event:   sse.MessageEventDeleted,
		Type:    sse.MessageTypeSuccess,
		Target:  "",
		Data:    data,
		CRDType: "InstrumentedApplication",
	}
	batcher.AddEvent("Deleted", message)
}
