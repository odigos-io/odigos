package diagnose

import (
	"context"
	"fmt"
	"path"

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

	odigosv1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
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
func FetchOdigosWorkloads(
	ctx context.Context,
	client kubernetes.Interface,
	dynamicClient dynamic.Interface,
	collector Collector,
	rootDir, odigosNamespace string,
	includeLogs bool,
) error {
	fmt.Printf("Fetching Odigos Workloads and Logs...\n")
	klog.V(2).InfoS("Fetching Odigos Workloads", "namespace", odigosNamespace)

	var targets []WorkloadTarget

	// Collect Deployments
	deployments, err := client.AppsV1().Deployments(odigosNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.V(1).ErrorS(err, "Failed to list deployments")
	} else {
		for i := 0; i < len(deployments.Items); i++ {
			d := &deployments.Items[i]
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
		for i := 0; i < len(daemonsets.Items); i++ {
			d := &daemonsets.Items[i]
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
		for i := 0; i < len(statefulsets.Items); i++ {
			s := &statefulsets.Items[i]
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
	for i := 0; i < len(targets); i++ {
		t := &targets[i]
		workloadDir := path.Join(rootDir, t.Namespace, t.DirName)
		err := collectWorkload(ctx, client, dynamicClient, collector, workloadDir, t.Namespace, t.Name, t.Kind, t.IncludeLogs)
		if err != nil && !apierrors.IsNotFound(err) {
			klog.V(1).ErrorS(err, "Failed to collect workload", "name", t.Name, "kind", t.Kind)
		}
	}

	return nil
}

func collectWorkload(
	ctx context.Context,
	client kubernetes.Interface,
	dynamicClient dynamic.Interface,
	collector Collector,
	workloadDir, namespace, name string,
	kind k8sconsts.WorkloadKind,
	includeLogs bool,
) error {
	var selector labels.Selector

	kindLower := string(workload.WorkloadKindLowerCaseFromKind(kind))

	switch kind {
	case k8sconsts.WorkloadKindDeployment:
		obj, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := addWorkloadYAML(collector, workloadDir, kindLower, name, obj); err != nil {
			return err
		}
		selector = labels.SelectorFromSet(obj.Spec.Selector.MatchLabels)

	case k8sconsts.WorkloadKindDaemonSet:
		obj, err := client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := addWorkloadYAML(collector, workloadDir, kindLower, name, obj); err != nil {
			return err
		}
		selector = labels.SelectorFromSet(obj.Spec.Selector.MatchLabels)

	case k8sconsts.WorkloadKindStatefulSet:
		obj, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := addWorkloadYAML(collector, workloadDir, kindLower, name, obj); err != nil {
			return err
		}
		selector = labels.SelectorFromSet(obj.Spec.Selector.MatchLabels)

	case k8sconsts.WorkloadKindCronJob:
		obj, err := client.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := addWorkloadYAML(collector, workloadDir, kindLower, name, obj); err != nil {
			return err
		}
		if obj.Spec.JobTemplate.Spec.Selector != nil {
			selector = labels.SelectorFromSet(obj.Spec.JobTemplate.Spec.Selector.MatchLabels)
		}

	case k8sconsts.WorkloadKindJob:
		obj, err := client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := addWorkloadYAML(collector, workloadDir, kindLower, name, obj); err != nil {
			return err
		}
		if obj.Spec.Selector != nil {
			selector = labels.SelectorFromSet(obj.Spec.Selector.MatchLabels)
		}

	case k8sconsts.WorkloadKindDeploymentConfig:
		// OpenShift DeploymentConfig - use dynamic client
		gvr := schema.GroupVersionResource{
			Group:    "apps.openshift.io",
			Version:  "v1",
			Resource: "deploymentconfigs",
		}
		obj, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		obj.SetManagedFields(nil)
		if err := addWorkloadYAML(collector, workloadDir, kindLower, name, obj.Object); err != nil {
			return err
		}
		// Get selector from spec.selector
		if spec, ok := obj.Object["spec"].(map[string]interface{}); ok {
			if sel, ok := spec["selector"].(map[string]interface{}); ok {
				selectorMap := make(map[string]string)
				for k, v := range sel {
					if vs, ok := v.(string); ok {
						selectorMap[k] = vs
					}
				}
				if len(selectorMap) > 0 {
					selector = labels.SelectorFromSet(selectorMap)
				}
			}
		}

	case k8sconsts.WorkloadKindArgoRollout:
		// Argo Rollout - use dynamic client
		gvr := schema.GroupVersionResource{
			Group:    "argoproj.io",
			Version:  "v1alpha1",
			Resource: "rollouts",
		}
		obj, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		obj.SetManagedFields(nil)
		if err := addWorkloadYAML(collector, workloadDir, kindLower, name, obj.Object); err != nil {
			return err
		}
		// Get selector from spec.selector.matchLabels
		if spec, ok := obj.Object["spec"].(map[string]interface{}); ok {
			if sel, ok := spec["selector"].(map[string]interface{}); ok {
				if matchLabels, ok := sel["matchLabels"].(map[string]interface{}); ok {
					selectorMap := make(map[string]string)
					for k, v := range matchLabels {
						if vs, ok := v.(string); ok {
							selectorMap[k] = vs
						}
					}
					if len(selectorMap) > 0 {
						selector = labels.SelectorFromSet(selectorMap)
					}
				}
			}
		}

	default:
		return workload.ErrKindNotSupported
	}

	if selector == nil {
		return nil
	}

	// Collect pods
	return collectPods(ctx, client, collector, namespace, workloadDir, selector, includeLogs)
}

func collectPods(
	ctx context.Context,
	client kubernetes.Interface,
	collector Collector,
	namespace, componentDir string,
	selector labels.Selector,
	includeLogs bool,
) error {
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
	case *batchv1.Job:
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
func FetchSourceWorkloads(
	ctx context.Context,
	client kubernetes.Interface,
	dynamicClient dynamic.Interface,
	odigosClient odigosv1alpha1.OdigosV1alpha1Interface,
	collector Collector,
	rootDir string,
	namespaceFilter []string,
	includeLogs bool,
) error {
	fmt.Printf("Fetching Instrumented Source Workloads...\n")
	klog.V(2).InfoS("Fetching Source Workloads", "namespaceFilter", namespaceFilter)

	// Create a set of allowed namespaces for quick lookup
	allowedNamespaces := make(map[string]bool)
	for _, ns := range namespaceFilter {
		allowedNamespaces[ns] = true
	}
	filterByNamespace := len(allowedNamespaces) > 0

	// List all Source CRDs using the typed client (empty namespace = all namespaces)
	sourceList, err := odigosClient.Sources("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list Source CRDs: %w", err)
	}

	// Track collected workloads to avoid duplicates
	collected := make(map[string]bool)

	for i := range sourceList.Items {
		source := &sourceList.Items[i]
		wl := source.Spec.Workload

		// Skip invalid entries
		if wl.Kind == "" || wl.Name == "" || wl.Namespace == "" {
			continue
		}

		// Skip namespace-level and static pod sources (not collectable workloads)
		if wl.Kind == k8sconsts.WorkloadKindNamespace || wl.Kind == k8sconsts.WorkloadKindStaticPod {
			continue
		}

		// Skip invalid workload kinds
		if !workload.IsValidWorkloadKind(wl.Kind) {
			klog.V(2).InfoS("Skipping invalid workload kind", "kind", wl.Kind, "name", wl.Name)
			continue
		}

		// Apply namespace filter if provided
		if filterByNamespace && !allowedNamespaces[wl.Namespace] {
			continue
		}

		// Skip duplicates
		key := fmt.Sprintf("%s/%s/%s", wl.Namespace, wl.Kind, wl.Name)
		if collected[key] {
			continue
		}
		collected[key] = true

		kindLower := workload.WorkloadKindLowerCaseFromKind(wl.Kind)
		dirName := fmt.Sprintf("%s-%s", kindLower, wl.Name)
		workloadDir := path.Join(rootDir, wl.Namespace, dirName)

		err := collectWorkload(ctx, client, dynamicClient, collector, workloadDir, wl.Namespace, wl.Name, wl.Kind, includeLogs)
		if err != nil {
			if workload.IsErrorKindNotSupported(err) {
				klog.V(2).InfoS("Workload kind not supported for collection", "kind", wl.Kind, "name", wl.Name)
			} else if !apierrors.IsNotFound(err) {
				klog.V(1).ErrorS(err, "Failed to collect source workload", "namespace", wl.Namespace, "name", wl.Name, "kind", wl.Kind)
			}
		}
	}

	klog.V(2).InfoS("Finished collecting source workloads", "count", len(collected))
	return nil
}
