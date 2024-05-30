package watchers

import (
	"context"
	"fmt"
	"log"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/endpoints/sse"
	"github.com/odigos-io/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func StartDestinationWatcher(namespace string) error {
	watcher, err := kube.DefaultClient.OdigosClient.Destinations(namespace).Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error creating watcher: %v", err)
	}

	go handleDestinationWatchEvents(watcher)
	return nil
}

func handleDestinationWatchEvents(watcher watch.Interface) {
	ch := watcher.ResultChan()
	for event := range ch {
		switch event.Type {
		case watch.Added:
			handleAddedDestination(event)
		case watch.Modified:
			handleModifiedDestination(event)
		case watch.Deleted:
			handleDeletedDestination(event)
		default:
			log.Printf("unexpected type: %T", event.Object)
		}
	}
}

func handleAddedDestination(event watch.Event) {
	destination, ok := event.Object.(*v1alpha1.Destination)
	if !ok {
		genericErrorMessage(sse.MessageEventAdded, "Destination", "error type assertion")
	}
	data := fmt.Sprintf("Destination %s created", destination.Spec.DestinationName)
	sse.SendMessageToClient(sse.SSEMessage{Event: sse.MessageEventAdded, Type: "success", Target: destination.Name, Data: data, CRDType: "Destination"})
}

func handleModifiedDestination(event watch.Event) {
	destination, ok := event.Object.(*v1alpha1.Destination)
	if !ok {
		genericErrorMessage(sse.MessageEventModified, "Destination", "error type assertion")
	}
	if len(destination.Status.Conditions) == 0 {
		return
	}

	lastCondition := destination.Status.Conditions[len(destination.Status.Conditions)-1]
	data := lastCondition.Message
	conditionType := sse.MessageTypeSuccess
	if lastCondition.Status == "False" {
		conditionType = sse.MessageTypeError
	}
	sse.SendMessageToClient(sse.SSEMessage{Event: sse.MessageEventModified, Type: conditionType, Target: destination.Name, Data: data, CRDType: "Destination"})
}

func handleDeletedDestination(event watch.Event) {
	destination, ok := event.Object.(*v1alpha1.Destination)
	if !ok {
		genericErrorMessage(sse.MessageEventDeleted, "Destination", "error type assertion")
	}
	data := fmt.Sprintf("Destination %s deleted successfully", destination.Spec.DestinationName)
	sse.SendMessageToClient(sse.SSEMessage{Event: sse.MessageEventDeleted, Type: "success", Target: "", Data: data, CRDType: "Destination"})
}
