package diagnose

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

// ProfileInterface defines the interface for different profile types
type ProfileInterface interface {
	GetFileName() string
	GetUrlSuffix() string
}

// CPUProfiler captures CPU profiles
type CPUProfiler struct{}

func (c CPUProfiler) GetFileName() string {
	return "cpu_profile.prof"
}

func (c CPUProfiler) GetUrlSuffix() string {
	return "/profile"
}

// HeapProfiler captures heap profiles
type HeapProfiler struct{}

func (h HeapProfiler) GetFileName() string {
	return "heap_profile.prof"
}

func (h HeapProfiler) GetUrlSuffix() string {
	return "/heap"
}

// GoRoutineProfiler captures goroutine profiles
type GoRoutineProfiler struct{}

func (g GoRoutineProfiler) GetFileName() string {
	return "goroutine_profile.prof"
}

func (g GoRoutineProfiler) GetUrlSuffix() string {
	return "/goroutine"
}

// AllocsProfiler captures allocation profiles
type AllocsProfiler struct{}

func (a AllocsProfiler) GetFileName() string {
	return "allocs_profile.prof"
}

func (a AllocsProfiler) GetUrlSuffix() string {
	return "/allocs"
}

// ProfilingMetricsFunctions is the list of profile types to collect
var ProfilingMetricsFunctions = []ProfileInterface{CPUProfiler{}, HeapProfiler{}, GoRoutineProfiler{}, AllocsProfiler{}}

// ProfilingPodConfig holds configuration for profiling a pod
type ProfilingPodConfig struct {
	Port     int32
	Selector labels.Selector
}

// servicesProfilingMetadata maps service names to their pprof endpoint configurations
var servicesProfilingMetadata = map[string]ProfilingPodConfig{
	"odiglet": {
		Port: k8sconsts.DefaultDebugPort,
		Selector: labels.Set{
			"app.kubernetes.io/name": k8sconsts.OdigletAppLabelValue,
		}.AsSelector(),
	},
	"data-collection": {
		Port: k8sconsts.CollectorsDebugPort,
		Selector: labels.Set{
			k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleNodeCollector),
		}.AsSelector(),
	},
	"gateway": {
		Port: k8sconsts.CollectorsDebugPort,
		Selector: labels.Set{
			k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleClusterGateway),
		}.AsSelector(),
	},
}

// FetchOdigosProfiles collects pprof profiles from Odigos components
func FetchOdigosProfiles(ctx context.Context, client kubernetes.Interface, builder Builder, profileDir, odigosNamespace string) error {
	fmt.Printf("Fetching Odigos Profiles...\n")
	klog.V(2).InfoS("Fetching Odigos Profiles", "namespace", odigosNamespace)

	var podsWaitGroup sync.WaitGroup
	var totalPods int

	for serviceName, service := range servicesProfilingMetadata {
		podsToProfile, err := client.CoreV1().Pods(odigosNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: service.Selector.String(),
		})
		if err != nil {
			fmt.Printf("  Warning: Failed to list pods for %s: %v\n", serviceName, err)
			continue
		}

		if len(podsToProfile.Items) == 0 {
			fmt.Printf("  No %s pods found for profiling\n", serviceName)
			continue
		}

		totalPods += len(podsToProfile.Items)
		fmt.Printf("  Found %d %s pod(s) for profiling\n", len(podsToProfile.Items), serviceName)

		for i := 0; i < len(podsToProfile.Items); i++ {
			pod := &podsToProfile.Items[i]
			podsWaitGroup.Add(1)

			go func(pod corev1.Pod, pprofPort int32, svcName string) {
				defer podsWaitGroup.Done()

				directoryName := fmt.Sprintf("%s-%s-%s", pod.Name, pod.Spec.NodeName, svcName)
				nodeProfileDir := fmt.Sprintf("%s/%s", profileDir, directoryName)

				var profileWaitGroup sync.WaitGroup
				for _, profileMetricFunction := range ProfilingMetricsFunctions {
					profileFunc := profileMetricFunction // capture range variable
					profileWaitGroup.Add(1)

					go func(profileFunc ProfileInterface) {
						defer profileWaitGroup.Done()

						const maxRetries = 3
						for attempt := 1; attempt <= maxRetries; attempt++ {
							data, err := captureProfile(ctx, client, pod.Name, pprofPort, odigosNamespace, profileFunc)
							if err == nil {
								if err := builder.AddFile(nodeProfileDir, profileFunc.GetFileName(), data); err != nil {
									klog.V(1).ErrorS(err, "Failed to save profile", "podName", pod.Name, "profileType", profileFunc.GetFileName())
								}
								break
							}

							klog.V(1).ErrorS(err, "Failed to capture profile data",
								"podName", pod.Name,
								"node", pod.Spec.NodeName,
								"profileType", profileFunc.GetFileName(),
								"attempt", attempt)

							if attempt < maxRetries {
								time.Sleep(5 * time.Second)
							} else {
								klog.V(1).ErrorS(err, "Max retries reached, giving up",
									"podName", pod.Name,
									"profileType", profileFunc.GetFileName())
							}
						}
					}(profileFunc)
				}

				profileWaitGroup.Wait()
			}(*pod, service.Port, serviceName)
		}
	}

	podsWaitGroup.Wait()

	if totalPods == 0 {
		fmt.Printf("  No pods found for profiling in namespace %s\n", odigosNamespace)
	}

	return nil
}

func captureProfile(
	ctx context.Context,
	client kubernetes.Interface,
	podName string,
	pprofPort int32,
	namespace string,
	profileInterface ProfileInterface,
) ([]byte, error) {
	proxyURL := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s:%d/proxy/debug/pprof%s", namespace, podName, pprofPort, profileInterface.GetUrlSuffix())

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
