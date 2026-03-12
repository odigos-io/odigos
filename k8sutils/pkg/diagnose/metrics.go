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

// Stage constant for the metrics diagnose phase.
const StageMetrics Stage = "metrics"

// FetchOdigosMetrics gathers Prometheus metrics from Odigos system components:
// - Odiglet pods (odiglet container + data-collection container per pod)
// - Gateway collector pods
func FetchOdigosMetrics(ctx context.Context, client kubernetes.Interface, builder Builder, metricsDir, odigosNamespace string) error {
	klog.V(2).InfoS("Fetching Odigos metrics", "namespace", odigosNamespace)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := fetchMetricsFromOdigletPods(ctx, client, builder, odigosNamespace, metricsDir); err != nil {
			klog.V(1).ErrorS(err, "Failed to get metrics from odiglet pods")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := fetchMetricsFromGatewayPods(ctx, client, builder, odigosNamespace, metricsDir); err != nil {
			klog.V(1).ErrorS(err, "Failed to get metrics from gateway pods")
		}
	}()

	wg.Wait()
	return nil
}

// fetchMetricsFromOdigletPods gathers metrics from every container in each odiglet pod:
// the odiglet container (metrics port) and the data-collection container (collector metrics).
func fetchMetricsFromOdigletPods(ctx context.Context, client kubernetes.Interface, builder Builder, namespace, metricsDir string) error {
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: k8sconsts.OdigosCollectorRoleLabel + "=" + string(k8sconsts.CollectorsRoleNodeCollector),
	})
	if err != nil {
		return err
	}
	if len(pods.Items) == 0 {
		return nil
	}

	endpoints := []struct {
		fileSuffix string
		port       int32
	}{
		{k8sconsts.OdigletContainerName, k8sconsts.OdigletMetricsServerPort},
		{k8sconsts.OdigosNodeCollectorContainerName, k8sconsts.OdigosNodeCollectorOwnTelemetryPortDefault},
	}

	var wg sync.WaitGroup
	for i := range pods.Items {
		pod := &pods.Items[i]
		podName := pod.Name
		for _, ep := range endpoints {
			ep := ep
			wg.Add(1)
			go func() {
				defer wg.Done()
				klog.V(2).InfoS("Fetching metrics for odiglet pod", "podName", podName, "container", ep.fileSuffix)
				data, err := captureMetrics(ctx, client, podName, namespace, ep.port)
				if err != nil {
					klog.V(1).ErrorS(err, "Failed to get metrics", "podName", podName, "container", ep.fileSuffix)
					return
				}
				filename := podName + "-" + ep.fileSuffix
				if err := builder.AddFile(metricsDir, filename, data); err != nil {
					klog.V(1).ErrorS(err, "Failed to save metrics", "podName", podName, "filename", filename)
				}
			}()
		}
	}
	wg.Wait()
	return nil
}

// fetchMetricsFromGatewayPods gathers metrics from each gateway collector pod.
func fetchMetricsFromGatewayPods(ctx context.Context, client kubernetes.Interface, builder Builder, namespace, metricsDir string) error {
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: k8sconsts.OdigosCollectorRoleLabel + "=" + string(k8sconsts.CollectorsRoleClusterGateway),
	})
	if err != nil {
		return err
	}
	if len(pods.Items) == 0 {
		return nil
	}

	port := k8sconsts.OdigosClusterCollectorOwnTelemetryPortDefault
	var wg sync.WaitGroup
	for i := range pods.Items {
		pod := &pods.Items[i]
		podName := pod.Name
		wg.Add(1)
		go func() {
			defer wg.Done()
			klog.V(2).InfoS("Fetching metrics for gateway pod", "podName", podName)
			data, err := captureMetrics(ctx, client, podName, namespace, port)
			if err != nil {
				klog.V(1).ErrorS(err, "Failed to get metrics", "podName", podName)
				return
			}
			if err := builder.AddFile(metricsDir, podName, data); err != nil {
				klog.V(1).ErrorS(err, "Failed to save metrics", "podName", podName)
			}
		}()
	}
	wg.Wait()
	return nil
}

func captureMetrics(ctx context.Context, client kubernetes.Interface, podName, namespace string, port int32) ([]byte, error) {
	portNumber := strconv.Itoa(int(port))
	proxyURL := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s:%s/proxy/metrics", namespace, podName, portNumber)

	request := client.CoreV1().RESTClient().
		Get().
		AbsPath(proxyURL).
		Do(ctx)

	response, err := request.Raw()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, string(response))
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, bytes.NewReader(response)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
