package watchers

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/endpoints/sse"
	"github.com/odigos-io/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func StartDestinationWatcher(ctx context.Context, namespace string) error {
	watcher, err := kube.DefaultClient.OdigosClient.Destinations(namespace).Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error creating watcher: %v", err)
	}

	go handleDestinationWatchEvents(ctx, watcher)
	return nil
}

func handleDestinationWatchEvents(ctx context.Context, watcher watch.Interface) {
	ch := watcher.ResultChan()
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

	data := fmt.Sprintf(`%s "%s" created`, consts.Destination, name)
	sse.SendMessageToClient(sse.SSEMessage{
		Type:    sse.MessageTypeSuccess,
		Event:   sse.MessageEventAdded,
		Data:    data,
		CRDType: consts.Destination,
		Target:  destination.Name,
	})
}

func handleModifiedDestination(destination *v1alpha1.Destination) {
	length := len(destination.Status.Conditions)
	if length == 0 {
		return
	}

	lastCondition := destination.Status.Conditions[length-1]
	data := lastCondition.Message

	conditionType := sse.MessageTypeInfo
	if lastCondition.Status == metav1.ConditionTrue {
		conditionType = sse.MessageTypeSuccess
	} else if lastCondition.Status == metav1.ConditionFalse {
		conditionType = sse.MessageTypeError
	}

	sse.SendMessageToClient(sse.SSEMessage{
		Type:    conditionType,
		Event:   sse.MessageEventModified,
		Data:    data,
		CRDType: consts.Destination,
		Target:  destination.Name,
	})
}

func handleDeletedDestination(destination *v1alpha1.Destination) {
	name := destination.Spec.DestinationName
	if name == "" {
		name = string(destination.Spec.Type)
	}

	data := fmt.Sprintf(`%s "%s" deleted`, consts.Destination, name)
	sse.SendMessageToClient(sse.SSEMessage{
		Type:    sse.MessageTypeSuccess,
		Event:   sse.MessageEventDeleted,
		Data:    data,
		CRDType: consts.Destination,
		Target:  destination.Name,
	})
}
