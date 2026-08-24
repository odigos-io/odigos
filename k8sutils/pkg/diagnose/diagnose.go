package diagnose

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
)

// collectionTimeout bounds diagnose collection when it is detached from an HTTP
// request context (see CollectionContext).
const collectionTimeout = 10 * time.Minute

// maxConcurrentK8sOps caps parallel Kubernetes API calls within a diagnose stage
// so the shared client-go rate limiter is not saturated by logs/profiles/metrics.
const maxConcurrentK8sOps = 16

// CollectionContext returns a context for diagnose collection that is not canceled
// when parent is (for example when a GraphQL client aborts or a proxy times out).
// A timeout is still applied so collection cannot run forever.
func CollectionContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(parent), collectionTimeout)
}

// Stage identifies a single phase of diagnose collection.
// When a stage completes, a StageResult is sent on the onStageComplete channel (if non-nil).
// Stage constants are defined in the file for each stage (workloads.go, crds.go, etc.).
type Stage string

// StageResult is sent on onStageComplete when a stage finishes.
// Status is nil on success, or the error returned by that stage's fetch.
type StageResult struct {
	Stage  Stage
	Status error
}

// runStage runs fetch in a goroutine and sends a StageResult on onStageComplete (Status is nil on success, or the fetch error).
func runStage(
	wg *sync.WaitGroup,
	stage Stage,
	onStageComplete chan<- StageResult,
	fetch func() error,
) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := fetch()
		if err != nil {
			klog.V(1).ErrorS(err, "Stage failed", "stage", stage)
		}
		if onStageComplete != nil {
			onStageComplete <- StageResult{Stage: stage, Status: err}
		}
	}()
}

// RunDiagnose collects all diagnostic data based on the provided options.
// If onStageComplete is non-nil, each stage sends a StageResult (Stage and Status: nil on success, or the error) when it completes.
// The caller must not close the channel; RunDiagnose does not close it.
func RunDiagnose(
	ctx context.Context,
	client kubernetes.Interface,
	dynamicClient dynamic.Interface,
	discoveryClient discovery.DiscoveryInterface,
	odigosClient odigosv1alpha1.OdigosV1alpha1Interface,
	builder Builder,
	rootDir string,
	opts Options,
	onStageComplete chan<- StageResult,
) error {
	if opts.OdigosNamespace == "" {
		return fmt.Errorf("odigos namespace is required")
	}

	klog.V(1).InfoS("Starting diagnose collection", "namespace", opts.OdigosNamespace, "rootDir", rootDir)

	var wg sync.WaitGroup

	// Fetch Odigos workloads (deployments, daemonsets, statefulsets with their pods and logs)
	runStage(&wg, StageWorkloads, onStageComplete, func() error {
		return FetchOdigosWorkloads(ctx, client, dynamicClient, builder, rootDir, opts.OdigosNamespace, opts.IncludeLogs)
	})

	if opts.IncludeCRDs {
		runStage(&wg, StageCRDs, onStageComplete, func() error {
			return FetchOdigosCRDs(ctx, dynamicClient, discoveryClient, builder, rootDir, opts.OdigosNamespace)
		})
	}

	if opts.IncludeProfiles {
		profileDir := GetProfileDir(rootDir, opts.OdigosNamespace)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, svcName := range ProfileServiceNames() {
				err := FetchServiceProfiles(ctx, client, builder, profileDir, opts.OdigosNamespace, svcName)
				if err != nil {
					klog.V(1).ErrorS(err, "Stage failed", "stage", StageProfileService(svcName))
				}
				if onStageComplete != nil {
					onStageComplete <- StageResult{Stage: StageProfileService(svcName), Status: err}
				}
			}
		}()
	}

	if opts.IncludeMetrics {
		runStage(&wg, StageMetrics, onStageComplete, func() error {
			metricsDir := GetMetricsDir(rootDir, opts.OdigosNamespace)
			return FetchOdigosMetrics(ctx, client, builder, metricsDir, opts.OdigosNamespace)
		})
	}

	if opts.IncludeConfigMaps {
		runStage(&wg, StageConfigMaps, onStageComplete, func() error {
			configMapDir := GetConfigMapsDir(rootDir, opts.OdigosNamespace)
			return FetchConfigMaps(ctx, client, builder, configMapDir, opts.OdigosNamespace)
		})
	}

	if opts.IncludeSourceWorkloads {
		runStage(&wg, StageSourceWorkloads, onStageComplete, func() error {
			// Source workloads collect YAMLs only. IncludeLogs is for Odigos
			// component pods; fetching logs from every instrumented application
			// saturates the API client and cancels in-flight profile lists.
			return FetchSourceWorkloads(ctx, client, dynamicClient, odigosClient, builder, rootDir, opts.SourceWorkloadNamespaces, false)
		})
	}

	wg.Wait()

	stats := builder.GetStats()
	klog.V(1).InfoS("Diagnose collection completed",
		"fileCount", stats.FileCount,
		"totalSize", FormatBytes(stats.TotalSize))

	return nil
}

// RequestedStages returns the list of stages that will run for the given options.
// Callers can use this to show planned stages before calling RunDiagnose (e.g. for progress UI or initial SSE).
func RequestedStages(opts Options) []Stage {
	var stages []Stage
	stages = append(stages, StageWorkloads)
	if opts.IncludeCRDs {
		stages = append(stages, StageCRDs)
	}
	if opts.IncludeProfiles {
		for _, svcName := range ProfileServiceNames() {
			stages = append(stages, StageProfileService(svcName))
		}
	}
	if opts.IncludeMetrics {
		stages = append(stages, StageMetrics)
	}
	if opts.IncludeConfigMaps {
		stages = append(stages, StageConfigMaps)
	}
	if opts.IncludeSourceWorkloads {
		stages = append(stages, StageSourceWorkloads)
	}
	return stages
}
