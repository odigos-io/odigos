package collectormetrics

import (
	"context"
	"sync"

	"github.com/odigos-io/odigos/frontend/kube"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type collectorWatcher struct {
	odigosNS string
	nodeCollectorDeleted chan string
	clusterCollectorDeleted chan string
}

func startCollectorWatcher(ctx context.Context, cw *collectorWatcher, wg *sync.WaitGroup) error {
	nodeWatcher, err := newCollectorWatcher(ctx, cw.odigosNS, k8sconsts.OdigosNodeCollectorLabel)
	if err != nil {
		return err
	}
	clusterWatcher, err := newCollectorWatcher(ctx, cw.odigosNS, k8sconsts.OdigosClusterCollectorLabel)
	if err != nil {
		return err
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		runNodeCollectorWatcher(ctx, nodeWatcher, cw.nodeCollectorDeleted)
	}()

	wg.Add(1)
	go func(){
		defer wg.Done()
		runNodeCollectorWatcher(ctx, clusterWatcher, cw.clusterCollectorDeleted)
	} ()
	return nil
}

func newCollectorWatcher(ctx context.Context, odigosNS string, matchLabel string) (watch.Interface, error) {
	return kube.DefaultClient.CoreV1().Pods(odigosNS).Watch(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				matchLabel: "true",
			},
		}),
	})
}

func runNodeCollectorWatcher(ctx context.Context, watcher watch.Interface, notifyChan chan string) {
	ch := watcher.ResultChan()
	for {
		select {
		case <-ctx.Done():
			watcher.Stop()
			close(notifyChan)
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			switch event.Type {
			case watch.Deleted:
				pod := event.Object.(*corev1.Pod)
				notifyChan <- pod.Name
			}
		}
	}
}