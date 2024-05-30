package watchers

import (
	"context"
	"fmt"
	"log"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/endpoints/sse"
	"github.com/odigos-io/odigos/frontend/kube"
	commonutils "github.com/odigos-io/odigos/k8sutils/pkg/workload"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func StartInstrumentationInstanceWatcher(namespace string) error {
	watcher, err := kube.DefaultClient.OdigosClient.InstrumentationInstances(namespace).Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error creating watcher: %v", err)
	}

	go handleInstrumentationInstanceWatchEvents(watcher)
	return nil
}

func handleInstrumentationInstanceWatchEvents(watcher watch.Interface) {
	ch := watcher.ResultChan()
	for event := range ch {
		switch event.Type {
			// We only care about Modified events currently
			// This Go syntax might be a bit confusing, but it's just a way to ignore the other cases
		case watch.Deleted:
		case watch.Added:
		case watch.Modified:
			handleModifiedInstrumentationInstance(event)
		default:
			log.Printf("unexpected type for object %T go event type %v", event.Object, event.Type)
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

	name, kind, err := commonutils.GetWorkloadInfoRuntimeName(instrumentedAppName)
	if err != nil {
		genericErrorMessage(sse.MessageEventModified, "InstrumentationInstance", "error getting workload info")
	}

	namespace := instrumentedInstance.Namespace

	target := fmt.Sprintf("name=%s&kind=%s&namespace=%s", name, kind, namespace)
	data := fmt.Sprintf("%s %s", instrumentedInstance.Status.Reason, instrumentedInstance.Status.Message)

	sse.SendMessageToClient(sse.SSEMessage{Event: sse.MessageEventModified, Type: sse.MessageTypeError, Target: target, Data: data, CRDType: "InstrumentationInstance"})
}
