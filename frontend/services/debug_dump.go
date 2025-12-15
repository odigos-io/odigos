package services

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/version"
	"sigs.k8s.io/yaml"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

const odigosGroupSuffix = "odigos.io"

// tarCollector wraps tar.Writer and tracks created directories
type tarCollector struct {
	tarWriter   *tar.Writer
	createdDirs map[string]bool
}

func newTarCollector(tw *tar.Writer) *tarCollector {
	return &tarCollector{
		tarWriter:   tw,
		createdDirs: make(map[string]bool),
	}
}

// DebugDump generates a tar.gz file containing logs and YAML manifests
// for all Odigos components running in the odigos system namespace.
// Query params:
//   - includeWorkloads: if "true", also include workload and pod YAMLs for each Source
func DebugDump(c *gin.Context) {
	ctx := c.Request.Context()
	ns := env.GetCurrentNamespace()
	includeWorkloads := c.Query("includeWorkloads") == "true"

	// Set headers for file download
	timestamp := time.Now().Format("20060102-150405")
	rootDir := fmt.Sprintf("odigos-debug-%s", timestamp)
	filename := fmt.Sprintf("%s.tar.gz", rootDir)
	c.Header("Content-Type", "application/gzip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Create gzip writer directly to response
	gzipWriter := gzip.NewWriter(c.Writer)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	collector := newTarCollector(tarWriter)

	// Collect odigos workloads (and optionally source workloads)
	if err := collectAllWorkloads(ctx, collector, rootDir, ns, includeWorkloads); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to collect workloads: %v", err)})
		return
	}

	// Collect Odigos CRDs dynamically
	if err := collectOdigosCRDs(ctx, collector, rootDir, ns); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to collect odigos CRDs: %v", err)})
		return
	}

	// Collect ConfigMaps from odigos namespace
	if err := collectConfigMaps(ctx, collector, rootDir, ns); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to collect configmaps: %v", err)})
		return
	}

	c.Status(http.StatusOK)
}

// workloadTarget represents a workload to collect
type workloadTarget struct {
	namespace   string
	name        string
	kind        k8sconsts.WorkloadKind
	dirName     string // folder name (e.g., "deployment-foo" or just "foo")
	includeLogs bool
}

// collectAllWorkloads collects workloads from odigos namespace and optionally from Sources
func collectAllWorkloads(ctx context.Context, collector *tarCollector, rootDir, odigosNs string, includeWorkloads bool) error {
	var targets []workloadTarget

	// Add all workloads from odigos namespace (with logs)
	deployments, _ := kube.DefaultClient.AppsV1().Deployments(odigosNs).List(ctx, metav1.ListOptions{})
	for _, d := range deployments.Items {
		targets = append(targets, workloadTarget{odigosNs, d.Name, k8sconsts.WorkloadKindDeployment, fmt.Sprintf("deployment-%s", d.Name), true})
	}

	daemonsets, _ := kube.DefaultClient.AppsV1().DaemonSets(odigosNs).List(ctx, metav1.ListOptions{})
	for _, d := range daemonsets.Items {
		targets = append(targets, workloadTarget{odigosNs, d.Name, k8sconsts.WorkloadKindDaemonSet, fmt.Sprintf("daemonset-%s", d.Name), true})
	}

	statefulsets, _ := kube.DefaultClient.AppsV1().StatefulSets(odigosNs).List(ctx, metav1.ListOptions{})
	for _, s := range statefulsets.Items {
		targets = append(targets, workloadTarget{odigosNs, s.Name, k8sconsts.WorkloadKindStatefulSet, fmt.Sprintf("statefulset-%s", s.Name), true})
	}

	// Optionally add workloads from Sources (without logs)
	if includeWorkloads {
		sourceList, _ := kube.DefaultClient.OdigosClient.Sources("").List(ctx, metav1.ListOptions{})
		if sourceList != nil {
			for _, source := range sourceList.Items {
				w := source.Spec.Workload
				if w.Name != "" && w.Namespace != "" && w.Kind != "" && w.Kind != k8sconsts.WorkloadKindNamespace {
					targets = append(targets, workloadTarget{w.Namespace, w.Name, w.Kind, fmt.Sprintf("%s-%s", workload.WorkloadKindLowerCaseFromKind(w.Kind), w.Name), false})
				}
			}
		}
	}

	// Collect all targets
	for _, t := range targets {
		workloadDir := path.Join(rootDir, t.namespace, t.dirName)
		_ = collectWorkload(ctx, collector, workloadDir, t.namespace, t.name, t.kind, t.includeLogs)
	}

	return nil
}

// discoverOdigosCRDs uses the discovery client to find all CRDs in groups ending with ".odigos.io"
func discoverOdigosCRDs() []schema.GroupVersionResource {
	var gvrs []schema.GroupVersionResource

	// Get all API groups and resources
	// Note: ServerGroupsAndResources can return partial results with errors for some groups,
	// which is normal. We use the results we get and ignore errors.
	_, apiResourceLists, _ := kube.DefaultClient.Discovery().ServerGroupsAndResources()

	for _, resourceList := range apiResourceLists {
		if resourceList == nil {
			continue
		}

		// Parse the GroupVersion
		gv, err := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			continue
		}

		// Check if this group is "odigos.io" or ends with ".odigos.io" (e.g., "actions.odigos.io")
		if gv.Group != odigosGroupSuffix && !strings.HasSuffix(gv.Group, "."+odigosGroupSuffix) {
			continue
		}

		// Add all resources from this group
		for _, resource := range resourceList.APIResources {
			// Skip subresources (they contain "/")
			if strings.Contains(resource.Name, "/") {
				continue
			}

			gvrs = append(gvrs, schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: resource.Name,
			})
		}
	}

	return gvrs
}

// collectOdigosCRDs dynamically discovers and collects all Odigos CRDs
func collectOdigosCRDs(ctx context.Context, collector *tarCollector, rootDir, ns string) error {
	gvrs := discoverOdigosCRDs()

	for _, gvr := range gvrs {
		// Errors are not fatal - continue with other CRDs
		_ = collectCRD(ctx, collector, rootDir, ns, gvr)
	}
	return nil
}

// collectCRD collects a single CRD type using dynamic client
func collectCRD(ctx context.Context, collector *tarCollector, rootDir, odigosNs string, gvr schema.GroupVersionResource) error {
	// Use capitalized resource name as directory (e.g., "destinations" -> "Destinations")
	dirName := capitalizeFirst(gvr.Resource)

	// Try to list from all namespaces first (works for both namespaced and cluster-scoped resources)
	list, err := kube.DefaultClient.DynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		// If all-namespace list fails, try namespace-scoped
		list, err = kube.DefaultClient.DynamicClient.Resource(gvr).Namespace(odigosNs).List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list %s: %w", gvr.Resource, err)
		}
	}

	if len(list.Items) == 0 {
		return nil
	}

	for i := range list.Items {
		item := &list.Items[i]

		// Clean managedFields from the object
		unstructured.RemoveNestedField(item.Object, "metadata", "managedFields")

		// Determine the namespace folder - use item's namespace, or odigos namespace for cluster-scoped resources
		itemNs := item.GetNamespace()
		if itemNs == "" {
			itemNs = odigosNs // cluster-scoped resources go under odigos namespace
		}

		crdDir := path.Join(rootDir, itemNs, dirName)
		filename := fmt.Sprintf("%s.yaml", item.GetName())

		// Marshal to YAML
		yamlData, err := yaml.Marshal(item.Object)
		if err != nil {
			continue // Skip this item but continue with others
		}

		_ = collector.addFile(crdDir, filename, yamlData)
	}

	return nil
}

// collectConfigMaps collects all ConfigMaps from the odigos namespace
func collectConfigMaps(ctx context.Context, collector *tarCollector, rootDir, ns string) error {
	configMaps, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list configmaps: %w", err)
	}

	configMapDir := path.Join(rootDir, ns, "ConfigMaps")
	for _, cm := range configMaps.Items {
		if err := addResourceYAML(collector, configMapDir, "configmap", cm.Name, &cm); err != nil {
			continue // Skip this item but continue with others
		}
	}

	return nil
}

// collectWorkload collects a specific workload and its pods by kind
func collectWorkload(ctx context.Context, collector *tarCollector, workloadDir, ns, name string, kind k8sconsts.WorkloadKind, includeLogs bool) error {
	var selector labels.Selector

	switch kind {
	case k8sconsts.WorkloadKindDeployment:
		obj, err := kube.DefaultClient.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := addResourceYAML(collector, workloadDir, "deployment", name, obj); err != nil {
			return err
		}
		selector = labels.SelectorFromSet(obj.Spec.Selector.MatchLabels)

	case k8sconsts.WorkloadKindDaemonSet:
		obj, err := kube.DefaultClient.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := addResourceYAML(collector, workloadDir, "daemonset", name, obj); err != nil {
			return err
		}
		selector = labels.SelectorFromSet(obj.Spec.Selector.MatchLabels)

	case k8sconsts.WorkloadKindStatefulSet:
		obj, err := kube.DefaultClient.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := addResourceYAML(collector, workloadDir, "statefulset", name, obj); err != nil {
			return err
		}
		selector = labels.SelectorFromSet(obj.Spec.Selector.MatchLabels)

	case k8sconsts.WorkloadKindCronJob:
		// CronJobs use batchv1beta1 for k8s < 1.21, batchv1 for >= 1.21
		selector, err := collectCronJob(ctx, collector, workloadDir, ns, name)
		if err != nil {
			return err
		}
		if selector == nil {
			return nil
		}
		return collectPods(ctx, collector, ns, workloadDir, selector, includeLogs)

	default:
		return nil
	}

	return collectPods(ctx, collector, ns, workloadDir, selector, includeLogs)
}

// collectCronJob handles CronJob collection with version detection
func collectCronJob(ctx context.Context, collector *tarCollector, workloadDir, ns, name string) (labels.Selector, error) {
	ver, err := getKubeVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to detect Kubernetes version: %w", err)
	}

	// batchv1beta1 is deprecated in k8s 1.21 and removed in 1.25
	if ver.LessThan(version.MustParseSemantic("1.21.0")) {
		obj, err := kube.DefaultClient.BatchV1beta1().CronJobs(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if err := addResourceYAML(collector, workloadDir, "cronjob", name, obj); err != nil {
			return nil, err
		}
		if obj.Spec.JobTemplate.Spec.Selector != nil {
			return labels.SelectorFromSet(obj.Spec.JobTemplate.Spec.Selector.MatchLabels), nil
		}
		return nil, nil
	}

	obj, err := kube.DefaultClient.BatchV1().CronJobs(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if err := addResourceYAML(collector, workloadDir, "cronjob", name, obj); err != nil {
		return nil, err
	}
	if obj.Spec.JobTemplate.Spec.Selector != nil {
		return labels.SelectorFromSet(obj.Spec.JobTemplate.Spec.Selector.MatchLabels), nil
	}
	return nil, nil
}

// collectPods collects pod YAMLs and optionally logs for pods matching selector
func collectPods(ctx context.Context, collector *tarCollector, ns, componentDir string, selector labels.Selector, includeLogs bool) error {
	pods, err := kube.DefaultClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	for i := range pods.Items {
		pod := &pods.Items[i]
		if err := addResourceYAML(collector, componentDir, "pod", pod.Name, pod); err != nil {
			return err
		}
		if includeLogs {
			collectPodLogs(ctx, collector, ns, componentDir, pod)
		}
	}

	return nil
}

func collectPodLogs(ctx context.Context, collector *tarCollector, ns, componentDir string, pod *corev1.Pod) {
	for _, container := range pod.Spec.Containers {
		// Get current logs
		addContainerLogs(ctx, collector, ns, componentDir, pod.Name, container.Name, false)

		// Check if container has been restarted and get previous logs
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == container.Name && status.RestartCount > 0 {
				addContainerLogs(ctx, collector, ns, componentDir, pod.Name, container.Name, true)
			}
		}
	}

	// Also collect logs from init containers
	for _, container := range pod.Spec.InitContainers {
		addContainerLogs(ctx, collector, ns, componentDir, pod.Name, container.Name, false)
	}
}

func addContainerLogs(ctx context.Context, collector *tarCollector, ns, componentDir, podName, containerName string, previous bool) {
	// Create filename - include container name for clarity
	var filename string
	if previous {
		filename = fmt.Sprintf("pod-%s.%s.previous.logs", podName, containerName)
	} else {
		filename = fmt.Sprintf("pod-%s.%s.logs", podName, containerName)
	}

	req := kube.DefaultClient.CoreV1().Pods(ns).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
		Previous:  previous,
	})

	logStream, err := req.Stream(ctx)
	if err != nil {
		// Write error message to log file so user knows what happened
		errorMsg := fmt.Sprintf("Error fetching logs: %v\n", err)
		_ = collector.addFile(componentDir, filename, []byte(errorMsg))
		return
	}
	defer logStream.Close()

	// Read all logs into memory
	logData, err := io.ReadAll(logStream)
	if err != nil {
		errorMsg := fmt.Sprintf("Error reading logs: %v\n", err)
		_ = collector.addFile(componentDir, filename, []byte(errorMsg))
		return
	}

	// Write logs to tar (even if empty)
	_ = collector.addFile(componentDir, filename, logData)
}

func addResourceYAML(collector *tarCollector, componentDir, resourceType, name string, obj interface{}) error {
	// Clean the object for YAML export
	cleanedObj := cleanObjectForExport(obj)

	yamlData, err := yaml.Marshal(cleanedObj)
	if err != nil {
		return fmt.Errorf("failed to marshal %s %s to YAML: %w", resourceType, name, err)
	}

	filename := fmt.Sprintf("%s-%s.yaml", resourceType, name)
	return collector.addFile(componentDir, filename, yamlData)
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func cleanObjectForExport(obj interface{}) interface{} {
	// Create a copy and clean managed fields and other noisy metadata
	switch v := obj.(type) {
	case *appsv1.Deployment:
		cleaned := v.DeepCopy()
		cleaned.ManagedFields = nil
		return cleaned
	case *appsv1.DaemonSet:
		cleaned := v.DeepCopy()
		cleaned.ManagedFields = nil
		return cleaned
	case *appsv1.StatefulSet:
		cleaned := v.DeepCopy()
		cleaned.ManagedFields = nil
		return cleaned
	case *batchv1.CronJob:
		cleaned := v.DeepCopy()
		cleaned.ManagedFields = nil
		return cleaned
	case *batchv1beta1.CronJob:
		cleaned := v.DeepCopy()
		cleaned.ManagedFields = nil
		return cleaned
	case *corev1.Pod:
		cleaned := v.DeepCopy()
		cleaned.ManagedFields = nil
		return cleaned
	case *corev1.ConfigMap:
		cleaned := v.DeepCopy()
		cleaned.ManagedFields = nil
		return cleaned
	default:
		return obj
	}
}

func (c *tarCollector) addFile(dir, filename string, data []byte) error {
	// Ensure directory entries exist
	dirs := strings.Split(dir, "/")
	currentPath := ""
	for _, d := range dirs {
		if d == "" {
			continue
		}
		if currentPath == "" {
			currentPath = d
		} else {
			currentPath = path.Join(currentPath, d)
		}

		// Skip if directory already created
		if c.createdDirs[currentPath] {
			continue
		}

		// Add directory entry
		header := &tar.Header{
			Name:     currentPath + "/",
			Mode:     0755,
			Typeflag: tar.TypeDir,
		}
		if err := c.tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write directory header: %w", err)
		}
		c.createdDirs[currentPath] = true
	}

	// Add file
	filePath := path.Join(dir, filename)
	header := &tar.Header{
		Name: filePath,
		Mode: 0644,
		Size: int64(len(data)),
	}

	if err := c.tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write file header: %w", err)
	}

	if _, err := c.tarWriter.Write(data); err != nil {
		return fmt.Errorf("failed to write file data: %w", err)
	}

	return nil
}
