package collectormetrics

import (
	"context"
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/endpoints/common"
	"github.com/odigos-io/odigos/frontend/kube"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	commonutils "github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type deleteNotification struct {
	notificationType deletedObject
	object           string
	// used for source deletion notification
	sourceID common.SourceID
}

type deleteWatcher struct {
	odigosNS            string
	deleteNotifications chan deleteNotification
}

type deletedObject int

const (
	nodeCollector deletedObject = iota
	clusterCollector
	destination
	source
)

type watchers struct {
	nodeCollectors, clusterCollectors, destinations, sources watch.Interface
}

func runDeleteWatcher(ctx context.Context, cw *deleteWatcher) error {
	nodeWatcher, err := newCollectorWatcher(ctx, cw.odigosNS, k8sconsts.CollectorsRoleNodeCollector)
	if err != nil {
		return err
	}
	clusterWatcher, err := newCollectorWatcher(ctx, cw.odigosNS, k8sconsts.CollectorsRoleClusterGateway)
	if err != nil {
		return err
	}
	destsWatcher, err := kube.DefaultClient.OdigosClient.Destinations(cw.odigosNS).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	sourcesWatcher, err := kube.DefaultClient.OdigosClient.InstrumentationConfigs("").Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	return runWatcherLoop(ctx,
		watchers{
			nodeCollectors:    nodeWatcher,
			clusterCollectors: clusterWatcher,
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

func runWatcherLoop(ctx context.Context, w watchers, notifyChan chan<- deleteNotification) error {
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
				notifyChan <- deleteNotification{notificationType: nodeCollector, object: pod.Name}
			}
		case event, ok := <-cch:
			if !ok {
				return errors.New("cluster collector watcher closed")
			}
			switch event.Type {
			case watch.Deleted:
				pod := event.Object.(*corev1.Pod)
				notifyChan <- deleteNotification{notificationType: clusterCollector, object: pod.Name}
			}
		case event, ok := <-dch:
			if !ok {
				return errors.New("destination watcher closed")
			}
			switch event.Type {
			case watch.Deleted:
				d := event.Object.(*v1alpha1.Destination)
				notifyChan <- deleteNotification{notificationType: destination, object: d.Name}
			}
		case event, ok := <-sch:
			if !ok {
				return errors.New("source watcher closed")
			}
			switch event.Type {
			case watch.Deleted:
				ic := event.Object.(*v1alpha1.InstrumentationConfig)
				name, kind, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(ic.Name)
				if err != nil {
					fmt.Printf("error getting workload info: %v\n", err)
				}
				notifyChan <- deleteNotification{notificationType: source, sourceID: common.SourceID{Kind: kind, Name: name, Namespace: ic.Namespace}}
			}
		}
	}
}
