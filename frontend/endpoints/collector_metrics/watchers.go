package collectormetrics

import (
	"context"

	"github.com/odigos-io/odigos/frontend/kube"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func newNodeCollectorWatcher(ctx context.Context, odigosNS string) (watch.Interface, error) {
	return kube.DefaultClient.CoreV1().Pods(odigosNS).Watch(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				k8sconsts.OdigosNodeCollectorLabel: "true",
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