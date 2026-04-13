package diagnose

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

// StageProfileService returns the stage name for profiling a specific service,
// e.g. "profiles/odiglet". Each service is reported as its own stage so that
// the progress UI can show incremental updates instead of one long "profiles" wait.
func StageProfileService(serviceName string) Stage {
	return Stage("profiles/" + serviceName)
}

// ProfileServiceNames returns the ordered list of services that will be profiled.
func ProfileServiceNames() []string {
	names := make([]string, 0, len(servicesProfilingMetadata))
	for name := range servicesProfilingMetadata {
		names = append(names, name)
	}
	return names
}

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
	return "/profile?seconds=10"
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
		Port: k8sconsts.DefaultPprofEndpointPort,
		Selector: labels.Set{
			"app.kubernetes.io/name": k8sconsts.OdigletAppLabelValue,
		}.AsSelector(),
	},
	"data-collection": {
		Port: k8sconsts.CollectorsPprofEndpointPort,
		Selector: labels.Set{
			k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleNodeCollector),
		}.AsSelector(),
	},
	"gateway": {
		Port: k8sconsts.CollectorsPprofEndpointPort,
		Selector: labels.Set{
			k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleClusterGateway),
		}.AsSelector(),
	},
}

// FetchServiceProfiles collects pprof profiles for a single Odigos service (e.g. "odiglet").
func FetchServiceProfiles(ctx context.Context, client kubernetes.Interface, builder Builder, profileDir, odigosNamespace, serviceName string) error {
	service, ok := servicesProfilingMetadata[serviceName]
	if !ok {
		return fmt.Errorf("unknown profiling service %q", serviceName)
	}

	klog.V(2).InfoS("Fetching profiles for service", "service", serviceName, "namespace", odigosNamespace)

	podsToProfile, err := client.CoreV1().Pods(odigosNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: service.Selector.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to list pods for service %s: %w", serviceName, err)
	}

	if len(podsToProfile.Items) == 0 {
		return nil
	}

	var podsWaitGroup sync.WaitGroup
	for i := range podsToProfile.Items {
		pod := &podsToProfile.Items[i]
		podsWaitGroup.Add(1)

		go func(pod corev1.Pod, pprofPort int32) {
			defer podsWaitGroup.Done()

			directoryName := fmt.Sprintf("%s-%s-%s", pod.Name, pod.Spec.NodeName, serviceName)
			nodeProfileDir := fmt.Sprintf("%s/%s", profileDir, directoryName)

			var profileWaitGroup sync.WaitGroup
			for _, profileMetricFunction := range ProfilingMetricsFunctions {
				profileFunc := profileMetricFunction
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
		}(*pod, service.Port)
	}

	podsWaitGroup.Wait()
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
	// Query params must use Request.Param; embedding "?" in AbsPath is escaped into Path and never reaches the apiserver.
	suffix := strings.TrimPrefix(profileInterface.GetUrlSuffix(), "/")
	pprofPath, rawQuery, hasQuery := strings.Cut(suffix, "?")
	proxyURL := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s:%d/proxy/debug/pprof/%s", namespace, podName, pprofPort, pprofPath)

	req := client.CoreV1().RESTClient().Get().AbsPath(proxyURL)
	if hasQuery {
		q, err := url.ParseQuery(rawQuery)
		if err != nil {
			return nil, fmt.Errorf("parse pprof query %q: %w", rawQuery, err)
		}
		for key, vals := range q {
			for _, v := range vals {
				req.Param(key, v)
			}
		}
	}

	response, err := req.Do(ctx).Raw()
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
