package collectormetrics

import (
	"context"
	"errors"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services/common"
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

	return runWatcherLoop(ctx, nodeCollectorWatcher, clusterCollectorWatcher, cw.deleteNotifications)
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

func runWatcherLoop(ctx context.Context, nodeCollectors, clusterCollectors watch.Interface, notifyChan chan<- notification) error {
	nch := nodeCollectors.ResultChan()
	cch := clusterCollectors.ResultChan()
	for {
		select {
		case <-ctx.Done():
			nodeCollectors.Stop()
			clusterCollectors.Stop()
			close(notifyChan)
			return nil
		case event, ok := <-nch:
			if !ok {
				return errors.New("node collector watcher closed")
			}
			if event.Type == watch.Deleted {
				pod := event.Object.(*corev1.Pod)
				notifyChan <- notification{notificationType: nodeCollector, object: pod.Name}
			}
		case event, ok := <-cch:
			if !ok {
				return errors.New("cluster collector watcher closed")
			}
			if event.Type == watch.Deleted {
				pod := event.Object.(*corev1.Pod)
				notifyChan <- notification{notificationType: clusterCollector, object: pod.Name, eventType: watch.Deleted}
			}
		}
	}
}
