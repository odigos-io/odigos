package diagnose_util

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

var ProfilingMetricsFunctions = []ProfileInterface{CPUProfiler{}, HeapProfiler{}, GoRoutineProfiler{}, AllocsProfiler{}}

type ProfilingPodConfig struct {
	Port     int32
	Selector labels.Selector
}

// servicesProfilingMetadata is a map that associates service names with their corresponding pprof endpoint ports and selectors.
// To add new Odigos services to be profiled, include the service name along with the appropriate port and selectors in this map.
// Note: Since HostNetwork is set to true in DaemonSet services, pods expose ports on the node itself.
// Therefore, the same port cannot be used for multiple services on the same node.
var servicesProfilingMetadata = map[string]ProfilingPodConfig{
	"odiglet": {
		Port: k8sconsts.OdigletPprofEndpointPort,
		Selector: labels.Set{
			"app.kubernetes.io/name": k8sconsts.OdigletAppLabelValue}.AsSelector(),
	},
	"data-collection": {
		Port: k8sconsts.CollectorsPprofEndpointPort,
		Selector: labels.Set{
			k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleNodeCollector)}.AsSelector(),
	},
	"gateway": {
		Port: k8sconsts.CollectorsPprofEndpointPort,
		Selector: labels.Set{
			k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleClusterGateway)}.AsSelector(),
	},
}

type ProfileInterface interface {
	GetFileName() string
	GetUrlSuffix() string
}

type CPUProfiler struct{}

func (c CPUProfiler) GetFileName() string {
	return "cpu_profile.prof"
}

func (c CPUProfiler) GetUrlSuffix() string {
	return "/profile"
}

type HeapProfiler struct{}

func (h HeapProfiler) GetFileName() string {
	return "heap_profile.prof"
}

func (h HeapProfiler) GetUrlSuffix() string {
	return "/heap"
}

type GoRoutineProfiler struct{}

func (h GoRoutineProfiler) GetFileName() string {
	return "goroutine_profile.prof"
}

func (h GoRoutineProfiler) GetUrlSuffix() string {
	return "/goroutine"
}

type AllocsProfiler struct{}

func (h AllocsProfiler) GetFileName() string {
	return "allocs_profile.prof"
}

func (h AllocsProfiler) GetUrlSuffix() string {
	return "/allocs"
}

func FetchOdigosProfiles(ctx context.Context, client *kube.Client, profileDir string) error {
	fmt.Printf("Fetching Odigos Profiles...\n")
	odigosNamespace, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		return nil
	}

	var podsWaitGroup sync.WaitGroup
	for serviceName, service := range servicesProfilingMetadata {
		selector := service.Selector
		podsToProfile, err := client.CoreV1().Pods(odigosNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			return err
		}
		for _, pod := range podsToProfile.Items {
			klog.V(2).InfoS("Fetching profile for Pod", "podName", pod.Name, "service", serviceName)
			podsWaitGroup.Add(1)

			go func(pod v1.Pod, pprofPort int32, svcName string) {
				defer podsWaitGroup.Done()

				directoryName := fmt.Sprintf("%s-%s-%s", pod.Name, pod.Spec.NodeName, svcName)
				nodeFilePath := filepath.Join(profileDir, directoryName)
				err := os.MkdirAll(nodeFilePath, os.ModePerm)
				if err != nil {
					klog.V(1).ErrorS(err, "Failed to create directory for node", "nodeFilePath", nodeFilePath)
					return
				}

				var profileWaitGroup sync.WaitGroup
				for _, profileMetricFunction := range ProfilingMetricsFunctions {
					profileMetricFunction := profileMetricFunction
					metricFilePath := filepath.Join(nodeFilePath, profileMetricFunction.GetFileName())

					profileWaitGroup.Add(1)

					go func(metricFilePath string, profileFunc ProfileInterface) {
						defer profileWaitGroup.Done()

						metricFile, err := os.OpenFile(metricFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
						if err != nil {
							klog.V(1).ErrorS(err, "Failed to create file", "metricFilePath", metricFilePath)
							return
						}
						defer metricFile.Close()

						const maxRetries = 3
						for attempt := 1; attempt <= maxRetries; attempt++ {
							err = captureProfile(ctx, client, pod.Name, pprofPort, odigosNamespace, metricFile, profileFunc)
							if err == nil {
								break
							}

							klog.V(1).ErrorS(err, "Failed to capture profile data", "podName", pod.Name, "node", pod.Spec.NodeName, "profileType", profileFunc.GetFileName(), "attempt", attempt)

							if attempt < maxRetries {
								time.Sleep(5 * time.Second)
							} else {
								klog.V(1).ErrorS(err, "Max retries reached, giving up", "podName", pod.Name, "profileType", profileFunc.GetFileName())
							}
						}
					}(metricFilePath, profileMetricFunction)
				}

				profileWaitGroup.Wait()
			}(pod, service.Port, serviceName)
		}
	}
	podsWaitGroup.Wait()
	return nil
}

func captureProfile(ctx context.Context, client *kube.Client, podName string, pprofPort int32, namespace string, metricFile *os.File, profileInterface ProfileInterface) error {
	proxyURL := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s:%d/proxy/debug/pprof%s", namespace, podName, pprofPort, profileInterface.GetUrlSuffix())

	request := client.Clientset.CoreV1().RESTClient().
		Get().
		AbsPath(proxyURL).
		Do(ctx)

	response, err := request.Raw()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(response))
	}

	_, err = io.Copy(metricFile, bytes.NewReader(response))
	if err != nil {
		return err
	}

	return nil
}

func FetchOdigosCollectorMetrics(ctx context.Context, client *kube.Client, metricsDir string) error {
	fmt.Printf("Fetching Odigos Collectors Metrics...\n")

	odigosNamespace, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		return nil
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = collectMetrics(ctx, client, odigosNamespace, metricsDir, k8sconsts.CollectorsRoleClusterGateway)
		if err != nil {
			klog.V(1).ErrorS(err, "Failed to get metrics data", "collectorRole", k8sconsts.CollectorsRoleClusterGateway)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = collectMetrics(ctx, client, odigosNamespace, metricsDir, k8sconsts.CollectorsRoleNodeCollector)
		if err != nil {
			klog.V(1).ErrorS(err, "Failed to get metrics data", "collectorRole", k8sconsts.CollectorsRoleNodeCollector)
		}
	}()

	wg.Wait()

	return nil
}

func collectMetrics(ctx context.Context, client *kube.Client, odigosNamespace string, metricsDir string, collectorRole k8sconsts.CollectorRole) error {

	collectorPods, err := client.CoreV1().Pods(odigosNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "odigos.io/collector-role=" + string(collectorRole),
	})
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	for _, collectorPod := range collectorPods.Items {
		klog.V(2).InfoS("Fetching metrics for pod", "podName", collectorPod.Name)
		metricFilePath := filepath.Join(metricsDir, collectorPod.Name)
		metricFile, err := os.OpenFile(metricFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			klog.V(1).ErrorS(err, "Failed to create file", "metricFilePath", metricFilePath)
			continue
		}
		defer metricFile.Close()

		wg.Add(1)

		go func() {
			defer wg.Done()
			err = captureMetrics(ctx, client, collectorPod.Name, odigosNamespace, metricFile, collectorRole)
			if err != nil {
				klog.V(1).ErrorS(err, "Failed to get metrics data", "podName", collectorPod.Name)
			}
		}()
	}

	wg.Wait()
	return nil
}

func captureMetrics(ctx context.Context, client *kube.Client, podName string, namespace string, metricFile *os.File, collectorRole k8sconsts.CollectorRole) error {
	portNumber := ""
	if collectorRole == k8sconsts.CollectorsRoleClusterGateway {
		portNumber = strconv.Itoa(int(k8sconsts.OdigosClusterCollectorOwnTelemetryPortDefault))
	} else if collectorRole == k8sconsts.CollectorsRoleNodeCollector {
		portNumber = strconv.Itoa(int(k8sconsts.OdigosNodeCollectorOwnTelemetryPortDefault))
	} else {
		return nil
	}

	proxyURL := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s:%s/proxy/metrics", namespace, podName, portNumber)

	// Make the HTTP GET request via the API server proxy
	request := client.Clientset.CoreV1().RESTClient().
		Get().
		AbsPath(proxyURL).
		Do(ctx)

	response, err := request.Raw()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(response))
	}

	_, err = io.Copy(metricFile, bytes.NewReader(response))
	if err != nil {
		return err
	}

	return nil
}
