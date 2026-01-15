package diagnose

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

// FetchOdigosCollectorMetrics collects Prometheus metrics from Odigos collectors
func FetchOdigosCollectorMetrics(ctx context.Context, client kubernetes.Interface, collector Collector, metricsDir, odigosNamespace string) error {
	fmt.Printf("Fetching Odigos Collectors Metrics...\n")
	klog.V(2).InfoS("Fetching Odigos Collector Metrics", "namespace", odigosNamespace)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := collectMetricsForRole(ctx, client, collector, odigosNamespace, metricsDir, k8sconsts.CollectorsRoleClusterGateway); err != nil {
			klog.V(1).ErrorS(err, "Failed to get metrics data", "collectorRole", k8sconsts.CollectorsRoleClusterGateway)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := collectMetricsForRole(ctx, client, collector, odigosNamespace, metricsDir, k8sconsts.CollectorsRoleNodeCollector); err != nil {
			klog.V(1).ErrorS(err, "Failed to get metrics data", "collectorRole", k8sconsts.CollectorsRoleNodeCollector)
		}
	}()

	wg.Wait()
	return nil
}

func collectMetricsForRole(
	ctx context.Context,
	client kubernetes.Interface,
	collector Collector,
	odigosNamespace string,
	metricsDir string,
	collectorRole k8sconsts.CollectorRole,
) error {
	collectorPods, err := client.CoreV1().Pods(odigosNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: k8sconsts.OdigosCollectorRoleLabel + "=" + string(collectorRole),
	})
	if err != nil {
		fmt.Printf("  Warning: Failed to list %s pods: %v\n", collectorRole, err)
		return err
	}

	if len(collectorPods.Items) == 0 {
		fmt.Printf("  No %s pods found for metrics collection\n", collectorRole)
		return nil
	}

	fmt.Printf("  Found %d %s pod(s) for metrics collection\n", len(collectorPods.Items), collectorRole)

	var wg sync.WaitGroup

	for i := 0; i < len(collectorPods.Items); i++ {
		pod := &collectorPods.Items[i]
		wg.Add(1)

		go func() {
			defer wg.Done()
			klog.V(2).InfoS("Fetching metrics for pod", "podName", pod.Name)

			data, err := captureMetrics(ctx, client, pod.Name, odigosNamespace, collectorRole)
			if err != nil {
				klog.V(1).ErrorS(err, "Failed to get metrics data", "podName", pod.Name)
				return
			}

			filename := pod.Name
			if err := collector.AddFile(metricsDir, filename, data); err != nil {
				klog.V(1).ErrorS(err, "Failed to save metrics", "podName", pod.Name)
			}
		}()
	}

	wg.Wait()
	return nil
}

func captureMetrics(
	ctx context.Context,
	client kubernetes.Interface,
	podName string,
	namespace string,
	collectorRole k8sconsts.CollectorRole,
) ([]byte, error) {
	portNumber := ""
	switch collectorRole {
	case k8sconsts.CollectorsRoleClusterGateway:
		portNumber = strconv.Itoa(int(k8sconsts.OdigosClusterCollectorOwnTelemetryPortDefault))
	case k8sconsts.CollectorsRoleNodeCollector:
		portNumber = strconv.Itoa(int(k8sconsts.OdigosNodeCollectorOwnTelemetryPortDefault))
	default:
		return nil, nil
	}

	proxyURL := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s:%s/proxy/metrics", namespace, podName, portNumber)

	request := client.CoreV1().RESTClient().
		Get().
		AbsPath(proxyURL).
		Do(ctx)

	response, err := request.Raw()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, string(response))
	}

	// Copy the response to a buffer
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, bytes.NewReader(response)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
