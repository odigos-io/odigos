package collectormetrics

import (
	"context"
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services/common"
	commonutils "github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	toolsWatch "k8s.io/client-go/tools/watch"
)

type notification struct {
	notificationType deletedObject
	eventType        watch.EventType
	object           string

	// used for source deletion notification
	sourceID common.SourceID
}

type deleteWatcher struct {
	odigosNS            string
	deleteNotifications chan notification
}

type deletedObject int

const (
	nodeCollector deletedObject = iota
	clusterCollector
	destination
	source
)

type watcherInterfaces struct {
	nodeCollectors, clusterCollectors, destinations, sources watch.Interface
}

func runWatcher(ctx context.Context, cw *deleteWatcher) error {
	nodeCollectorWatcher, err := toolsWatch.NewRetryWatcherWithContext(ctx, "1", &cache.ListWatch{WatchFunc: func(_ metav1.ListOptions) (watch.Interface, error) {
		return newCollectorWatcher(ctx, cw.odigosNS, k8sconsts.CollectorsRoleNodeCollector)
	}})
	if err != nil {
		return err
	}

	clusterCollectorWatcher, err := toolsWatch.NewRetryWatcherWithContext(ctx, "1", &cache.ListWatch{WatchFunc: func(_ metav1.ListOptions) (watch.Interface, error) {
		return newCollectorWatcher(ctx, cw.odigosNS, k8sconsts.CollectorsRoleClusterGateway)
	}})
	if err != nil {
		return err
	}

	sourcesWatcher, err := toolsWatch.NewRetryWatcherWithContext(ctx, "1", &cache.ListWatch{WatchFunc: func(_ metav1.ListOptions) (watch.Interface, error) {
		return kube.DefaultClient.OdigosClient.InstrumentationConfigs("").Watch(ctx, metav1.ListOptions{})
	}})
	if err != nil {
		return err
	}

	destsWatcher, err := toolsWatch.NewRetryWatcherWithContext(ctx, "1", &cache.ListWatch{WatchFunc: func(_ metav1.ListOptions) (watch.Interface, error) {
		return kube.DefaultClient.OdigosClient.Destinations(cw.odigosNS).Watch(ctx, metav1.ListOptions{})
	}})
	if err != nil {
		return err
	}

	return runWatcherLoop(ctx,
		watcherInterfaces{
			nodeCollectors:    nodeCollectorWatcher,
			clusterCollectors: clusterCollectorWatcher,
			destinations:      destsWatcher,
			sources:           sourcesWatcher,
		}, cw.deleteNotifications)
}

func newCollectorWatcher(ctx context.Context, odigosNS string, collectorRole k8sconsts.CollectorRole) (watch.Interface, error) {
	return kube.DefaultClient.CoreV1().Pods(odigosNS).Watch(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				k8sconsts.OdigosCollectorRoleLabel: string(collectorRole),
			},
		}),
	})
}

func runWatcherLoop(ctx context.Context, w watcherInterfaces, notifyChan chan<- notification) error {
	nch := w.nodeCollectors.ResultChan()
	cch := w.clusterCollectors.ResultChan()
	dch := w.destinations.ResultChan()
	sch := w.sources.ResultChan()
	for {
		select {
		case <-ctx.Done():
			w.nodeCollectors.Stop()
			w.clusterCollectors.Stop()
			w.destinations.Stop()
			w.sources.Stop()
			close(notifyChan)
			return nil
		case event, ok := <-nch:
			if !ok {
				return errors.New("node collector watcher closed")
			}
			switch event.Type {
			case watch.Deleted:
				pod := event.Object.(*corev1.Pod)
				notifyChan <- notification{notificationType: nodeCollector, object: pod.Name}
			}
		case event, ok := <-cch:
			if !ok {
				return errors.New("cluster collector watcher closed")
			}
			switch event.Type {
			case watch.Deleted:
				pod := event.Object.(*corev1.Pod)
				notifyChan <- notification{notificationType: clusterCollector, object: pod.Name, eventType: watch.Deleted}
			}
		case event, ok := <-dch:
			if !ok {
				return errors.New("destination watcher closed")
			}
			switch event.Type {
			case watch.Deleted:
				d := event.Object.(*v1alpha1.Destination)
				notifyChan <- notification{notificationType: destination, object: d.Name, eventType: watch.Deleted}
			}
		case event, ok := <-sch:
			if !ok {
				return errors.New("source watcher closed")
			}
			t := event.Type
			switch t {
			case watch.Deleted, watch.Added:
				app := event.Object.(*v1alpha1.InstrumentationConfig)
				pw, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(app.Name, app.Namespace)
				if err != nil {
					fmt.Printf("error getting workload info: %v\n", err)
				}
				notifyChan <- notification{notificationType: source, sourceID: common.SourceID{Kind: pw.Kind, Name: pw.Name, Namespace: pw.Namespace}, eventType: t}
			}
		}
	}
}
