package diagnose_util

import (
	"bytes"
	"context"
	"fmt"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sync"
)

var ProfilingMetricsFunctions = []ProfileInterface{CPUProfiler{}, HeapProfiler{}, GoRoutineProfiler{}, AllocsProfiler{}}

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

	odigletPods, err := client.CoreV1().Pods(odigosNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=odiglet",
	})
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	for _, odigletPod := range odigletPods.Items {
		fmt.Printf("Fetching profile for node: %v", odigletPod.Spec.NodeName)
		nodeFilePath := filepath.Join(profileDir, odigletPod.Spec.NodeName)
		err := os.Mkdir(nodeFilePath, os.ModePerm)
		if err != nil {
			fmt.Printf("Error creating directory for node: %v, because: %v", nodeFilePath, err)
			continue
		}

		for _, profileMetricFunction := range ProfilingMetricsFunctions {
			metricFilePath := filepath.Join(nodeFilePath, profileMetricFunction.GetFileName())
			metricFile, err := os.OpenFile(metricFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
			if err != nil {
				fmt.Printf("Error creating file: %v, because: %v", metricFilePath, err)
				continue
			}
			defer metricFile.Close()

			wg.Add(1)

			go func() {
				defer wg.Done()
				podName := odigletPod.Name
				err = captureProfile(ctx, client, podName, odigosNamespace, metricFile, profileMetricFunction)
				if err != nil {
					fmt.Printf("Error Getting Profile Data  of: %v, because: %v\n", profileMetricFunction, err)
				}
			}()
		}

		wg.Wait()
	}

	return nil
}

func captureProfile(ctx context.Context, client *kube.Client, podName string, namespace string, metricFile *os.File, profileInterface ProfileInterface) error {
	proxyURL := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s:6060/proxy/debug/pprof%s", namespace, podName, profileInterface.GetUrlSuffix())

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
		fmt.Printf("Fetching metrics for pod: %v", collectorPod.Name)
		metricFilePath := filepath.Join(metricsDir, collectorPod.Name)
		metricFile, err := os.OpenFile(metricFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Printf("Error creating file: %v, because: %v", metricFilePath, err)
			continue
		}
		defer metricFile.Close()

		wg.Add(1)

		go func() {
			defer wg.Done()
			err = captureMetrics(ctx, client, collectorPod.Name, odigosNamespace, metricFile, collectorRole)
			if err != nil {
				fmt.Printf("Error Getting Metrics Data of: %v, because: %v\n", metricFile, err)
			}
		}()
	}

	wg.Wait()
	return nil
}

func captureMetrics(ctx context.Context, client *kube.Client, podName string, namespace string, metricFile *os.File, collectorRole consts.CollectorRole) error {
	portNumber := ""
	if collectorRole == consts.CollectorsRoleClusterGateway {
		portNumber = "8888"
	} else if collectorRole == consts.CollectorsRoleNodeCollector {
		portNumber = "55682"
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
