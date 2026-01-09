package diagnose

import (
	"context"
	"fmt"
	"path"

	"github.com/odigos-io/odigos/api/k8sconsts"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

// WorkloadTarget represents a workload to collect
type WorkloadTarget struct {
	Namespace   string
	Name        string
	Kind        k8sconsts.WorkloadKind
	DirName     string // folder name (e.g., "deployment-foo")
	IncludeLogs bool
}

// FetchOdigosWorkloads collects workloads from the Odigos namespace (deployments, daemonsets, statefulsets)
// with their pod YAMLs and optionally logs under component folders
func FetchOdigosWorkloads(ctx context.Context, client kubernetes.Interface, collector Collector, rootDir, odigosNamespace string, includeLogs bool) error {
	fmt.Printf("Fetching Odigos Workloads and Logs...\n")
	klog.V(2).InfoS("Fetching Odigos Workloads", "namespace", odigosNamespace)

	var targets []WorkloadTarget

	// Collect Deployments
	deployments, err := client.AppsV1().Deployments(odigosNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.V(1).ErrorS(err, "Failed to list deployments")
	} else {
		for _, d := range deployments.Items {
			targets = append(targets, WorkloadTarget{
				Namespace:   odigosNamespace,
				Name:        d.Name,
				Kind:        k8sconsts.WorkloadKindDeployment,
				DirName:     fmt.Sprintf("deployment-%s", d.Name),
				IncludeLogs: includeLogs,
			})
		}
	}

	// Collect DaemonSets
	daemonsets, err := client.AppsV1().DaemonSets(odigosNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.V(1).ErrorS(err, "Failed to list daemonsets")
	} else {
		for _, d := range daemonsets.Items {
			targets = append(targets, WorkloadTarget{
				Namespace:   odigosNamespace,
				Name:        d.Name,
				Kind:        k8sconsts.WorkloadKindDaemonSet,
				DirName:     fmt.Sprintf("daemonset-%s", d.Name),
				IncludeLogs: includeLogs,
			})
		}
	}

	// Collect StatefulSets
	statefulsets, err := client.AppsV1().StatefulSets(odigosNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.V(1).ErrorS(err, "Failed to list statefulsets")
	} else {
		for _, s := range statefulsets.Items {
			targets = append(targets, WorkloadTarget{
				Namespace:   odigosNamespace,
				Name:        s.Name,
				Kind:        k8sconsts.WorkloadKindStatefulSet,
				DirName:     fmt.Sprintf("statefulset-%s", s.Name),
				IncludeLogs: includeLogs,
			})
		}
	}

	// Collect all targets
	for _, t := range targets {
		workloadDir := path.Join(rootDir, t.Namespace, t.DirName)
		if err := collectWorkload(ctx, client, collector, workloadDir, t.Namespace, t.Name, t.Kind, t.IncludeLogs); err != nil && !apierrors.IsNotFound(err) {
			klog.V(1).ErrorS(err, "Failed to collect workload", "name", t.Name, "kind", t.Kind)
		}
	}

	return nil
}

func collectWorkload(ctx context.Context, client kubernetes.Interface, collector Collector, workloadDir, namespace, name string, kind k8sconsts.WorkloadKind, includeLogs bool) error {
	var selector labels.Selector

	switch kind {
	case k8sconsts.WorkloadKindDeployment:
		obj, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := addWorkloadYAML(collector, workloadDir, "deployment", name, obj); err != nil {
			return err
		}
		selector = labels.SelectorFromSet(obj.Spec.Selector.MatchLabels)

	case k8sconsts.WorkloadKindDaemonSet:
		obj, err := client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := addWorkloadYAML(collector, workloadDir, "daemonset", name, obj); err != nil {
			return err
		}
		selector = labels.SelectorFromSet(obj.Spec.Selector.MatchLabels)

	case k8sconsts.WorkloadKindStatefulSet:
		obj, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := addWorkloadYAML(collector, workloadDir, "statefulset", name, obj); err != nil {
			return err
		}
		selector = labels.SelectorFromSet(obj.Spec.Selector.MatchLabels)

	case k8sconsts.WorkloadKindCronJob:
		obj, err := client.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := addWorkloadYAML(collector, workloadDir, "cronjob", name, obj); err != nil {
			return err
		}
		if obj.Spec.JobTemplate.Spec.Selector != nil {
			selector = labels.SelectorFromSet(obj.Spec.JobTemplate.Spec.Selector.MatchLabels)
		}

	default:
		return nil
	}

	if selector == nil {
		return nil
	}

	// Collect pods
	return collectPods(ctx, client, collector, namespace, workloadDir, selector, includeLogs)
}

func collectPods(ctx context.Context, client kubernetes.Interface, collector Collector, namespace, componentDir string, selector labels.Selector, includeLogs bool) error {
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	for i := range pods.Items {
		pod := &pods.Items[i]
		if err := addPodYAML(collector, componentDir, pod); err != nil {
			klog.V(1).ErrorS(err, "Failed to add pod YAML", "pod", pod.Name)
		}
		if includeLogs {
			if err := FetchWorkloadLogs(ctx, client, collector, namespace, componentDir, []corev1.Pod{*pod}); err != nil {
				klog.V(1).ErrorS(err, "Failed to collect pod logs", "pod", pod.Name)
			}
		}
	}

	return nil
}

func addWorkloadYAML(collector Collector, componentDir, resourceType, name string, obj interface{}) error {
	cleanedObj := cleanObjectForExport(obj)
	yamlData, err := yaml.Marshal(cleanedObj)
	if err != nil {
		return fmt.Errorf("failed to marshal %s %s to YAML: %w", resourceType, name, err)
	}

	filename := fmt.Sprintf("%s-%s.yaml", resourceType, name)
	return collector.AddFile(componentDir, filename, yamlData)
}

func addPodYAML(collector Collector, componentDir string, pod *corev1.Pod) error {
	cleanedPod := pod.DeepCopy()
	cleanedPod.ManagedFields = nil

	yamlData, err := yaml.Marshal(cleanedPod)
	if err != nil {
		return fmt.Errorf("failed to marshal pod %s to YAML: %w", pod.Name, err)
	}

	filename := fmt.Sprintf("pod-%s.yaml", pod.Name)
	return collector.AddFile(componentDir, filename, yamlData)
}

func cleanObjectForExport(obj interface{}) interface{} {
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

// FetchSourceWorkloads collects workloads that are instrumented by Odigos (user's applications)
// It reads Source CRDs to find which workloads are instrumented and collects them
func FetchSourceWorkloads(ctx context.Context, client kubernetes.Interface, dynamicClient dynamic.Interface, collector Collector, rootDir string, namespaceFilter []string, includeLogs bool) error {
	fmt.Printf("Fetching Instrumented Source Workloads...\n")
	klog.V(2).InfoS("Fetching Source Workloads", "namespaceFilter", namespaceFilter)

	// Create a set of allowed namespaces for quick lookup
	allowedNamespaces := make(map[string]bool)
	for _, ns := range namespaceFilter {
		allowedNamespaces[ns] = true
	}
	filterByNamespace := len(allowedNamespaces) > 0

	// List all Source CRDs
	sourceGVR := schema.GroupVersionResource{
		Group:    "odigos.io",
		Version:  "v1alpha1",
		Resource: "sources",
	}

	sourceList, err := dynamicClient.Resource(sourceGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list Source CRDs: %w", err)
	}

	// Track collected workloads to avoid duplicates
	collected := make(map[string]bool)

	for _, item := range sourceList.Items {
		spec, ok := item.Object["spec"].(map[string]interface{})
		if !ok {
			continue
		}

		workload, ok := spec["workload"].(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := workload["name"].(string)
		namespace, _ := workload["namespace"].(string)
		kindStr, _ := workload["kind"].(string)

		// Skip namespace-level sources (they have kind "Namespace")
		if kindStr == "Namespace" || kindStr == "" || name == "" || namespace == "" {
			continue
		}

		// Apply namespace filter if provided
		if filterByNamespace && !allowedNamespaces[namespace] {
			continue
		}

		// Skip duplicates
		key := fmt.Sprintf("%s/%s/%s", namespace, kindStr, name)
		if collected[key] {
			continue
		}
		collected[key] = true

		// Map string kind to WorkloadKind
		var kind k8sconsts.WorkloadKind
		switch kindStr {
		case "Deployment":
			kind = k8sconsts.WorkloadKindDeployment
		case "DaemonSet":
			kind = k8sconsts.WorkloadKindDaemonSet
		case "StatefulSet":
			kind = k8sconsts.WorkloadKindStatefulSet
		case "CronJob":
			kind = k8sconsts.WorkloadKindCronJob
		default:
			klog.V(2).InfoS("Skipping unknown workload kind", "kind", kindStr, "name", name)
			continue
		}

		dirName := fmt.Sprintf("%s-%s", kindStr, name)
		workloadDir := path.Join(rootDir, namespace, dirName)

		if err := collectWorkload(ctx, client, collector, workloadDir, namespace, name, kind, includeLogs); err != nil {
			if !apierrors.IsNotFound(err) {
				klog.V(1).ErrorS(err, "Failed to collect source workload", "namespace", namespace, "name", name, "kind", kindStr)
			}
		}
	}

	klog.V(2).InfoS("Finished collecting source workloads", "count", len(collected))
	return nil
}
