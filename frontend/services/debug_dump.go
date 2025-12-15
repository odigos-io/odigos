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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
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
func DebugDump(c *gin.Context) {
	ctx := c.Request.Context()
	ns := env.GetCurrentNamespace()

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

	// Collect all workloads (deployments, daemonsets, statefulsets)
	if err := collectDeployments(ctx, collector, rootDir, ns); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to collect deployments: %v", err)})
		return
	}

	if err := collectDaemonSets(ctx, collector, rootDir, ns); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to collect daemonsets: %v", err)})
		return
	}

	if err := collectStatefulSets(ctx, collector, rootDir, ns); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to collect statefulsets: %v", err)})
		return
	}

	// Collect Odigos CRDs dynamically
	if err := collectOdigosCRDs(ctx, collector, rootDir, ns); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to collect odigos CRDs: %v", err)})
		return
	}

	c.Status(http.StatusOK)
}

func collectDeployments(ctx context.Context, collector *tarCollector, rootDir, ns string) error {
	deployments, err := kube.DefaultClient.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	for _, deployment := range deployments.Items {
		componentDir := path.Join(rootDir, ns, fmt.Sprintf("deployment-%s", deployment.Name))

		// Add deployment YAML
		if err := addResourceYAML(collector, componentDir, "deployment", deployment.Name, &deployment); err != nil {
			return err
		}

		// Get pods for this deployment
		selector := labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels)
		if err := collectPodsForWorkload(ctx, collector, ns, componentDir, selector); err != nil {
			return err
		}
	}

	return nil
}

func collectDaemonSets(ctx context.Context, collector *tarCollector, rootDir, ns string) error {
	daemonsets, err := kube.DefaultClient.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list daemonsets: %w", err)
	}

	for _, daemonset := range daemonsets.Items {
		componentDir := path.Join(rootDir, ns, fmt.Sprintf("daemonset-%s", daemonset.Name))

		// Add daemonset YAML
		if err := addResourceYAML(collector, componentDir, "daemonset", daemonset.Name, &daemonset); err != nil {
			return err
		}

		// Get pods for this daemonset
		selector := labels.SelectorFromSet(daemonset.Spec.Selector.MatchLabels)
		if err := collectPodsForWorkload(ctx, collector, ns, componentDir, selector); err != nil {
			return err
		}
	}

	return nil
}

func collectStatefulSets(ctx context.Context, collector *tarCollector, rootDir, ns string) error {
	statefulsets, err := kube.DefaultClient.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list statefulsets: %w", err)
	}

	for _, statefulset := range statefulsets.Items {
		componentDir := path.Join(rootDir, ns, fmt.Sprintf("statefulset-%s", statefulset.Name))

		// Add statefulset YAML
		if err := addResourceYAML(collector, componentDir, "statefulset", statefulset.Name, &statefulset); err != nil {
			return err
		}

		// Get pods for this statefulset
		selector := labels.SelectorFromSet(statefulset.Spec.Selector.MatchLabels)
		if err := collectPodsForWorkload(ctx, collector, ns, componentDir, selector); err != nil {
			return err
		}
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
func collectCRD(ctx context.Context, collector *tarCollector, rootDir, ns string, gvr schema.GroupVersionResource) error {
	// Use capitalized resource name as directory (e.g., "destinations" -> "Destinations")
	dirName := capitalizeFirst(gvr.Resource)
	crdDir := path.Join(rootDir, ns, dirName)

	// Try to list from all namespaces first (works for both namespaced and cluster-scoped resources)
	list, err := kube.DefaultClient.DynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		// If all-namespace list fails, try namespace-scoped
		list, err = kube.DefaultClient.DynamicClient.Resource(gvr).Namespace(ns).List(ctx, metav1.ListOptions{})
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

		// Include namespace in filename if the resource is from a different namespace
		var filename string
		itemNs := item.GetNamespace()
		if itemNs != "" && itemNs != ns {
			filename = fmt.Sprintf("%s.%s.yaml", itemNs, item.GetName())
		} else {
			filename = fmt.Sprintf("%s.yaml", item.GetName())
		}

		// Marshal to YAML
		yamlData, err := yaml.Marshal(item.Object)
		if err != nil {
			continue // Skip this item but continue with others
		}

		_ = collector.addFile(crdDir, filename, yamlData)
	}

	return nil
}

func collectPodsForWorkload(ctx context.Context, collector *tarCollector, ns, componentDir string, selector labels.Selector) error {
	pods, err := kube.DefaultClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	for i := range pods.Items {
		pod := &pods.Items[i]

		// Add pod YAML
		if err := addResourceYAML(collector, componentDir, "pod", pod.Name, pod); err != nil {
			return err
		}

		// Add logs for each container in the pod
		collectPodLogs(ctx, collector, ns, componentDir, pod)
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
	case *corev1.Pod:
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
