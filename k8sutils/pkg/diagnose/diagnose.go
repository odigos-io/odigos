package diagnose

import (
	"context"
	"fmt"
	"sync"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// RunDiagnose collects all diagnostic data based on the provided options
func RunDiagnose(
	ctx context.Context,
	client kubernetes.Interface,
	dynamicClient dynamic.Interface,
	discoveryClient discovery.DiscoveryInterface,
	odigosClient odigosv1alpha1.OdigosV1alpha1Interface,
	collector Collector,
	rootDir string,
	opts Options,
) error {
	if opts.OdigosNamespace == "" {
		return fmt.Errorf("odigos namespace is required")
	}

	klog.V(1).InfoS("Starting diagnose collection", "namespace", opts.OdigosNamespace, "rootDir", rootDir)

	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	// Fetch Odigos workloads (deployments, daemonsets, statefulsets with their pods and logs)
	// This is always collected as it's core diagnostic data
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := FetchOdigosWorkloads(ctx, client, dynamicClient, collector, rootDir, opts.OdigosNamespace, opts.IncludeLogs); err != nil {
			klog.V(1).ErrorS(err, "Failed to fetch Odigos workloads")
			errChan <- fmt.Errorf("workloads: %w", err)
		}
	}()

	// Fetch Odigos CRDs (organized by namespace, cluster-scoped under odigos namespace)
	if opts.IncludeCRDs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := FetchOdigosCRDs(ctx, dynamicClient, discoveryClient, collector, rootDir, opts.OdigosNamespace); err != nil {
				klog.V(1).ErrorS(err, "Failed to fetch Odigos CRDs")
				errChan <- fmt.Errorf("crds: %w", err)
			}
		}()
	}

	// Fetch Odigos Profiles (goes under odigos namespace)
	if opts.IncludeProfiles {
		wg.Add(1)
		go func() {
			defer wg.Done()
			profileDir := GetProfileDir(rootDir, opts.OdigosNamespace)
			if err := FetchOdigosProfiles(ctx, client, collector, profileDir, opts.OdigosNamespace); err != nil {
				klog.V(1).ErrorS(err, "Failed to fetch Odigos profiles")
				errChan <- fmt.Errorf("profiles: %w", err)
			}
		}()
	}

	// Fetch Odigos Collector Metrics (goes under odigos namespace)
	if opts.IncludeMetrics {
		wg.Add(1)
		go func() {
			defer wg.Done()
			metricsDir := GetMetricsDir(rootDir, opts.OdigosNamespace)
			if err := FetchOdigosCollectorMetrics(ctx, client, collector, metricsDir, opts.OdigosNamespace); err != nil {
				klog.V(1).ErrorS(err, "Failed to fetch Odigos metrics")
				errChan <- fmt.Errorf("metrics: %w", err)
			}
		}()
	}

	// Fetch ConfigMaps (goes under odigos namespace)
	if opts.IncludeConfigMaps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			configMapDir := GetConfigMapsDir(rootDir, opts.OdigosNamespace)
			if err := FetchConfigMaps(ctx, client, collector, configMapDir, opts.OdigosNamespace); err != nil {
				klog.V(1).ErrorS(err, "Failed to fetch ConfigMaps")
				errChan <- fmt.Errorf("configmaps: %w", err)
			}
		}()
	}

	// Fetch instrumented source workloads (user's applications)
	if opts.IncludeSourceWorkloads {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := FetchSourceWorkloads(ctx, client, dynamicClient, odigosClient, collector, rootDir, opts.SourceWorkloadNamespaces, opts.IncludeLogs); err != nil {
				klog.V(1).ErrorS(err, "Failed to fetch source workloads")
				errChan <- fmt.Errorf("source workloads: %w", err)
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Collect and log all errors, but don't fail the overall operation
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		klog.V(1).InfoS("Some diagnose operations had errors", "errorCount", len(errs))
		for _, err := range errs {
			klog.V(1).ErrorS(err, "Diagnose error")
		}
	}

	stats := collector.GetStats()
	klog.V(1).InfoS("Diagnose collection completed",
		"fileCount", stats.FileCount,
		"totalSize", FormatBytes(stats.TotalSize))

	return nil
}
