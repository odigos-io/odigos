package kube

import (
	"context"
	"fmt"
	"log"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/endpoints/sse"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// StartDestinationWatcher starts watching Destination resources in the specified namespace.
func StartDestinationWatcher(namespace string) error {
	watcher, err := DefaultClient.OdigosClient.Destinations(namespace).Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error creating watcher: %v", err)
	}

	go handleDestinationWatchEvents(watcher)
	return nil
}

// handleDestinationWatchEvents processes events from the Destination watcher.
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
		case watch.Error:
			handleErrorDestination(event)
		}
	}
}

func handleAddedDestination(event watch.Event) {
	destination, ok := event.Object.(*v1alpha1.Destination)
	if !ok {
		log.Printf("unexpected type: %T", event.Object)
		return
	}
	target := destination.Name
	fmt.Printf("New destination added: %s\n", target)
	data := fmt.Sprintf("Destination %s created", destination.Spec.DestinationName)
	sse.SendMessageToClient(sse.SSEMessage{Event: "Created", Type: "success", Target: target, Data: data, CRDType: "Destination"})
}

func handleModifiedDestination(event watch.Event) {
	destination, ok := event.Object.(*v1alpha1.Destination)
	if !ok {
		log.Printf("unexpected type: %T", event.Object)
		return
	}
	fmt.Printf("Destination modified: %s\n", destination.Name)
	conditions := destination.Status.Conditions
	if len(conditions) == 0 {
		return
	}

	lastCondition := conditions[len(conditions)-1]
	data := lastCondition.Message
	target := destination.Name
	conditionType := "success"
	if lastCondition.Status == "False" {
		conditionType = "error"
	}

	sse.SendMessageToClient(sse.SSEMessage{Event: "Modified", Type: conditionType, Target: target, Data: data, CRDType: "Destination"})
}

func handleDeletedDestination(event watch.Event) {
	destination, ok := event.Object.(*v1alpha1.Destination)
	if !ok {
		log.Printf("unexpected type: %T", event.Object)
		return
	}
	fmt.Printf("Destination deleted: %s\n", destination.Name)
	data := fmt.Sprintf("Destination %s deleted successfully", destination.Spec.DestinationName)
	sse.SendMessageToClient(sse.SSEMessage{Event: "Deleted", Type: "success", Target: "", Data: data, CRDType: "Destination"})
}

func handleErrorDestination(event watch.Event) {
	destination, ok := event.Object.(*v1alpha1.Destination)
	if !ok {
		log.Printf("unexpected type: %T", event.Object)
		return
	}
	fmt.Printf("Error watching destination: %v\n", event.Object)
	data := fmt.Sprintf("Error watching Destination %s", destination.Name)
	target := destination.Name
	sse.SendMessageToClient(sse.SSEMessage{Event: "Error", Type: "error", Target: target, Data: data, CRDType: "Destination"})
}
