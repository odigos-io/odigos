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

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
		Port: consts.OdigletPprofEndpointPort,
		Selector: labels.Set{
			"app.kubernetes.io/name": resources.OdigletAppLabelValue}.AsSelector(),
	},
	"data-collection": {
		Port: consts.CollectorsPprofEndpointPort,
		Selector: labels.Set{
			consts.OdigosCollectorRoleLabel: string(consts.CollectorsRoleNodeCollector)}.AsSelector(),
	},
	"gateway": {
		Port: consts.CollectorsPprofEndpointPort,
		Selector: labels.Set{
			consts.OdigosCollectorRoleLabel: string(consts.CollectorsRoleClusterGateway)}.AsSelector(),
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
	odigosNamespace, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		return nil
	}
	var podsWaitGroup sync.WaitGroup
	for _, service := range servicesProfilingMetadata {
		selector := service.Selector
		podsToProfile, err := client.CoreV1().Pods(odigosNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			return err
		}
		for _, pod := range podsToProfile.Items {
			fmt.Printf("Fetching profile for Pod: %s\n", pod.Name)
			podsWaitGroup.Add(1)
			go func(pod v1.Pod, pprofPort int32) {
				defer podsWaitGroup.Done()

				directoryName := fmt.Sprintf("%s-%s", pod.Name, pod.Spec.NodeName)
				nodeFilePath := filepath.Join(profileDir, directoryName)
				err := os.Mkdir(nodeFilePath, os.ModePerm)
				if err != nil {
					fmt.Printf("Error creating directory for node: %v, because: %v", nodeFilePath, err)
					return
				}
				// Inner WaitGroup for profiling functions of this pod
				var profileWaitGroup sync.WaitGroup
				for _, profileMetricFunction := range ProfilingMetricsFunctions {
					profileMetricFunction := profileMetricFunction // Capture range variable
					metricFilePath := filepath.Join(nodeFilePath, profileMetricFunction.GetFileName())

					profileWaitGroup.Add(1)

					go func(metricFilePath string, profileFunc ProfileInterface) {
						defer profileWaitGroup.Done()
						metricFile, err := os.OpenFile(metricFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
						if err != nil {
							fmt.Printf("Error creating file: %v, because: %v\n", metricFilePath, err)
							return
						}
						defer metricFile.Close()

						err = captureProfile(ctx, client, pod.Name, pprofPort, odigosNamespace, metricFile, profileFunc)
						if err != nil {
							fmt.Printf(
								"Failed to capture profile data for Pod: %s, Node: %s, Profile Type: %s. Reason: %v\n",
								pod.Name,
								pod.Spec.NodeName,
								profileFunc.GetFileName(),
								err,
							)
						}
					}(metricFilePath, profileMetricFunction)
				}
				// Wait for all profiling tasks of pod to complete
				profileWaitGroup.Wait()
			}(pod, service.Port)
		}
	}
	// Wait for all pod-level tasks to complete
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
		return err
	}

	_, err = io.Copy(metricFile, bytes.NewReader(response))
	if err != nil {
		return err
	}

	return nil
}

func FetchOdigosCollectorMetrics(ctx context.Context, client *kube.Client, metricsDir string) error {
	odigosNamespace, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		return nil
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = collectMetrics(ctx, client, odigosNamespace, metricsDir, consts.CollectorsRoleClusterGateway)
		if err != nil {
			fmt.Printf("Error Getting Metrics Data of: %v, because: %v\n", consts.CollectorsRoleClusterGateway, err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = collectMetrics(ctx, client, odigosNamespace, metricsDir, consts.CollectorsRoleNodeCollector)
		if err != nil {
			fmt.Printf("Error Getting Metrics Data of: %v, because: %v\n", consts.CollectorsRoleNodeCollector, err)
		}
	}()

	wg.Wait()

	return nil
}

func collectMetrics(ctx context.Context, client *kube.Client, odigosNamespace string, metricsDir string, collectorRole consts.CollectorRole) error {
	collectorPods, err := client.CoreV1().Pods(odigosNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "odigos.io/collector-role=" + string(collectorRole),
	})
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	for _, collectorPod := range collectorPods.Items {
		fmt.Println("Fetching metrics for pod:", collectorPod.Name)
		metricFilePath := filepath.Join(metricsDir, collectorPod.Name)
		metricFile, err := os.OpenFile(metricFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Println("Error creating file: %v, because: %v", metricFilePath, err)
			continue
		}
		defer metricFile.Close()

		wg.Add(1)

		go func() {
			defer wg.Done()
			err = captureMetrics(ctx, client, collectorPod.Name, odigosNamespace, metricFile, collectorRole)
			if err != nil {
				fmt.Println("Error Getting Metrics Data of: %v, because: %v\n", metricFile, err)
			}
		}()
	}

	wg.Wait()
	return nil
}

func captureMetrics(ctx context.Context, client *kube.Client, podName string, namespace string, metricFile *os.File, collectorRole consts.CollectorRole) error {
	portNumber := ""
	if collectorRole == consts.CollectorsRoleClusterGateway {
		portNumber = strconv.Itoa(int(consts.OdigosClusterCollectorOwnTelemetryPortDefault))
	} else if collectorRole == consts.CollectorsRoleNodeCollector {
		portNumber = strconv.Itoa(int(consts.OdigosNodeCollectorOwnTelemetryPortDefault))
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
		return err
	}

	_, err = io.Copy(metricFile, bytes.NewReader(response))
	if err != nil {
		return err
	}

	return nil
}
